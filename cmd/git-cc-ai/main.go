package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/dlnilsson/git-cc-ai/pkg/agentrc"
	"github.com/dlnilsson/git-cc-ai/pkg/providers"
	"github.com/dlnilsson/git-cc-ai/pkg/providers/claude"
	"github.com/dlnilsson/git-cc-ai/pkg/providers/codex"
	"github.com/dlnilsson/git-cc-ai/pkg/providers/gemini"
	"github.com/dlnilsson/git-cc-ai/pkg/ui"
)

const (
	menuSentinel       = "menu"
	errInvalidModelFmt = "invalid model %q (use -m for interactive pick, or one of: %s)\n"
)

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

func printHelp() {
	const help = `git-cc-ai — generate conventional commit messages from staged changes.

The tool runs an AI backend on your staged diff and prints a conventional commit
message to stdout. Use it with git commit (e.g. via the git-ai script) and
optionally edit the message in your editor before committing.

Requirements:
  Claude, Gemini or Codex must be installed and on your PATH.
  The backend is auto-detected (claude preferred) or set via GIT_AI_BACKEND.

Backends:
  claude   Anthropic Claude CLI (preferred when found in PATH)
  gemini   Google Gemini CLI
  codex    OpenAI Codex CLI

Environment:
  GIT_AI_BACKEND: backend provider (auto-detected from PATH if unset).
  GIT_AI_MODEL:   model name (overridden by -m / --model flags).
  GIT_AI_NO_CC:      set to "true" to use standard commit style instead of
                     Conventional Commits.
  GIT_AI_NO_SESSION: set to "true" to skip resuming a CLAUDE_SESSION_ID.
  GIT_AI_BUDGET:     maximum spend in USD per run (default: 1.0).

Get started:
  1. Stage your changes: git add ...
  2. Run: git ai (or git-cc-ai if not using a git alias)
  3. The backend drafts a conventional commit message and opens your editor so
     you can confirm or edit, then commit.

Flags:
`
	fmt.Fprint(os.Stderr, help)
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
}

func execInPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
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
	flag.BoolVar(&noSpinner, "no-spinner", false, "disable spinner while the backend runs")
	flag.StringVar(&model, "model", "", "model name (overrides -m)")
	flag.StringVar(&mFlag, "m", "", "model name, or no value for interactive selection")
	flag.Usage = printHelp
	flag.Parse()
	if flag.NArg() > 0 {
		extraNote = strings.Join(flag.Args(), " ")
	}

	rc := agentrc.Load(".agentrc")

	backends := map[string]providers.Backend{
		"codex":  codex.Backend{},
		"claude": claude.Backend{},
		"gemini": gemini.Backend{},
	}
	backend := strings.TrimSpace(os.Getenv("GIT_AI_BACKEND"))
	if backend == "" {
		backend = rc.Backend
	}
	if backend == "" {
		switch {
		case execInPath("claude"):
			backend = "claude"
		case execInPath("gemini"):
			backend = "gemini"
		case execInPath("codex"):
			backend = "codex"
		default:
			fmt.Fprintln(os.Stderr, "no supported backend found in PATH (install claude, gemini or codex)")
			os.Exit(1)
		}
	}
	b, ok := backends[backend]
	if !ok {
		available := make([]string, 0, len(backends))
		for name := range backends {
			available = append(available, name)
		}
		sort.Strings(available)
		fmt.Fprintf(os.Stderr, "invalid GIT_AI_BACKEND value %q (available: %s)\n", backend, strings.Join(available, ", "))
		os.Exit(1)
	}

	// --model flag is explicit user intent — validate strictly.
	// GIT_AI_MODEL / .agentrc is a soft preference — silently fall back to
	// the provider default when the model doesn't match.
	modelFromFlag := strings.TrimSpace(model) != "" || strings.TrimSpace(mFlag) != ""
	if envModel := strings.TrimSpace(os.Getenv("GIT_AI_MODEL")); envModel != "" && !modelFromFlag {
		model = envModel
	}
	if rc.Model != "" && strings.TrimSpace(model) == "" && !modelFromFlag {
		model = rc.Model
	}

	availableModels := b.Models()
	switch {
	case strings.TrimSpace(model) != "":
		model = strings.TrimSpace(model)
		if !slices.Contains(availableModels, model) {
			if modelFromFlag {
				fmt.Fprintf(os.Stderr, errInvalidModelFmt, model, strings.Join(availableModels, ", "))
				os.Exit(1)
			}
			model = ""
		}
	case strings.TrimSpace(mFlag) == "":
		// No model specified — provider will use its default.
	case mFlag == menuSentinel:
		selected, err := ui.SelectModelMenu(availableModels)
		if err != nil {
			os.Exit(1)
		}
		model = selected
	default:
		candidate := strings.TrimSpace(mFlag)
		if !slices.Contains(availableModels, candidate) {
			fmt.Fprintf(os.Stderr, errInvalidModelFmt, candidate, strings.Join(availableModels, ", "))
			os.Exit(1)
		}
		model = candidate
	}

	var registry providers.Registry
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		for sig := range sigCh {
			registry.ForwardSignal(sig)
			registry.StopSpinnerIfSet()
		}
	}()

	noCC := strings.EqualFold(strings.TrimSpace(os.Getenv("GIT_AI_NO_CC")), "true") || rc.NoCC
	noSession := strings.EqualFold(strings.TrimSpace(os.Getenv("GIT_AI_NO_SESSION")), "true") || rc.NoSession

	var budget float64
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("GIT_AI_BUDGET")), 64); err == nil && v > 0 {
		budget = v
	} else if rc.Budget > 0 {
		budget = rc.Budget
	}

	var sessionID string
	if !noSession {
		sessionID = rc.SessionID
	}

	message, err := b.Generate(ctx, &registry, providers.Options{
		SkillPath:   skillPath,
		ExtraNote:   extraNote,
		Model:       model,
		SessionID:   sessionID,
		ShowSpinner: !noSpinner,
		NoCC:        noCC,
		Budget:      budget,
	})
	if err != nil {
		fmt.Fprintf(os.Stdout, "\n\n\n# something went wrong %s\n", err.Error()) //nolint:errcheck
		fmt.Fprintln(os.Stderr, err.Error())                                     //nolint:errcheck
		os.Exit(1)
	}
	if strings.TrimSpace(message) == "" {
		fmt.Print("\n\n# something went wrong\n")
		return
	}
	fmt.Print(strings.TrimSpace(message))
}
