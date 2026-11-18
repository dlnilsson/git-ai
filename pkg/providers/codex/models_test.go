package codex

import (
	"slices"
	"testing"
)

func TestParseModels(t *testing.T) {
	data := []byte(`{"models":[
		{"slug":"b","visibility":"list","priority":2},
		{"slug":"hidden","visibility":"hide","priority":0},
		{"slug":"a","visibility":"list","priority":1},
		{"slug":"","visibility":"list","priority":3}
	]}`)
	got, err := parseModels(data)
	if err != nil {
		t.Fatalf("parseModels: %v", err)
	}
	if want := []string{"a", "b"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseModelsErrors(t *testing.T) {
	for name, data := range map[string]string{
		"invalid json": `not json`,
		"none visible": `{"models":[{"slug":"x","visibility":"hide"}]}`,
		"empty":        `{"models":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseModels([]byte(data)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
