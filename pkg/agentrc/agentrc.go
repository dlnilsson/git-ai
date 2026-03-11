package agentrc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dlnilsson/git-cc-ai/pkg/git"
)

// Config holds values parsed from a .agentrc file.
type Config struct {
	SessionID string
	Backend   string
	Model     string
	NoCC      bool
	NoSession bool
	Budget    float64 // GIT_AI_BUDGET — max spend in USD (0 means unset)
}

// FindAndLoad looks for .agentrc first in the current worktree root, then
// falls back to the main repository root. Returns a zero Config if neither
// location has a .agentrc or if the working directory is not inside a git
// repository.
func FindAndLoad() Config {
	// Try the current worktree root first.
	if root, err := git.TopLevel(); err == nil {
		if cfg := Load(filepath.Join(root, ".agentrc")); cfg != (Config{}) {
			return cfg
		}
	}

	// Fall back to the main repository root (parent of the shared .git dir).
	if root, err := git.CommonRoot(); err == nil {
		return Load(filepath.Join(root, ".agentrc"))
	}

	return Config{}
}

// Load reads a .agentrc file and returns its parsed configuration.
// Returns a zero Config (no error) if the file does not exist.
func Load(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}
	var cfg Config
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := cutEnvValue(line, "CLAUDE_SESSION_ID"); ok {
			cfg.SessionID = strings.TrimSpace(after)
		}
		if after, ok := cutEnvValue(line, "GIT_AI_BACKEND"); ok {
			cfg.Backend = strings.TrimSpace(after)
		}
		if after, ok := cutEnvValue(line, "GIT_AI_MODEL"); ok {
			cfg.Model = strings.TrimSpace(after)
		}
		if after, ok := cutEnvValue(line, "GIT_AI_NO_CC"); ok {
			cfg.NoCC = strings.EqualFold(strings.TrimSpace(after), "true")
		}
		if after, ok := cutEnvValue(line, "GIT_AI_NO_SESSION"); ok {
			cfg.NoSession = strings.EqualFold(strings.TrimSpace(after), "true")
		}
		if after, ok := cutEnvValue(line, "GIT_AI_BUDGET"); ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(after), 64); err == nil && v > 0 {
				cfg.Budget = v
			}
		}
	}
	return cfg
}

func cutEnvValue(line, key string) (string, bool) {
	if afterExport, ok := strings.CutPrefix(line, "export "); ok {
		line = strings.TrimSpace(afterExport)
	}
	return strings.CutPrefix(line, key+"=")
}
