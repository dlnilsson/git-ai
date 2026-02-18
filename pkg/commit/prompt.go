package commit

import "strings"

// PromptOptions contains the pieces used to build the commit prompt.
type PromptOptions struct {
	SkillText string
	Diff      string
	ExtraNote string
	NoCC      bool
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
