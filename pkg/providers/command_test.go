package providers

import "testing"

func TestFormatCommand(t *testing.T) {
	got := FormatCommand("codex", []string{"--model", "gpt-5", "--prompt", "line one\nline two", "--label", "O'Reilly"})
	want := "codex --model gpt-5 --prompt 'line one\nline two' --label 'O'\"'\"'Reilly'"
	if got != want {
		t.Fatalf("FormatCommand() = %q, want %q", got, want)
	}
}

func TestFormatCommandWithStdin(t *testing.T) {
	got := FormatCommandWithStdin("line one\nline two", "codex", []string{"exec", "--json"})
	want := "printf '%s' 'line one\nline two' | codex exec --json"
	if got != want {
		t.Fatalf("FormatCommandWithStdin() = %q, want %q", got, want)
	}
}
