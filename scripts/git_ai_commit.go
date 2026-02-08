package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type inference struct {
	typ    string
	reason string
}

type spinnerDoneMsg struct{}

type spinnerModel struct {
	spinner spinner.Model
	message string
}

func newSpinnerModel(message string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinnerModel{spinner: s, message: message}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinnerDoneMsg:
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m spinnerModel) View() string {
	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.message)
}

func main() {
	var typ string
	var scope string
	var summary string
	var body string
	var footer string
	var breaking bool
	var breakingDesc string
	var maxSummary int
	var useCodex bool
	var noCodex bool
	var codexCmd string
	var codexArgs string
	var skillPath string
	var noSpinner bool

	flag.StringVar(&typ, "type", "", "commit type")
	flag.StringVar(&scope, "scope", "", "commit scope")
	flag.StringVar(&summary, "summary", "", "summary line")
	flag.StringVar(&body, "body", "", "body text")
	flag.StringVar(&footer, "footer", "", "footer text")
	flag.BoolVar(&breaking, "breaking", false, "add breaking change marker")
	flag.StringVar(&breakingDesc, "breaking-desc", "", "breaking change description")
	flag.IntVar(&maxSummary, "max-summary", 72, "max summary length")
	flag.BoolVar(&useCodex, "use-codex", true, "use codex to generate a detailed message from staged diff")
	flag.BoolVar(&noCodex, "no-codex", false, "disable codex and use heuristic message")
	flag.StringVar(&codexCmd, "codex-cmd", "codex", "codex command name or path")
	flag.StringVar(&codexArgs, "codex-args", "exec --skip-git-repo-check --json", "args for codex invocation")
	flag.StringVar(&skillPath, "skill-path", "", "path to SKILL.md (optional, used for prompt)")
	flag.BoolVar(&noSpinner, "no-spinner", false, "disable spinner while codex runs")
	flag.Parse()

	files, err := stagedFiles()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No staged changes found.")
		os.Exit(1)
	}

	if noCodex {
		useCodex = false
	}

	if useCodex {
		message, err := generateWithCodex(codexCmd, codexArgs, skillPath, typ, scope, summary, body, footer, breaking, breakingDesc, !noSpinner)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if strings.TrimSpace(message) != "" {
			fmt.Print(strings.TrimSpace(message))
			return
		}
		fmt.Fprintln(os.Stderr, "Codex returned empty output, falling back to heuristic message.")
	}

	if typ == "" {
		inf := inferType(files)
		typ = inf.typ
		if inf.reason != "" {
			fmt.Fprintln(os.Stderr, "Inferred type:", typ+" ("+inf.reason+")")
		}
	}

	if scope == "" {
		scope = inferScope(files)
		if scope != "" {
			fmt.Fprintln(os.Stderr, "Inferred scope:", scope)
		}
	}

	if summary == "" {
		if scope != "" {
			summary = "update " + scope
		} else {
			summary = "update files"
		}
	}

	summary = strings.TrimSpace(summary)
	if maxSummary > 0 && len(summary) > maxSummary {
		summary = summary[:maxSummary]
		fmt.Fprintln(os.Stderr, "Summary truncated to", maxSummary, "characters.")
	}

	subject := buildSubject(typ, scope, summary, breaking)
	out := bytes.Buffer{}
	out.WriteString(subject)

	body = strings.TrimSpace(body)
	footer = strings.TrimSpace(footer)
	if body != "" {
		out.WriteString("\n\n")
		out.WriteString(body)
	}

	footers := []string{}
	if breaking {
		desc := strings.TrimSpace(breakingDesc)
		if desc == "" {
			desc = "behavior changed"
		}
		footers = append(footers, "BREAKING CHANGE: "+desc)
	}
	if footer != "" {
		footers = append(footers, footer)
	}
	if len(footers) > 0 {
		out.WriteString("\n\n")
		out.WriteString(strings.Join(footers, "\n"))
	}

	fmt.Print(out.String())
}

