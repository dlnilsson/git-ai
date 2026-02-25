package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/dlnilsson/git-cc-ai/pkg/commit"
	"github.com/dlnilsson/git-cc-ai/pkg/git"
	"github.com/dlnilsson/git-cc-ai/pkg/providers"
	"github.com/dlnilsson/git-cc-ai/pkg/ui"
)

const defaultModel = "gemini-2.5-flash"

var models = []string{
	"gemini-2.5-pro",
	"gemini-2.5-flash",
}

func Models() []string {
	return append([]string{}, models...)
}

func IsModelSupported(name string) bool {
	return modelInList(name, models)
}

func resolveModel(model string) string {
	model = strings.TrimSpace(model)
	if modelInList(model, models) {
		return model
	}
	return defaultModel
}

func Generate(ctx context.Context, reg *providers.Registry, opts providers.Options) (string, error) {
	diff, err := git.DiffStaged()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("no staged diff content found")
	}

	skillText := commit.ConventionalSpec
	if opts.NoCC {
		skillText = commit.StandardCommitRule
	}
	if opts.SkillPath != "" {
		if data, readErr := os.ReadFile(opts.SkillPath); readErr == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				skillText = skillText + "\nAdditional instructions:\n" + trimmed
			}
		}
	}

	prompt := commit.BuildConventionalPrompt(commit.PromptOptions{
		SkillText: skillText,
		Diff:      diff,
		ExtraNote: opts.ExtraNote,
		NoCC:      opts.NoCC,
	})
	model := resolveModel(opts.Model)

	args := []string{
		"--prompt", prompt,
		"--output-format", "stream-json",
	}
	args = append(args, "--model", model)
	if strings.TrimSpace(opts.SessionID) != "" {
		args = append(args, "--resume", opts.SessionID)
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)
	setProcessGroup(cmd)

	startTime := time.Now()
	var stopSpinner func()
	if opts.ShowSpinner {
		backendLabel := "gemini +" + model
		stopSpinner = ui.StartSpinner(ui.RandomSpinnerMessage(), backendLabel, reg)
		defer stopSpinner()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err = cmd.Start(); err != nil {
		return "", fmt.Errorf("gemini invocation failed: %w", err)
	}
	reg.Register(cmd, stopSpinner)
	defer reg.Unregister()

	var (
		accumulatedContent strings.Builder
		stdoutBuf          strings.Builder
		sessionID          string
		stats              geminiStats
		status             string
	)

	reader := bufio.NewReader(io.TeeReader(stdout, &stdoutBuf))
	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return "", readErr
			}
			continue
		}
		parsed := parseGeminiEvent(raw)
		if parsed.SessionID != "" {
			sessionID = parsed.SessionID
		}
		if parsed.Status != "" {
			status = parsed.Status
		}
		if parsed.Stats != (geminiStats{}) {
			stats = parsed.Stats
		}
		if parsed.Role == "assistant" && parsed.Content != "" {
			accumulatedContent.WriteString(parsed.Content)
			if opts.ShowSpinner {
				ui.SendSpinnerReasoning(strings.TrimSpace(accumulatedContent.String()))
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if err = cmd.Wait(); err != nil {
		if reg.WasInterrupted() {
			return "", errors.New("gemini invocation interrupted")
		}
		if stderrBuf.Len() > 0 {
			return "", fmt.Errorf("gemini invocation failed: %w\n%s", err, strings.TrimSpace(stderrBuf.String()))
		}
		return "", fmt.Errorf("gemini invocation failed: %w", err)
	}

	if status == "error" {
		return "", errors.New("gemini returned an error")
	}

	responseText := accumulatedContent.String()
	text := commit.StripCodeFence(strings.TrimSpace(responseText))
	if text == "" {
		return "", errors.New("gemini returned empty response")
	}

	msg := commit.WrapMessage(text, commit.BodyLineWidth)
	return appendUsageComment(msg, sessionID, stats, time.Since(startTime), model), nil
}

type geminiEvent struct {
	SessionID string
	Role      string
	Content   string
	Status    string
	Stats     geminiStats
}

type geminiStats struct {
	TotalTokens  int `json:"total_tokens"`
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	DurationMS   int `json:"duration_ms"`
}

func parseGeminiEvent(raw map[string]any) geminiEvent {
	ev := geminiEvent{
		SessionID: asString(raw["session_id"]),
		Role:      asString(raw["role"]),
		Status:    asString(raw["status"]),
	}
	if asString(raw["type"]) == "result" {
		if stats, ok := raw["stats"].(map[string]any); ok {
			ev.Stats = parseGeminiStats(stats)
		}
	}
	if asString(raw["type"]) == "message" {
		ev.Content = asString(raw["content"])
	}
	return ev
}

func parseGeminiStats(raw map[string]any) geminiStats {
	return geminiStats{
		TotalTokens:  toInt(raw["total_tokens"]),
		InputTokens:  toInt(raw["input_tokens"]),
		OutputTokens: toInt(raw["output_tokens"]),
		DurationMS:   toInt(raw["duration_ms"]),
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func modelInList(name string, list []string) bool {
	return slices.Contains(list, name)
}

func appendUsageComment(message string, sessionID string, stats geminiStats, elapsed time.Duration, model string) string {
	elapsedText := elapsed.Round(100 * time.Millisecond)

	var b strings.Builder
	b.WriteString(message)
	b.WriteString("\n\n# tokens: input=")
	b.WriteString(fmt.Sprint(stats.InputTokens))
	b.WriteString(" output=")
	b.WriteString(fmt.Sprint(stats.OutputTokens))
	b.WriteString(" elapsed=")
	b.WriteString(elapsedText.String())

	if sessionID != "" {
		b.WriteString("\n# session=")
		b.WriteString(sessionID)
	}
	if model != "" {
		b.WriteString(" model=")
		b.WriteString(model)
	}

	return b.String()
}
