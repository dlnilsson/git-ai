package providers

import "context"

type Options struct {
	SkillPath   string
	ExtraNote   string
	Model       string
	SessionID   string
	ShowSpinner bool
	NoCC        bool
	Budget      float64 // max spend in USD; 0 means use backend default
}

type Backend interface {
	Generate(ctx context.Context, reg *Registry, opts Options) (string, error)
}
