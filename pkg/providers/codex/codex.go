package codex

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dlnilsson/git-cc-ai/pkg/commit"
	"github.com/dlnilsson/git-cc-ai/pkg/git"
	"github.com/dlnilsson/git-cc-ai/pkg/providers"
	"github.com/dlnilsson/git-cc-ai/pkg/ui"
)

type threadTracker struct {
	mu       sync.Mutex
	threadID string
}

func (t *threadTracker) set(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.threadID == "" && strings.TrimSpace(id) != "" {
		t.threadID = strings.TrimSpace(id)
	}
}

func (t *threadTracker) get() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.threadID
}

type codexUsage struct {
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
}

func Models() []string {
	return append([]string{}, models...)
}

func IsModelSupported(name string) bool {
	return modelInList(name, models)
}

// https://developers.openai.com/codex/models/
var models = []string{
	"gpt-5.1-codex-max",
	"gpt-5.1-codex-mini",
	"gpt-5.2-codex",
	"gpt-5.3-codex",
	"gpt-5.3-codex-spark",
	"gpt-5-codex-mini",
}

func Generate(reg *providers.Registry, opts providers.Options) (string, error) {
	const (
		codexCmd  = "codex"
		codexArgs = "exec --json"
	)
	var (
		args          []string
		buffer        strings.Builder
		cmd           *exec.Cmd
		diff          string
		err           error
		lastError     string
		output        string
		reasoningText string
		scanner       *bufio.Scanner
		skillText     string
		stderr        io.ReadCloser
		stdout        io.ReadCloser
		stopSpinner   func()
		usage         codexUsage
		startTime     time.Time
	)

	diff, err = git.DiffStaged()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("no staged diff content found")
	}

	skillText = commit.ConventionalSpec
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
	})

	args = splitArgs(codexArgs)
	args = addNoAltScreenArg(args)
	if strings.TrimSpace(opts.Model) != "" {
		args = addModelArg(args, opts.Model)
	}
	cmd = exec.Command(codexCmd, args...)
	cmd.Stdin = strings.NewReader(prompt)
	setProcessGroup(cmd)
	startTime = time.Now()
	if opts.ShowSpinner {
		backendLabel := "codex"
		if strings.TrimSpace(opts.Model) != "" {
			backendLabel += " +" + opts.Model
		}
		stopSpinner = ui.StartSpinner(ui.RandomSpinnerMessage(), backendLabel, reg)
		defer stopSpinner()
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err = cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}
	reg.Register(cmd, stopSpinner)
	defer reg.Unregister()

	var (
		thread    threadTracker
		stderrBuf strings.Builder
	)
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			stderrBuf.WriteString(line)
			stderrBuf.WriteByte('\n')
			if id := parseThreadStartedJSON(line); id != "" {
				thread.set(id)
			}
		}
	}()

	scanner = bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteByte('\n')
		if id := parseThreadStartedJSON(line); id != "" {
			thread.set(id)
		}
		if opts.ShowSpinner {
			reasoningText = parseReasoningJSON(line)
			if strings.TrimSpace(reasoningText) != "" {
				ui.SendSpinnerReasoning(reasoningText)
			}
		}
		if updated, ok := parseUsageJSON(line); ok {
			usage = updated
		}
		if errMsg := parseErrorJSON(line); errMsg != "" {
			lastError = errMsg
		}
	}
	if err = scanner.Err(); err != nil {
		return "", err
	}
	if err = cmd.Wait(); err != nil {
		if lastError != "" {
			return "", fmt.Errorf("codex invocation failed: %s", lastError)
		}
		if errText := strings.TrimSpace(stderrBuf.String()); errText != "" {
			return "", fmt.Errorf("codex invocation failed: %w\n%s", err, errText)
		}
		return "", fmt.Errorf("codex invocation failed: %w", err)
	}
	if reg.WasInterrupted() {
		if id := thread.get(); id != "" {
			fmt.Fprintln(os.Stderr, id)
		}
		return "", errors.New("codex invocation failed")
	}

	output = strings.TrimSpace(buffer.String())
	if output == "" {
		return "", nil
	}

	if parsed := parseCodexJSON(output); strings.TrimSpace(parsed) != "" {
		text := commit.StripCodeFence(strings.TrimSpace(parsed))
		return appendUsageComment(commit.WrapMessage(text, commit.BodyLineWidth), usage, time.Since(startTime), opts.Model), nil
	}

	if strings.HasPrefix(output, "{") {
		if extracted := extractJSONField(output, []string{"output", "stdout", "result", "message"}); strings.TrimSpace(extracted) != "" {
			text := commit.StripCodeFence(strings.TrimSpace(extracted))
			return appendUsageComment(commit.WrapMessage(text, commit.BodyLineWidth), usage, time.Since(startTime), opts.Model), nil
		}
	}

	return appendUsageComment(commit.WrapMessage(commit.StripCodeFence(output), commit.BodyLineWidth), usage, time.Since(startTime), opts.Model), nil
}

