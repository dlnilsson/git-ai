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
