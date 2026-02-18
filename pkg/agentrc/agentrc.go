package agentrc

import (
	"os"
	"strconv"
	"strings"
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
		if after, ok := strings.CutPrefix(line, "export CLAUDE_SESSION_ID="); ok {
			cfg.SessionID = strings.TrimSpace(after)
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_BACKEND="); ok {
			cfg.Backend = strings.TrimSpace(after)
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_MODEL="); ok {
			cfg.Model = strings.TrimSpace(after)
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_NO_CC="); ok {
			cfg.NoCC = strings.EqualFold(strings.TrimSpace(after), "true")
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_NO_SESSION="); ok {
			cfg.NoSession = strings.EqualFold(strings.TrimSpace(after), "true")
		}
		if after, ok := strings.CutPrefix(line, "export GIT_AI_BUDGET="); ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(after), 64); err == nil && v > 0 {
				cfg.Budget = v
			}
		}
	}
	return cfg
}
