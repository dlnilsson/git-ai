package commit

import "strings"

// PromptOptions contains the pieces used to build the commit prompt.
type PromptOptions struct {
	SkillText string
	Diff      string
	ExtraNote string
	NoCC      bool
}

// BuildSystemPrompt returns the stable system-prompt text (instructions +
// skill rules). It is suitable for passing as --system-prompt so that Claude
// can cache it across invocations where only the diff changes.
func BuildSystemPrompt(opts PromptOptions) string {
	var b strings.Builder
	if opts.NoCC {
		b.WriteString("Generate a commit message from the staged git diff.\n")
	} else {
		b.WriteString("Generate a Conventional Commit message from the staged git diff.\n")
	}
	b.WriteString("Use the instructions below and output only the commit message.\n")
	b.WriteString("Limit each line in the commit body to 72 characters; wrap at sentence boundaries (e.g. after a period and space) when possible so lines do not break mid-sentence.\n\n")
	b.WriteString("Instructions:\n")
	b.WriteString(opts.SkillText)
	return b.String()
}

// BuildUserMessage returns the user-facing message text (diff + optional
// extra note). This is the part that changes on every run.
func BuildUserMessage(opts PromptOptions) string {
	var b strings.Builder
	b.WriteString("Staged diff:\n")
	b.WriteString(opts.Diff)
	b.WriteByte('\n')
	if strings.TrimSpace(opts.ExtraNote) != "" {
		b.WriteString("\nExtra context:\n")
		b.WriteString(strings.TrimSpace(opts.ExtraNote))
		b.WriteByte('\n')
	}
	return b.String()
}

// BuildConventionalPrompt builds a backend-agnostic prompt for generating
// a commit message from a staged diff.
func BuildConventionalPrompt(opts PromptOptions) string {
	var prompt strings.Builder

	if opts.NoCC {
		prompt.WriteString("Generate a commit message from the staged git diff.\n")
	} else {
		prompt.WriteString("Generate a Conventional Commit message from the staged git diff.\n")
	}
	prompt.WriteString("Use the instructions below and output only the commit message.\n")
	prompt.WriteString("Limit each line in the commit body to 72 characters; wrap at sentence boundaries (e.g. after a period and space) when possible so lines do not break mid-sentence.\n\n")
	prompt.WriteString("Instructions:\n")
	prompt.WriteString(opts.SkillText)
	prompt.WriteString("\n\n")
	prompt.WriteString("Staged diff:\n")
	prompt.WriteString(opts.Diff)
	prompt.WriteString("\n")
	if strings.TrimSpace(opts.ExtraNote) != "" {
		prompt.WriteString("\nExtra context:\n")
		prompt.WriteString(strings.TrimSpace(opts.ExtraNote))
		prompt.WriteString("\n")
	}

	return prompt.String()
}
