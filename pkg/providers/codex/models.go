package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"sort"
	"time"
)

const modelsTimeout = 5 * time.Second

type codexModel struct {
	Slug       string `json:"slug"`
	Visibility string `json:"visibility"`
	Priority   int    `json:"priority"`
}

// listModels asks the codex CLI for its model catalog. It tries the
// refreshed catalog first, then the one bundled with the binary, and falls
// back to the default model when both fail.
func listModels(ctx context.Context) []string {
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	for _, args := range [][]string{
		{"debug", "models"},
		{"debug", "models", "--bundled"},
	} {
		out, err := exec.CommandContext(ctx, "codex", args...).Output()
		if err != nil {
			continue
		}
		if slugs, err := parseModels(out); err == nil {
			return slugs
		}
	}
	return []string{defaultModel}
}

// parseModels returns the slugs of user-visible models, ordered by priority.
func parseModels(data []byte) ([]string, error) {
	var catalog struct {
		Models []codexModel `json:"models"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	visible := make([]codexModel, 0, len(catalog.Models))
	for _, m := range catalog.Models {
		if m.Visibility == "list" && m.Slug != "" {
			visible = append(visible, m)
		}
	}
	if len(visible) == 0 {
		return nil, errors.New("no visible codex models")
	}
	sort.SliceStable(visible, func(i, j int) bool { return visible[i].Priority < visible[j].Priority })
	slugs := make([]string, 0, len(visible))
	for _, m := range visible {
		slugs = append(slugs, m.Slug)
	}
	return slugs, nil
}
