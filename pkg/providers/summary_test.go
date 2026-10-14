package providers

import "testing"

func TestWrapAISummary(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		assisted string
		want     string
	}{
		{
			name:    "subject and body",
			message: "feat: add thing\n\nWhy it matters.\n\n- point one",
			want:    "feat: add thing\n\n<details><summary>AI Summary</summary>\n\n> Why it matters.\n>\n> - point one\n\nAssisted-by: Claude:opus\n\n</details>",
		},
		{
			name:    "subject only",
			message: "feat: add thing",
			want:    "feat: add thing\n\n<details><summary>AI Summary</summary>\n\nAssisted-by: Claude:opus\n\n</details>",
		},
		{
			name:     "subject only without trailer",
			message:  "feat: add thing",
			assisted: "false",
			want:     "feat: add thing",
		},
		{
			name:     "body without trailer",
			message:  "feat: add thing\n\nWhy it matters.\n\n- point one",
			assisted: "false",
			want:     "feat: add thing\n\n<details><summary>AI Summary</summary>\n\n> Why it matters.\n>\n> - point one\n\n</details>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GIT_AI_ASSISTED_BY", tt.assisted)
			if got := WrapAISummary(tt.message, "Claude", "opus"); got != tt.want {
				t.Fatalf("WrapAISummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
