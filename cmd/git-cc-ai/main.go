package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/dlnilsson/git-cc-ai/pkg/providers"
	"github.com/dlnilsson/git-cc-ai/pkg/providers/claude"
	"github.com/dlnilsson/git-cc-ai/pkg/providers/codex"
	"github.com/dlnilsson/git-cc-ai/pkg/ui"
)

const menuSentinel = "menu"
const errInvalidModelFmt = "invalid model %q (use -m for interactive pick, or one of: %s)\n"

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
	const help = `git-cc-ai — generate conventional commit messages from staged changes using Codex.

The tool runs Codex on your staged diff and prints a conventional commit message
to stdout. Use it with git commit (e.g. via the git-ai script) and optionally
edit the message in your editor before committing.

Requirements:
  Codex must be installed and on your PATH (the binary invokes "codex" by default).

Environment:
  GIT_AI_BACKEND: backend provider (default: codex).

Get started:
  1. Stage your changes: git add ...
  2. Run: git ai (or git-cc-ai if not using a git alias)
  3. Codex drafts a conventional commit message and opens your editor so you can
     confirm or edit, then commit.

Flags:
`
	fmt.Fprint(os.Stderr, help)
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
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
	flag.Usage = printHelp
	flag.Parse()
	if flag.NArg() > 0 {
		extraNote = strings.Join(flag.Args(), " ")
	}

	backends := map[string]providers.Backend{
		"codex":  codex.Backend{},
		"claude": claude.Backend{},
	}
	backend := strings.TrimSpace(os.Getenv("GIT_AI_BACKEND"))
	if backend == "" {
		backend = "codex"
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

	if backend == "codex" {
		models := codex.Models()
		switch {
		case strings.TrimSpace(model) != "":
			model = strings.TrimSpace(model)
			if !codex.IsModelSupported(model) {
				fmt.Fprintf(os.Stderr, errInvalidModelFmt, model, strings.Join(models, ", "))
				os.Exit(1)
			}
		case strings.TrimSpace(mFlag) == "":
		case mFlag == menuSentinel:
			selected, err := ui.SelectModelMenu(models)
			if err != nil {
				os.Exit(1)
			}
			model = selected
		default:
			candidate := strings.TrimSpace(mFlag)
			if !codex.IsModelSupported(candidate) {
				fmt.Fprintf(os.Stderr, errInvalidModelFmt, candidate, strings.Join(models, ", "))
				os.Exit(1)
			}
			model = candidate
		}
	}

	var registry providers.Registry
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			registry.ForwardSignal(sig)
			registry.StopSpinnerIfSet()
		}
	}()

	message, err := b.Generate(&registry, providers.Options{
		SkillPath:   skillPath,
		ExtraNote:   extraNote,
		Model:       model,
		ShowSpinner: !noSpinner,
	})
	if err != nil {
		fmt.Fprintf(os.Stdout, "\n\n\n# something went wrong %s\n", err.Error())
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if strings.TrimSpace(message) == "" {
		fmt.Print("\n\n# something went wrong\n")
		return
	}
	fmt.Print(strings.TrimSpace(message))
}
