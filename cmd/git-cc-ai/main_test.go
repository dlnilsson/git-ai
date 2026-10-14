package main

import (
	"testing"

	"github.com/dlnilsson/git-cc-ai/pkg/agentrc"
)

func TestSessionIDForBackend(t *testing.T) {
	cfg := agentrc.Config{SessionID: "claude-session"}

	if got := sessionIDForBackend("claude", false, cfg); got != "claude-session" {
		t.Fatalf("claude backend should receive session id, got %q", got)
	}

	for _, backend := range []string{"gemini", "codex"} {
		if got := sessionIDForBackend(backend, false, cfg); got != "" {
			t.Fatalf("%s backend should not receive session id, got %q", backend, got)
		}
	}

	if got := sessionIDForBackend("claude", true, cfg); got != "" {
		t.Fatalf("no-session should disable session id, got %q", got)
	}
}

func TestAssistedByEnabled(t *testing.T) {
	var (
		enabled  = true
		disabled = false
		tests    = []struct {
			name string
			env  string
			rc   *bool
			want bool
		}{
			{name: "default", want: true},
			{name: "file true", rc: &enabled, want: true},
			{name: "file false", rc: &disabled},
			{name: "env true", env: "true", want: true},
			{name: "env false", env: "false"},
			{name: "env true overrides file false", env: "true", rc: &disabled, want: true},
			{name: "env false overrides file true", env: "false", rc: &enabled},
			{name: "invalid env preserves enabled behavior", env: "invalid", rc: &disabled, want: true},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GIT_AI_ASSISTED_BY", tt.env)
			if got := assistedByEnabled(agentrc.Config{AssistedBy: tt.rc}); got != tt.want {
				t.Fatalf("assistedByEnabled() = %t, want %t", got, tt.want)
			}
		})
	}
}