func generateWithCodex(codexCmd, codexArgs, skillPath, typ, scope, summary, body, footer string, breaking bool, breakingDesc string, showSpinner bool) (string, error) {
	diff, err := gitDiffStaged()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("No staged diff content found.")
	}

	skillText := ""
	if skillPath != "" {
		if data, readErr := os.ReadFile(skillPath); readErr == nil {
			skillText = string(data)
		}
	}
	if skillText == "" {
		skillText = "Write Conventional Commit messages. Format: type(scope)!: summary with optional body and footer. Use BREAKING CHANGE footer for breaking changes."
	}

	constraints := []string{}
	if typ != "" {
		constraints = append(constraints, "type must be "+typ)
	}
	if scope != "" {
		constraints = append(constraints, "scope must be "+scope)
	}
	if summary != "" {
		constraints = append(constraints, "summary must be: "+summary)
	}
	if breaking {
		constraints = append(constraints, "include breaking change marker (!) and a BREAKING CHANGE footer")
	}
	if breakingDesc != "" {
		constraints = append(constraints, "BREAKING CHANGE description: "+breakingDesc)
	}
	if body != "" {
		constraints = append(constraints, "body must include: "+body)
	}
	if footer != "" {
		constraints = append(constraints, "footer must include: "+footer)
	}

	prompt := strings.Builder{}
	prompt.WriteString("Generate a Conventional Commit message from the staged git diff.\n")
	prompt.WriteString("Use the instructions below and output only the commit message.\n\n")
	prompt.WriteString("Instructions:\n")
	prompt.WriteString(skillText)
	prompt.WriteString("\n\n")
	if len(constraints) > 0 {
		prompt.WriteString("Constraints:\n")
		for _, c := range constraints {
			prompt.WriteString("- " + c + "\n")
		}
		prompt.WriteString("\n")
	}
	prompt.WriteString("Staged diff:\n")
	prompt.WriteString(diff)
	prompt.WriteString("\n")

	args := splitArgs(codexArgs)
	cmd := exec.Command(codexCmd, args...)
	cmd.Stdin = strings.NewReader(prompt.String())
	cmd.Stderr = os.Stderr
	var stopSpinner func()
	if showSpinner {
		stopSpinner = startSpinner("Generating commit message with Codex...")
		defer stopSpinner()
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("codex invocation failed")
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil
	}

	if parsed := parseCodexJSON(output); strings.TrimSpace(parsed) != "" {
		return strings.TrimSpace(parsed), nil
	}

	// Fallback: try to extract a common field from JSON payloads.
	if strings.HasPrefix(output, "{") {
		if extracted := extractJSONField(output, []string{"output", "stdout", "result", "message"}); strings.TrimSpace(extracted) != "" {
			return strings.TrimSpace(extracted), nil
		}
	}

	return output, nil
}

func startSpinner(message string) func() {
	p := tea.NewProgram(newSpinnerModel(message), tea.WithOutput(os.Stderr))
	done := make(chan struct{})
	go func() {
		_, _ = p.Run()
		close(done)
	}()
	return func() {
		p.Send(spinnerDoneMsg{})
		<-done
	}
}

func gitDiffStaged() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read staged diff (git diff --staged)")
	}
	return string(out), nil
}

func stagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read staged files (git diff --staged --name-only)")
	}
	lines := strings.Split(string(out), "\n")
	files := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
}

func splitArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Fields(raw)
	return parts
}

func parseCodexJSON(raw string) string {
	lines := strings.Split(raw, "\n")
	last := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if t, ok := msg["type"].(string); ok {
			// Direct agent message
			if t == "agent_message" {
				if text, ok := msg["text"].(string); ok && strings.TrimSpace(text) != "" {
					last = text
				}
				continue
			}
			// item.completed containing agent_message
			if t == "item.completed" {
				if item, ok := msg["item"].(map[string]interface{}); ok {
					if it, ok := item["type"].(string); ok && it == "agent_message" {
						if text, ok := item["text"].(string); ok && strings.TrimSpace(text) != "" {
							last = text
						}
					}
				}
			}
		}
	}
	return last
}

