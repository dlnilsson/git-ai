package claude

import (
	"github.com/dlnilsson/git-cc-ai/pkg/providers"
)

type Backend struct{}

func (Backend) Generate(reg *providers.Registry, opts providers.Options) (string, error) {
	return Generate(reg, opts)
}
