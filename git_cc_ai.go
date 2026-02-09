package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type spinnerDoneMsg struct{}

type spinnerReasoningMsg string

type spinnerModel struct {
	spinner           spinner.Model
	message           string
	reasoning         string
	reasoningRendered string
	done              bool
	start             time.Time
}

type codexUsage struct {
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
}

type modelSelectModel struct {
	choices  []string
	cursor   int
	selected string
	done     bool
}

var spinnerMessages = []string{
	"Generating commit message with Codex...",
	"Summarizing staged changes...",
	"Drafting Conventional Commit...",
	"Giving birth to skynet",
	"Analyzing diff hunks...",
	"Composing commit summary...",
	"Buying Sam Altman a new ferrari...",
}

var spinnerStyles = []spinner.Spinner{
	spinner.Line,
	spinner.Dot,
	spinner.MiniDot,
	spinner.Jump,
	spinner.Pulse,
	spinner.Points,
	spinner.Globe,
	spinner.Moon,
	spinner.Monkey,
}

var reasoningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
var markdownRenderer *glamour.TermRenderer

type spinnerHandle struct {
	program  *tea.Program
	reasonCh chan string
	doneCh   chan struct{}
}

var activeSpinner *spinnerHandle

// From: https://raw.githubusercontent.com/conventional-commits/conventionalcommits.org/refs/heads/master/content/v1.0.0/index.md
const conventionalSpec = `Conventional Commits 1.0.0 Spec
Summary
The Conventional Commits specification is a lightweight convention on top of commit messages. It provides an easy set of rules for creating an explicit commit history; which makes it easier to write automated tools on top of. This convention dovetails with SemVer, by describing the features, fixes, and breaking changes made in commit messages.

The commit message should be structured as follows:

<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]

The commit contains the following structural elements, to communicate intent to the consumers of your library:
- fix: a commit of the type fix patches a bug in your codebase (correlates with PATCH in Semantic Versioning).
- feat: a commit of the type feat introduces a new feature to the codebase (correlates with MINOR in Semantic Versioning).
- BREAKING CHANGE: a commit that has a footer BREAKING CHANGE:, or appends a ! after the type/scope, introduces a breaking API change (correlates with MAJOR in Semantic Versioning). A BREAKING CHANGE can be part of commits of any type.
- types other than fix and feat are allowed (e.g., build, chore, ci, docs, style, refactor, perf, test, and others).
- footers other than BREAKING CHANGE: may be provided and follow a convention similar to git trailer format.
- a scope may be provided to a commit's type and is contained within parenthesis, e.g., feat(parser): add ability to parse arrays.

Specification
1. Commits MUST be prefixed with a type, which consists of a noun (feat, fix, etc.), followed by the OPTIONAL scope, OPTIONAL !, and REQUIRED terminal colon and space.
2. The type feat MUST be used when a commit adds a new feature.
3. The type fix MUST be used when a commit represents a bug fix.
4. A scope MAY be provided after a type. A scope MUST consist of a noun describing a section of the codebase surrounded by parenthesis, e.g., fix(parser):
5. A description MUST immediately follow the colon and space after the type/scope prefix.
6. The description is a short summary of the code changes.
7. A longer commit body MAY be provided after the short description. The body MUST begin one blank line after the description.
8. A commit body is free-form and MAY consist of any number of newline separated paragraphs.
9. One or more footers MAY be provided one blank line after the body.
10. Each footer MUST consist of a word token, followed by either a : or # separator, followed by a string value (inspired by git trailer convention). A footer's token MUST use - in place of whitespace characters (e.g., Acked-by). An exception is made for BREAKING CHANGE which MAY also be used as a token.
11. A footer's value MAY contain spaces and newlines, and parsing MUST terminate when the next valid footer token/separator pair is observed.
12. Breaking changes MUST be indicated in the type/scope prefix of a commit, or as an entry in the footer.
13. If included as a footer, a breaking change MUST consist of the uppercase text BREAKING CHANGE, followed by a colon, space, and description.
14. If included in the type/scope prefix, breaking changes MUST be indicated by a ! immediately before the :. If ! is used, BREAKING CHANGE: MAY be omitted from the footer, and the commit description SHALL be used to describe the breaking change.
15. Types other than feat and fix MAY be used in your commit messages.
16. The units of information that make up Conventional Commits MUST NOT be treated as case sensitive by implementors, with the exception of BREAKING CHANGE which MUST be uppercase. BREAKING-CHANGE MUST be synonymous with BREAKING CHANGE, when used as a token in a footer.
`

const commitBodyLineWidth = 72

