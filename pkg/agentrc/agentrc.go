package agentrc

import (
	"os"
	"strings"
)

// Config holds values parsed from a .agentrc file.
type Config struct {
	SessionID string
	Backend   string
}

// Load reads a .agentrc file and returns its parsed configuration.
// Returns a zero Config (no error) if the file does not exist.
func Load(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}
	var cfg Config
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "export CLAUDE_SESSION_ID="); ok {
			cfg.SessionID = strings.TrimSpace(after)
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_BACKEND="); ok {
			cfg.Backend = strings.TrimSpace(after)
		}
	}
	return cfg
}
