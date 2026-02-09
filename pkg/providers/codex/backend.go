package codex

import (
	"errors"

	"github.com/dlnilsson/git-cc-ai/pkg/providers"
)

type Backend struct{}

func (Backend) Generate(reg providers.Registry, opts providers.Options) (string, error) {
	registry, ok := reg.(*Registry)
	if !ok {
		return "", errors.New("invalid registry type")
	}
	return Generate(registry, opts)
}