func wrapCommitMessage(msg string, width int) string {
	paragraphs := strings.Split(msg, "\n\n")
	out := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			out = append(out, "")
			continue
		}
		run := strings.ReplaceAll(p, "\n", " ")
		var (
			line strings.Builder
			pos  int
		)
		for pos < len(run) {
			for pos < len(run) && run[pos] == ' ' {
				pos++
			}
			if pos >= len(run) {
				break
			}
			start := pos
			for pos < len(run) && run[pos] != ' ' {
				pos++
			}
			word := run[start:pos]
			newLen := line.Len()
			if newLen > 0 {
				newLen++
			}
			newLen += len(word)
			if newLen > width && line.Len() > 0 {
				lineStr := line.String()
				lastSent := -1
				for i := len(lineStr) - 1; i >= 0 && i >= len(lineStr)-width; i-- {
					if i > 0 && (lineStr[i] == '.' || lineStr[i] == '?' || lineStr[i] == '!') && lineStr[i-1] != '.' {
						if i+1 < len(lineStr) && lineStr[i+1] == ' ' {
							lastSent = i + 2
							break
						}
						if i+1 >= len(lineStr) {
							lastSent = i + 1
							break
						}
					}
				}
				var breakAt int
				if lastSent > 0 && lastSent <= len(lineStr) {
					breakAt = lastSent
				} else {
					lastSpace := strings.LastIndex(lineStr, " ")
					if lastSpace > 0 {
						breakAt = lastSpace + 1
					} else {
						breakAt = len(lineStr)
					}
				}
				out = append(out, strings.TrimSpace(lineStr[:breakAt]))
				line.Reset()
				line.WriteString(strings.TrimLeft(lineStr[breakAt:], " "))
			}
			if line.Len() > 0 {
				line.WriteByte(' ')
			}
			line.WriteString(word)
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	return strings.Join(out, "\n")
}

func newSpinnerModel(message string) spinnerModel {
	s := spinner.New()
	s.Spinner = randomSpinnerStyle()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinnerModel{spinner: s, message: message, start: time.Now()}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		return m, tea.Quit
	case spinnerReasoningMsg:
		m.reasoning = string(msg)
		m.reasoningRendered = renderReasoning(m.reasoning)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m spinnerModel) View() string {
	if m.done {
		return "\r\033[2K"
	}
	elapsed := time.Since(m.start).Round(100 * time.Millisecond)
	if strings.TrimSpace(m.reasoningRendered) != "" {
		return fmt.Sprintf("\n  %s %s (%s)\n  %s\n", m.spinner.View(), m.message, elapsed, m.reasoningRendered)
	}
	return fmt.Sprintf("\n  %s %s (%s)\n", m.spinner.View(), m.message, elapsed)
}

const menuSentinel = "menu"

// https://developers.openai.com/codex/models/
var models = []string{
	"gpt-5.1-codex-max",
	"gpt-5.1-codex-mini",
	"gpt-5.2-codex",
	"gpt-5.3-codex",
}

func injectBareM() {
	args := os.Args
	var out []string
	for i := 0; i < len(args); i++ {
		out = append(out, args[i])
		if args[i] != "-m" {
			continue
		}
		next := i + 1
		if next >= len(args) || strings.HasPrefix(args[next], "-") {
			out = append(out, menuSentinel)
			continue
		}
		out = append(out, args[next])
		i = next
	}
	os.Args = out
}

func main() {
	var (
		mFlag     string
		model     string
		noSpinner bool
		skillPath string
		extraNote string
	)

	injectBareM()
	flag.StringVar(&skillPath, "skill-path", "", "path to SKILL.md (optional, used for prompt)")
	flag.BoolVar(&noSpinner, "no-spinner", false, "disable spinner while codex runs")
	flag.StringVar(&model, "model", "", "model name (overrides -m)")
	flag.StringVar(&mFlag, "m", "", "model name, or no value for interactive selection")
	flag.Parse()
	if flag.NArg() > 0 {
		extraNote = strings.Join(flag.Args(), " ")
	}

	switch {
	case strings.TrimSpace(model) != "":
		model = strings.TrimSpace(model)
		if !modelInList(model, models) {
			fmt.Fprintf(os.Stderr, "invalid model %q (use -m for interactive pick, or one of: %s)\n", model, strings.Join(models, ", "))
			os.Exit(1)
		}
	case strings.TrimSpace(mFlag) == "":
	case mFlag == menuSentinel:
		selected, err := selectModelMenu(models)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		model = selected
	default:
		candidate := strings.TrimSpace(mFlag)
		if !modelInList(candidate, models) {
			fmt.Fprintf(os.Stderr, "invalid model %q (use -m for interactive pick, or one of: %s)\n", candidate, strings.Join(models, ", "))
			os.Exit(1)
		}
		model = candidate
	}

	message, err := generateWithCodex(skillPath, extraNote, model, !noSpinner)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if strings.TrimSpace(message) == "" {
		fmt.Fprintln(os.Stderr, "Codex returned empty output.")
		os.Exit(1)
	}
	fmt.Print(strings.TrimSpace(message))
}