func extractJSONField(raw string, keys []string) string {
	for _, key := range keys {
		needle := `"` + key + `":`
		idx := strings.Index(raw, needle)
		if idx == -1 {
			continue
		}
		rest := raw[idx+len(needle):]
		rest = strings.TrimLeft(rest, " \n\r\t")
		if strings.HasPrefix(rest, "\"") {
			// naive JSON string extraction; handles basic escaped quotes
			rest = rest[1:]
			out := strings.Builder{}
			escaped := false
			for _, r := range rest {
				if escaped {
					out.WriteRune(r)
					escaped = false
					continue
				}
				if r == '\\' {
					escaped = true
					continue
				}
				if r == '"' {
					return out.String()
				}
				out.WriteRune(r)
			}
		}
	}
	return ""
}

func inferScope(files []string) string {
	scopes := map[string]struct{}{}
	for _, f := range files {
		f = filepath.ToSlash(f)
		parts := strings.Split(f, "/")
		if len(parts) > 1 {
			scopes[parts[0]] = struct{}{}
		}
	}
	if len(scopes) != 1 {
		return ""
	}
	for s := range scopes {
		return s
	}
	return ""
}

func inferType(files []string) inference {
	allDocs := true
	allTests := true
	allCI := true
	allBuild := true
	allChore := true

	for _, f := range files {
		f = filepath.ToSlash(f)
		if !isDocsFile(f) {
			allDocs = false
		}
		if !isTestFile(f) {
			allTests = false
		}
		if !isCIFile(f) {
			allCI = false
		}
		if !isBuildFile(f) {
			allBuild = false
		}
		if !isChoreFile(f) {
			allChore = false
		}
	}

	switch {
	case allDocs:
		return inference{typ: "docs", reason: "only documentation files"}
	case allTests:
		return inference{typ: "test", reason: "only test files"}
	case allCI:
		return inference{typ: "ci", reason: "only CI files"}
	case allBuild:
		return inference{typ: "build", reason: "only build or dependency files"}
	case allChore:
		return inference{typ: "chore", reason: "only repo config files"}
	default:
		return inference{typ: "feat", reason: "mixed changes (default)"}
	}
}

func buildSubject(typ, scope, summary string, breaking bool) string {
	header := typ
	if scope != "" {
		header += "(" + scope + ")"
	}
	if breaking {
		header += "!"
	}
	return header + ": " + summary
}

func isDocsFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "docs/") {
		return true
	}
	if strings.HasPrefix(lower, "doc/") {
		return true
	}
	if strings.HasPrefix(lower, "readme") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(lower))
	switch ext {
	case ".md", ".rst", ".txt":
		return true
	default:
		return false
	}
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") {
		return true
	}
	base := filepath.Base(lower)
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	if strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".test.js") {
		return true
	}
	if strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".test.ts") {
		return true
	}
	return false
}

func isCIFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, ".github/workflows/") {
		return true
	}
	if strings.HasPrefix(lower, ".circleci/") {
		return true
	}
	if strings.HasPrefix(lower, ".gitlab-ci") {
		return true
	}
	return false
}

func isBuildFile(path string) bool {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	buildFiles := map[string]struct{}{
		"go.mod":            {},
		"go.sum":            {},
		"package.json":      {},
		"package-lock.json": {},
		"yarn.lock":         {},
		"pnpm-lock.yaml":    {},
		"makefile":          {},
		"build.gradle":      {},
		"settings.gradle":   {},
		"gradle.properties": {},
		"pom.xml":           {},
		"cargo.toml":        {},
		"cargo.lock":        {},
		"setup.py":          {},
		"pyproject.toml":    {},
		"requirements.txt":  {},
		"gemfile":           {},
		"gemfile.lock":      {},
	}
	if _, ok := buildFiles[base]; ok {
		return true
	}
	return false
}

func isChoreFile(path string) bool {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	choreFiles := map[string]struct{}{
		".editorconfig":  {},
		".gitignore":     {},
		".gitattributes": {},
	}
	if _, ok := choreFiles[base]; ok {
		return true
	}
	return false
}
