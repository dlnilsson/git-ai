package claude

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dlnilsson/git-cc-ai/pkg/commit"
	"github.com/dlnilsson/git-cc-ai/pkg/git"
	"github.com/dlnilsson/git-cc-ai/pkg/providers"
	"github.com/dlnilsson/git-cc-ai/pkg/ui"
)

func Generate(reg *providers.Registry, opts providers.Options) (string, error) {
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
	skillText = skillText + "\n\n" + "Dont sign commit messages with claude code!"
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

	args := []string{
		"--output-format=stream-json", "--verbose", "--include-partial-messages",
		"--no-session-persistence",
		"--max-budget-usd", "1",
	}
	if opts.SessionID != "" {
		args = append([]string{"--resume=" + opts.SessionID, "--fork-session"}, args...)
	}
	cmd := exec.Command("claude", args...)
	cmd.Stdin = strings.NewReader(prompt)
	setProcessGroup(cmd)

	startTime := time.Now()
	var stopSpinner func()
	if opts.ShowSpinner {
		stopSpinner = ui.StartSpinner(ui.RandomSpinnerMessage(), "claude", reg)
		defer stopSpinner()
		if opts.SessionID != "" {
			ui.SendSpinnerReasoning("Resuming session " + opts.SessionID)
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		return "", err
	}
	reg.Register(cmd, stopSpinner)
	defer reg.Unregister()

	var (
		result        claudeResult
		lastAssistant string
		deltaAccum    strings.Builder
	)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		if opts.ShowSpinner {
			if delta := parseTextDelta(line); delta != "" {
				deltaAccum.WriteString(delta)
				ui.SendSpinnerReasoning(strings.TrimSpace(deltaAccum.String()))
			} else if text := parseStreamReasoning(line); text != "" {
				deltaAccum.Reset()
				ui.SendSpinnerReasoning(text)
			}
		}

		if text := parseAssistantText(line); text != "" {
			lastAssistant = text
		}
		if r, ok := parseResultEvent(line); ok {
			result = r
		}
	}
	if err = scanner.Err(); err != nil {
		return "", err
	}
	if err = cmd.Wait(); err != nil {
		if reg.WasInterrupted() {
			return "", errors.New("claude invocation interrupted")
		}
		return "", errors.New("claude invocation failed")
	}

	responseText := result.Result
	if responseText == "" && strings.HasPrefix(result.Subtype, "error_") {
		fmt.Fprintf(os.Stderr, "claude: %s\n", result.Subtype)
		responseText = lastAssistant
	}

	text := commit.StripCodeFence(strings.TrimSpace(responseText))
	if text == "" {
		if result.Subtype != "" {
			return "", fmt.Errorf("claude: %s", result.Subtype)
		}
		return "", errors.New("claude returned empty response")
	}

	msg := commit.WrapMessage(text, commit.BodyLineWidth)
	return appendUsageComment(msg, result, time.Since(startTime)), nil
}

// parseStreamReasoning extracts displayable reasoning text from assistant
// message events: tool_use descriptions/commands and text content.
// Prefers tool_use info over text within the same message.
func parseStreamReasoning(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if msg["type"] != "assistant" {
		return ""
	}
	message, _ := msg["message"].(map[string]any)
	if message == nil {
		return ""
	}
	content, _ := message["content"].([]any)
	var toolText string
	for _, c := range content {
		block, _ := c.(map[string]any)
		if block == nil {
			continue
		}
		switch block["type"] {
		case "tool_use":
			input, _ := block["input"].(map[string]any)
			if input == nil {
				continue
			}
			desc, _ := input["description"].(string)
			cmd, _ := input["command"].(string)
			switch {
			case desc != "" && cmd != "":
				toolText = desc + ": " + cmd
			case desc != "":
				toolText = desc
			case cmd != "":
				toolText = cmd
			}
		case "text":
			if toolText == "" {
				if text, _ := block["text"].(string); strings.TrimSpace(text) != "" {
					toolText = strings.TrimSpace(text)
				}
			}
		}
	}
	return toolText
}

// parseAssistantText extracts content[0].text from type "assistant" events.
func parseAssistantText(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if msg.Type != "assistant" || len(msg.Message.Content) == 0 {
		return ""
	}
	if msg.Message.Content[0].Type != "text" {
		return ""
	}
	return msg.Message.Content[0].Text
}

// parseTextDelta extracts text from stream_event content_block_delta
// text_delta events.
func parseTextDelta(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var ev struct {
		Type  string `json:"type"`
		Event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		} `json:"event"`
	}
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return ""
	}
	if ev.Type != "stream_event" || ev.Event.Type != "content_block_delta" || ev.Event.Delta.Type != "text_delta" {
		return ""
	}
	return ev.Event.Delta.Text
}

// parseResultEvent parses the final "result" event from stream-json output.
func parseResultEvent(raw string) (claudeResult, bool) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return claudeResult{}, false
	}
	if !strings.Contains(line, `"type":"result"`) && !strings.Contains(line, `"type": "result"`) {
		return claudeResult{}, false
	}
	var result claudeResult
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		return claudeResult{}, false
	}
	if result.Type != "result" {
		return claudeResult{}, false
	}
	return result, true
}

type claudeResult struct {
	Type         string                      `json:"type"`
	Subtype      string                      `json:"subtype"`
	Result       string                      `json:"result"`
	TotalCostUSD float64                     `json:"total_cost_usd"`
	DurationMS   int                         `json:"duration_ms"`
	DurationAPI  int                         `json:"duration_api_ms"`
	IsError      bool                        `json:"is_error"`
	NumTurns     int                         `json:"num_turns"`
	SessionID    string                      `json:"session_id"`
	Usage        claudeUsage                 `json:"usage"`
	ModelUsage   map[string]claudeModelUsage `json:"modelUsage"`
}

type claudeUsage struct {
	InputTokens              int           `json:"input_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	ServerToolUse            serverToolUse `json:"server_tool_use"`
}

type serverToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
}

type claudeModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
}

func appendUsageComment(message string, cr claudeResult, elapsed time.Duration) string {
	if cr.SessionID == "" && cr.TotalCostUSD == 0 {
		return message
	}

	elapsedText := elapsed.Round(100 * time.Millisecond)

	var b strings.Builder
	b.WriteString(message)
	b.WriteString("\n\n# cost=$")
	b.WriteString(fmt.Sprintf("%.4f", cr.TotalCostUSD))
	b.WriteString(" elapsed=")
	b.WriteString(elapsedText.String())
	b.WriteString("\n# session=")
	b.WriteString(cr.SessionID)

	for model, mu := range cr.ModelUsage {
		b.WriteString("\n# model=")
		b.WriteString(model)
		b.WriteString(" input=")
		b.WriteString(fmt.Sprint(mu.InputTokens))
		b.WriteString(" output=")
		b.WriteString(fmt.Sprint(mu.OutputTokens))
		b.WriteString(" cache_read=")
		b.WriteString(fmt.Sprint(mu.CacheReadInputTokens))
		b.WriteString(" cache_create=")
		b.WriteString(fmt.Sprint(mu.CacheCreationInputTokens))
		if mu.WebSearchRequests > 0 {
			b.WriteString(" web_searches=")
			b.WriteString(fmt.Sprint(mu.WebSearchRequests))
		}
	}

	return b.String()
}