func generateWithCodex(skillPath, extraNote, model string, showSpinner bool) (string, error) {
	const (
		codexCmd  = "codex"
		codexArgs = "exec --json"
	)
	var (
		args          []string
		buffer        strings.Builder
		cmd           *exec.Cmd
		diff          string
		err           error
		output        string
		prompt        strings.Builder
		reasoningText string
		scanner       *bufio.Scanner
		skillText     string
		stdout        io.ReadCloser
		stopSpinner   func()
		usage         codexUsage
		startTime     time.Time
	)

	diff, err = gitDiffStaged()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("No staged diff content found.")
	}

	skillText = conventionalSpec
	if skillPath != "" {
		if data, readErr := os.ReadFile(skillPath); readErr == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				skillText = skillText + "\nAdditional instructions:\n" + trimmed
			}
		}
	}

	prompt.WriteString("Generate a Conventional Commit message from the staged git diff.\n")
	prompt.WriteString("Use the instructions below and output only the commit message.\n")
	prompt.WriteString("Limit each line in the commit body to 72 characters; wrap at sentence boundaries (e.g. after a period and space) when possible so lines do not break mid-sentence.\n\n")
	prompt.WriteString("Instructions:\n")
	prompt.WriteString(skillText)
	prompt.WriteString("\n\n")
	prompt.WriteString("Staged diff:\n")
	prompt.WriteString(diff)
	prompt.WriteString("\n")
	if strings.TrimSpace(extraNote) != "" {
		prompt.WriteString("\nExtra context:\n")
		prompt.WriteString(strings.TrimSpace(extraNote))
		prompt.WriteString("\n")
	}

	args = splitArgs(codexArgs)
	if strings.TrimSpace(model) != "" {
		args = addModelArg(args, model)
	}
	cmd = exec.Command(codexCmd, args...)
	cmd.Stdin = strings.NewReader(prompt.String())
	cmd.Stderr = os.Stderr
	startTime = time.Now()
	if showSpinner {
		stopSpinner = startSpinner(randomSpinnerMessage())
		defer stopSpinner()
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}

	scanner = bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteByte('\n')
		if showSpinner {
			reasoningText = parseReasoningJSON(line)
			if strings.TrimSpace(reasoningText) != "" {
				sendSpinnerReasoning(reasoningText)
			}
		}
		if updated, ok := parseUsageJSON(line); ok {
			usage = updated
		}
	}
	if err = scanner.Err(); err != nil {
		return "", err
	}
	if err = cmd.Wait(); err != nil {
		return "", errors.New("codex invocation failed")
	}

	output = strings.TrimSpace(buffer.String())
	if output == "" {
		return "", nil
	}

	if parsed := parseCodexJSON(output); strings.TrimSpace(parsed) != "" {
		return appendUsageComment(wrapCommitMessage(strings.TrimSpace(parsed), commitBodyLineWidth), usage, time.Since(startTime), model), nil
	}

	if strings.HasPrefix(output, "{") {
		if extracted := extractJSONField(output, []string{"output", "stdout", "result", "message"}); strings.TrimSpace(extracted) != "" {
			return appendUsageComment(wrapCommitMessage(strings.TrimSpace(extracted), commitBodyLineWidth), usage, time.Since(startTime), model), nil
		}
	}

	return appendUsageComment(wrapCommitMessage(output, commitBodyLineWidth), usage, time.Since(startTime), model), nil
}

func randomSpinnerMessage() string {
	if len(spinnerMessages) == 0 {
		return "Generating commit message with Codex..."
	}
	seed := time.Now().UnixNano()
	return spinnerMessages[int(seed%int64(len(spinnerMessages)))]
}

func randomSpinnerStyle() spinner.Spinner {
	if len(spinnerStyles) == 0 {
		return spinner.Dot
	}
	seed := time.Now().UnixNano()
	return spinnerStyles[int(seed%int64(len(spinnerStyles)))]
}

func startSpinner(message string) func() {
	_ = os.Setenv("CLICOLOR_FORCE", "1")
	lipgloss.SetColorProfile(termenv.ANSI)
	markdownRenderer = newMarkdownRenderer()
	p := tea.NewProgram(newSpinnerModel(message), tea.WithOutput(os.Stderr))
	handle := &spinnerHandle{
		program:  p,
		reasonCh: make(chan string, 8),
		doneCh:   make(chan struct{}),
	}
	activeSpinner = handle
	done := make(chan struct{})
	go func() {
		_, _ = p.Run()
		close(done)
	}()
	go func() {
		for {
			select {
			case text := <-handle.reasonCh:
				if strings.TrimSpace(text) != "" {
					handle.program.Send(spinnerReasoningMsg(text))
				}
			case <-handle.doneCh:
				return
			}
		}
	}()
	return func() {
		handle.program.Send(spinnerDoneMsg{})
		<-done
		close(handle.doneCh)
		activeSpinner = nil
	}
}