func parseErrorJSON(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if t, ok := msg["type"].(string); !ok || t != "error" {
		return ""
	}
	text, _ := msg["message"].(string)
	// The message may itself be JSON with a "detail" field.
	var detail struct{ Detail string }
	if json.Unmarshal([]byte(text), &detail) == nil && detail.Detail != "" {
		return detail.Detail
	}
	return text
}

func parseReasoningJSON(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if t, ok := msg["type"].(string); ok {
		if t == "reasoning" {
			if text, ok := msg["text"].(string); ok {
				return text
			}
		}
		if t == "item.completed" {
			if item, ok := msg["item"].(map[string]any); ok {
				if it, ok := item["type"].(string); ok && it == "reasoning" {
					if text, ok := item["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func parseCodexJSON(raw string) string {
	var (
		lines = strings.Split(raw, "\n")
		last  = ""
	)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if t, ok := msg["type"].(string); ok {
			if t == "agent_message" {
				if text, ok := msg["text"].(string); ok && strings.TrimSpace(text) != "" {
					last = text
				}
				continue
			}
			if t == "item.completed" {
				if item, ok := msg["item"].(map[string]any); ok {
					if it, ok := item["type"].(string); ok && it == "agent_message" {
						if text, ok := item["text"].(string); ok && strings.TrimSpace(text) != "" {
							last = text
						}
					}
				}
			}
		}
	}
	return last
}

func parseThreadStartedJSON(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if t, ok := msg["type"].(string); !ok || t != "thread.started" {
		return ""
	}
	if id, ok := msg["thread_id"].(string); ok {
		return id
	}
	return ""
}

func parseUsageJSON(raw string) (codexUsage, bool) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return codexUsage{}, false
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return codexUsage{}, false
	}
	if t, ok := msg["type"].(string); !ok || t != "turn.completed" {
		return codexUsage{}, false
	}
	usage, ok := msg["usage"].(map[string]any)
	if !ok {
		return codexUsage{}, false
	}
	return codexUsage{
		InputTokens:       toInt(usage["input_tokens"]),
		CachedInputTokens: toInt(usage["cached_input_tokens"]),
		OutputTokens:      toInt(usage["output_tokens"]),
	}, true
}

func appendUsageComment(message string, usage codexUsage, elapsed time.Duration, model string) string {
	if usage == (codexUsage{}) {
		return message
	}
	elapsedText := elapsed.Round(100 * time.Millisecond)
	comment := message + "\n\n# tokens: input=" + fmt.Sprint(usage.InputTokens) +
		" cached=" + fmt.Sprint(usage.CachedInputTokens) +
		" output=" + fmt.Sprint(usage.OutputTokens) +
		" elapsed=" + elapsedText.String()
	if strings.TrimSpace(model) != "" {
		comment = comment + " model=" + model
	}
	return comment
}

func splitArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	return strings.Fields(raw)
}

func addModelArg(args []string, model string) []string {
	if len(args) == 0 {
		return []string{"-m", model}
	}
	out := make([]string, 0, len(args)+2)
	if execIdx := slices.Index(args, "exec"); execIdx != -1 {
		out = append(out, args[:execIdx+1]...)
		out = append(out, "-m", model)
		out = append(out, args[execIdx+1:]...)
		return out
	}
	out = append(out, args...)
	out = append(out, "-m", model)
	return out
}

func addNoAltScreenArg(args []string) []string {
	if len(args) == 0 {
		return []string{"--no-alt-screen"}
	}
	if execIdx := slices.Index(args, "exec"); execIdx != -1 {
		out := make([]string, 0, len(args)+1)
		out = append(out, args[:execIdx]...)
		out = append(out, "--no-alt-screen")
		out = append(out, args[execIdx:]...)
		return out
	}
	return append(args, "--no-alt-screen")
}

func modelInList(name string, list []string) bool {
	return slices.Contains(list, name)
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func extractJSONField(raw string, keys []string) string {
	for _, key := range keys {
		var (
			needle       = `"` + key + `":`
			_, after, ok = strings.Cut(raw, needle)
		)
		if !ok {
			continue
		}
		rest := after
		rest = strings.TrimLeft(rest, " \n\r\t")
		if strings.HasPrefix(rest, "\"") {
			rest = rest[1:]
			out := strings.Builder{}
			escaped := false
			for _, r := range rest {
				if escaped {
					out.WriteRune(r)
					escaped = false
					continue
				}
				if r == '\\' {
					escaped = true
					continue
				}
				if r == '"' {
					return out.String()
				}
				out.WriteRune(r)
			}
		}
	}
	return ""
}
