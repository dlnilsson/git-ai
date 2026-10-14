package providers

import (
	"context"
	"strings"
)

// AssistedByTrailer returns the "Assisted-by:" git trailer for AI-generated
// commits as specified by the Linux kernel coding-assistants guidelines.
func AssistedByTrailer(agentName, model string, noAssistedBy bool) string {
	if noAssistedBy {
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

// WrapAISummary keeps the subject line as-is and wraps the subject, the
// generated body and the Assisted-by trailer in a collapsible <details>
// block, leaving room for hand-written notes after it. A message without a
// body is returned as the bare subject.
func WrapAISummary(message, agentName, model string, noAssistedBy bool) string {
	subject, body, _ := strings.Cut(message, "\n")
	if body = strings.TrimSpace(body); body == "" {
		return subject
	}
	var (
		trailer = strings.TrimSpace(AssistedByTrailer(agentName, model, noAssistedBy))
		inner   = quote(subject + "\n\n" + body)
	)
	if trailer != "" {
		inner += "\n\n" + trailer
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
	SkillPath    string
	ExtraNote    string
	Model        string
	SessionID    string
	ShowSpinner  bool
	NoCC         bool
	NoAssistedBy bool
	DryRun       bool
	Budget       float64 // max spend in USD; 0 means use backend default
}

type Backend interface {
	Generate(ctx context.Context, reg *Registry, opts Options) (string, error)
	Models() []string
	DefaultModel() string
}