var errNotGitDir = errors.New("not a git directory")

func gitDiffStaged() (string, error) {
	check := exec.Command("git", "rev-parse", "--git-dir")
	check.Stderr = io.Discard
	if err := check.Run(); err != nil {
		return "", errNotGitDir
	}
	cmd := exec.Command("git", "diff", "--staged")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read staged diff (git diff --staged): %w", err)
	}
	return string(out), nil
}

func splitArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	return strings.Fields(raw)
}

func addModelArg(args []string, model string) []string {
	if len(args) == 0 {
		return []string{"-m", model}
	}
	out := make([]string, 0, len(args)+2)
	if args[0] == "exec" {
		out = append(out, args[0], "-m", model)
		out = append(out, args[1:]...)
		return out
	}
	out = append(out, args...)
	out = append(out, "-m", model)
	return out
}

func modelInList(name string, list []string) bool {
	return slices.Contains(list, name)
}

func selectModelMenu(choices []string) (string, error) {
	if len(choices) == 0 {
		return "", errors.New("no models available for selection")
	}
	m := modelSelectModel{choices: choices}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	selected := final.(modelSelectModel).selected
	if strings.TrimSpace(selected) == "" {
		return "", errors.New("no model selected")
	}
	return selected, nil
}

func (m modelSelectModel) Init() tea.Cmd {
	return nil
}

func (m modelSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "enter":
			m.selected = m.choices[m.cursor]
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m modelSelectModel) View() string {
	if m.done {
		return "\r\033[2K"
	}
	var b strings.Builder
	b.WriteString("\nSelect a model:\n\n")
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %s\n", cursor, choice))
	}
	b.WriteString("\nEnter to select, q/esc to cancel.\n")
	return b.String()
}

func sendSpinnerReasoning(text string) {
	if activeSpinner == nil {
		return
	}
	select {
	case activeSpinner.reasonCh <- text:
	default:
	}
}

func newMarkdownRenderer() *glamour.TermRenderer {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil
	}
	return renderer
}

func renderReasoning(text string) string {
	if markdownRenderer == nil {
		return reasoningStyle(text)
	}
	out, err := markdownRenderer.Render(text)
	if err != nil {
		return reasoningStyle(text)
	}
	return out
}

func parseReasoningJSON(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}
	if t, ok := msg["type"].(string); ok {
		if t == "reasoning" {
			if text, ok := msg["text"].(string); ok {
				return text
			}
		}
		if t == "item.completed" {
			if item, ok := msg["item"].(map[string]any); ok {
				if it, ok := item["type"].(string); ok && it == "reasoning" {
					if text, ok := item["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func parseCodexJSON(raw string) string {
	var (
		lines = strings.Split(raw, "\n")
		last  = ""
	)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if t, ok := msg["type"].(string); ok {
			if t == "agent_message" {
				if text, ok := msg["text"].(string); ok && strings.TrimSpace(text) != "" {
					last = text
				}
				continue
			}
			if t == "item.completed" {
				if item, ok := msg["item"].(map[string]any); ok {
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

func parseUsageJSON(raw string) (codexUsage, bool) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return codexUsage{}, false
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return codexUsage{}, false
	}
	if t, ok := msg["type"].(string); !ok || t != "turn.completed" {
		return codexUsage{}, false
	}
	usage, ok := msg["usage"].(map[string]any)
	if !ok {
		return codexUsage{}, false
	}
	return codexUsage{
		InputTokens:       toInt(usage["input_tokens"]),
		CachedInputTokens: toInt(usage["cached_input_tokens"]),
		OutputTokens:      toInt(usage["output_tokens"]),
	}, true
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func appendUsageComment(message string, usage codexUsage, elapsed time.Duration, model string) string {
	if usage == (codexUsage{}) {
		return message
	}
	elapsedText := elapsed.Round(100 * time.Millisecond)
	comment := message + "\n\n# tokens: input=" + fmt.Sprint(usage.InputTokens) +
		" cached=" + fmt.Sprint(usage.CachedInputTokens) +
		" output=" + fmt.Sprint(usage.OutputTokens) +
		" elapsed=" + elapsedText.String()
	if strings.TrimSpace(model) != "" {
		comment = comment + " model=" + model
	}
	return comment
}

func extractJSONField(raw string, keys []string) string {
	for _, key := range keys {
		var (
			needle = `"` + key + `":`
			idx    = strings.Index(raw, needle)
		)
		if idx == -1 {
			continue
		}
		rest := raw[idx+len(needle):]
		rest = strings.TrimLeft(rest, " \n\r\t")
		if strings.HasPrefix(rest, "\"") {
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
