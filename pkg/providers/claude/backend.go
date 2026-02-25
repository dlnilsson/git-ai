package claude

import (
	"context"

	"github.com/dlnilsson/git-cc-ai/pkg/providers"
)

type Backend struct{}

func (Backend) Generate(ctx context.Context, reg *providers.Registry, opts providers.Options) (string, error) {
	return Generate(ctx, reg, opts)
}

func (Backend) Models() []string     { return append([]string{}, allowedModels...) }
func (Backend) DefaultModel() string { return defaultModel }
