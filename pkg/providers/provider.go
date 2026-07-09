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
