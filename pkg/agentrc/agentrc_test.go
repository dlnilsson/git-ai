package agentrc

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadAssistedBy(t *testing.T) {
	var (
		enabled  = true
		disabled = false
		tests    = []struct {
			name    string
			content string
			want    *bool
		}{
			{name: "unset"},
			{name: "true", content: "GIT_AI_ASSISTED_BY=true\n", want: &enabled},
			{name: "false", content: "GIT_AI_ASSISTED_BY=false\n", want: &disabled},
			{name: "double quoted true", content: "GIT_AI_ASSISTED_BY=\"true\"\n", want: &enabled},
			{name: "double quoted false", content: "GIT_AI_ASSISTED_BY=\"false\"\n", want: &disabled},
			{name: "single quoted true", content: "GIT_AI_ASSISTED_BY='true'\n", want: &enabled},
			{name: "single quoted false", content: "GIT_AI_ASSISTED_BY='false'\n", want: &disabled},
			{name: "quoted whitespace", content: "  export GIT_AI_ASSISTED_BY= \"false\"  \n", want: &disabled},
			{name: "export", content: "export GIT_AI_ASSISTED_BY=false\n", want: &disabled},
			{name: "whitespace", content: "  export GIT_AI_ASSISTED_BY= true  \n", want: &enabled},
			{name: "empty", content: "GIT_AI_ASSISTED_BY=\n"},
			{name: "quoted empty", content: "GIT_AI_ASSISTED_BY=\"\"\n"},
			{name: "unclosed quote", content: "GIT_AI_ASSISTED_BY=\"false\n"},
			{name: "mismatched quotes", content: "GIT_AI_ASSISTED_BY=\"false'\n"},
			{name: "invalid", content: "GIT_AI_ASSISTED_BY=invalid\n"},
			{name: "other boolean syntax", content: "GIT_AI_ASSISTED_BY=0\n"},
			{name: "last valid value", content: "GIT_AI_ASSISTED_BY=false\nGIT_AI_ASSISTED_BY=true\n", want: &enabled},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				path = filepath.Join(t.TempDir(), ".agentrc")
				cfg  Config
			)
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg = Load(path)
			if tt.want == nil {
				if cfg.AssistedBy != nil {
					t.Fatalf("AssistedBy = %t, want unset", *cfg.AssistedBy)
				}
				return
			}
			if cfg.AssistedBy == nil {
				t.Fatalf("AssistedBy is unset, want %t", *tt.want)
			}
			if *cfg.AssistedBy != *tt.want {
				t.Fatalf("AssistedBy = %t, want %t", *cfg.AssistedBy, *tt.want)
			}
		})
	}
}

func TestLoadQuotedValues(t *testing.T) {
	const content = `CLAUDE_SESSION_ID="test-session"
export GIT_AI_BACKEND="codex"
GIT_AI_MODEL="gpt-6-luna"
GIT_AI_NO_CC="true"
GIT_AI_NO_SESSION="true"
GIT_AI_ASSISTED_BY="false"
GIT_AI_BUDGET="0.5"
`
	var (
		disabled = false
		want     = Config{
			SessionID:  "test-session",
			Backend:    "codex",
			Model:      "gpt-6-luna",
			NoCC:       true,
			NoSession:  true,
			AssistedBy: &disabled,
			Budget:     0.5,
		}
		tests = []struct {
			name  string
			quote string
		}{
			{name: "unquoted"},
			{name: "double quoted", quote: `"`},
			{name: "single quoted", quote: "'"},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				path = filepath.Join(t.TempDir(), ".agentrc")
				data = strings.ReplaceAll(content, `"`, tt.quote)
			)
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := Load(path); !reflect.DeepEqual(got, want) {
				t.Fatalf("Load() = %+v, want %+v", got, want)
			}
		})
	}
}
