package codex

import (
	"context"

	"github.com/dlnilsson/git-cc-ai/pkg/providers"
)

type Backend struct{}

func (Backend) Generate(ctx context.Context, reg *providers.Registry, opts providers.Options) (string, error) {
	return Generate(ctx, reg, opts)
}
