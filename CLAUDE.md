# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

git-ai (`git-cc-ai`) is a Go CLI tool that generates conventional commit messages from staged git changes using AI backends (Claude CLI or OpenAI Codex CLI). It runs the backend on `git diff --staged`, produces a commit message, and outputs it to stdout for use with `git commit`.

## Build & Development

```bash
# Build and install to ~/.local/bin
make install

# Build only
go build ./cmd/git-cc-ai

# Run tests
go test ./...

# Run a single test
go test ./pkg/commit/ -run TestBuildConventionalPrompt

# Vet
go vet ./...
```

There is no linter config (no golangci-lint); use `go vet` for static analysis.

## Architecture

**Entry point:** `cmd/git-cc-ai/main.go` — parses flags, loads `.agentrc` config, auto-detects backend from PATH (claude preferred), and delegates to the selected backend's `Generate` function.

**Key packages:**

- `pkg/providers/` — `Backend` interface with `Generate(reg *Registry, opts Options) (string, error)`. `Registry` manages the child process lifecycle and signal forwarding.
- `pkg/providers/claude/` — Claude CLI backend. Runs `claude` with `--output-format=stream-json`, parses streaming JSON events (text deltas, reasoning, result), extracts the final commit message.
- `pkg/providers/codex/` — Codex CLI backend. Runs `codex exec --json`, parses NDJSON events (`agent_message`, `reasoning`, `turn.completed`).
- `pkg/commit/` — Prompt building (`BuildConventionalPrompt`), message post-processing (`WrapMessage` at 72-char body width, `StripCodeFence`), and the embedded Conventional Commits spec.
- `pkg/git/` — Runs `git diff --staged` to get the diff.
- `pkg/ui/` — Bubbletea-based terminal spinner with live reasoning display and model selection menu.
- `pkg/agentrc/` — Parses `.agentrc` files for `CLAUDE_SESSION_ID` and `GIT_AI_BACKEND` exports.

**Data flow:** `main` → backend `Generate` → `git.DiffStaged()` → `commit.BuildConventionalPrompt()` → exec backend CLI → parse streaming output → `commit.StripCodeFence` → `commit.WrapMessage` → append usage comment → stdout.

**Shell wrapper:** `scripts/git-ai` calls `git-cc-ai`, captures the message, and runs `git commit -F - --edit` to open the editor.

## Go Code Conventions

- Use a single `var (...)` block instead of repeated `:=` for multiple local variables in a function.
- In tests, use `t.Context()` instead of `context.Background()` or `context.TODO()`.
- Use `errors.New()` for static errors; only use `fmt.Errorf()` when interpolating dynamic data.
- Pre-allocate slice capacity with `make([]T, 0, n)` when the size is known or estimable.
- Prefer type parameters over `interface{}` boxing; keep APIs concrete when possible.
- Use `strings.Builder` (with `Grow` when size is known) for string concatenation; use `strings.Join` for slices with a delimiter.
- Reuse large temporary buffers via `sync.Pool` in hot paths.
