package claude

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestListModelsNoAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if got := listModels(t.Context()); !slices.Equal(got, modelAliases) {
		t.Fatalf("got %v, want %v", got, modelAliases)
	}
}

func TestListModelsFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("x-api-key") != "key" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"data":[{"id":"claude-x-1"},{"id":"claude-y-2"}]}`)) //nolint:errcheck
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_API_KEY", "key")
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL+"/")

	want := append(append([]string{}, modelAliases...), "claude-x-1", "claude-y-2")
	if got := listModels(t.Context()); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListModelsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_API_KEY", "key")
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)

	if got := listModels(t.Context()); !slices.Equal(got, modelAliases) {
		t.Fatalf("got %v, want %v", got, modelAliases)
	}
}
