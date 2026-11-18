package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	modelsTimeout  = 5 * time.Second
	defaultBaseURL = "https://api.anthropic.com"
)

// modelAliases are resolved by the claude CLI to the latest model of each
// family, so they never go stale.
var modelAliases = []string{"haiku", "sonnet", "opus", "fable"}

// listModels returns the CLI aliases, followed by the model IDs from the
// Anthropic models API when ANTHROPIC_API_KEY is set. API failures are
// ignored and only the aliases are returned.
func listModels(ctx context.Context) []string {
	models := append([]string{}, modelAliases...)
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return models
	}
	baseURL := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	ids, err := fetchModelIDs(ctx, strings.TrimRight(baseURL, "/"), apiKey)
	if err != nil {
		return models
	}
	return append(models, ids...)
}

func fetchModelIDs(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models?limit=1000", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models: %s", resp.Status)
	}
	var page struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(page.Data))
	for _, m := range page.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}
