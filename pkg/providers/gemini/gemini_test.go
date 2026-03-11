package gemini

import (
	"slices"
	"testing"
)

func TestBuildArgsDoesNotIncludeResume(t *testing.T) {
	args := buildArgs("diff", "gemini-2.5-flash")

	if slices.Contains(args, "--resume") {
		t.Fatalf("gemini args should not include --resume: %v", args)
	}
}
