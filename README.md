# git ai

[![asciicast](https://asciinema.org/a/KWh6aVDScK35Pzjk.svg)](https://asciinema.org/a/KWh6aVDScK35Pzjk)

Generates conventional commit messages from your staged changes using Codex. Use the generated message with `git commit` (e.g. via the `git-ai` script) and optionally edit it in your editor before committing.

## Requirements

- **Go** — to build the binary
- **Codex** — installed and on your `PATH` (the binary invokes `codex` by default)

## Install

```bash
make install
```

This installs the Go binary and [scripts/git-ai](scripts/git-ai) into `$(BINDIR)` (default `~/.local/bin`). Ensure that directory is on your `PATH`.

## Get started

1. Stage your changes: `git add ...`
2. Run: `git ai` (or `git-ai` if not using a git alias)
3. Codex drafts a conventional commit message and opens your editor so you can confirm or edit, then commit.
