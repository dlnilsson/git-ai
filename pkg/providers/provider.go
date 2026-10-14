package providers

import (
	"context"
	"os"
	"strings"
)

// AssistedByTrailer returns the "Assisted-by:" git trailer for AI-generated
// commits as specified by the Linux kernel coding-assistants guidelines.
// It is enabled by default and can be disabled by setting GIT_AI_ASSISTED_BY=false.
func AssistedByTrailer(agentName, model string) string {
	if os.Getenv("GIT_AI_ASSISTED_BY") == "false" {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nAssisted-by: ")
	b.WriteString(agentName)
	if model != "" {
		b.WriteString(":")
		b.WriteString(model)
	}
	return b.String()
}

// WrapAISummary keeps the subject line as-is and wraps the generated body
// plus the Assisted-by trailer in a collapsible <details> block, leaving
// room for hand-written notes after it.
func WrapAISummary(message, agentName, model string) string {
	var (
		subject, body, _ = strings.Cut(message, "\n")
		trailer          = strings.TrimSpace(AssistedByTrailer(agentName, model))
		inner            = quote(strings.TrimSpace(body))
	)
	if trailer != "" {
		if inner != "" {
			inner += "\n\n"
		}
		inner += trailer
	}
	if inner == "" {
		return subject
	}
	var b strings.Builder
	b.WriteString(subject)
	b.WriteString("\n\n<details>\n<summary>\n\nAI Summary\n----------\n\n</summary>\n\n")
	b.WriteString(inner)
	b.WriteString("\n\n</details>")
	return b.String()
}

// quote prefixes each line with "> " so the text renders as a Markdown
// blockquote; blank lines get a bare ">" to keep a single quote block.
func quote(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l == "" {
			lines[i] = ">"
			continue
		}
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

type Options struct {
	SkillPath   string
	ExtraNote   string
	Model       string
	SessionID   string
	ShowSpinner bool
	NoCC        bool
	DryRun      bool
	Budget      float64 // max spend in USD; 0 means use backend default
}

type Backend interface {
	Generate(ctx context.Context, reg *Registry, opts Options) (string, error)
	Models() []string
	DefaultModel() string
}
