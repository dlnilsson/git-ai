# git ai

[![asciicast](https://asciinema.org/a/KWh6aVDScK35Pzjk.svg)](https://asciinema.org/a/KWh6aVDScK35Pzjk)

Generates conventional commit messages from your staged changes using Codex or Claude. Use the generated message with `git commit` (e.g. via the `git-ai` script) and optionally edit it in your editor before committing.

## Requirements

- **Go** — to build the binary
- **Codex** — installed and on your `PATH` (default backend)
- **Claude** — installed and on your `PATH` (optional, for the `claude` backend)

## Install

```bash
make install
```

This installs the Go binary and [scripts/git-ai](scripts/git-ai) into `$(BINDIR)` (default `~/.local/bin`). Ensure that directory is on your `PATH`.

## Backends

Set `GIT_AI_BACKEND` to choose which provider generates the commit message:

| Value   | Provider | Default |
|---------|----------|---------|
| `codex` | OpenAI Codex CLI | yes |
| `claude` | Anthropic Claude CLI | no |

```bash
# Use the default (codex)
git ai

# Use Claude
GIT_AI_BACKEND=claude git ai
```

## Get started

1. Stage your changes: `git add ...`
2. Run: `git ai` (or `git-ai` if not using a git alias)
3. The selected backend drafts a conventional commit message and opens your editor so you can confirm or edit, then commit.
