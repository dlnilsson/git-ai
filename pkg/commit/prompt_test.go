package commit

import (
	"strings"
	"testing"
)

func TestBuildConventionalPrompt(t *testing.T) {
	t.Parallel()

	out := BuildConventionalPrompt(PromptOptions{
		SkillText: "rules",
		Diff:      "diff --git a b",
		ExtraNote: "  this is context  ",
	})

	if !strings.Contains(out, "Instructions:\nrules\n\nStaged diff:\ndiff --git a b\n") {
		t.Fatalf("prompt missing required sections: %q", out)
	}
	if !strings.Contains(out, "\nExtra context:\nthis is context\n") {
		t.Fatalf("prompt missing trimmed extra context: %q", out)
	}
}

func TestBuildConventionalPromptWithoutExtraContext(t *testing.T) {
	t.Parallel()

	out := BuildConventionalPrompt(PromptOptions{
		SkillText: "rules",
		Diff:      "diff --git a b",
		ExtraNote: "   ",
	})

	if strings.Contains(out, "Extra context:") {
		t.Fatalf("prompt should not include extra context section: %q", out)
	}
}
