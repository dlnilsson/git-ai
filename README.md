# git ai

[![asciicast](https://asciinema.org/a/uHPdXi9wsZ23xQ42.svg)](https://asciinema.org/a/uHPdXi9wsZ23xQ42)

Generates conventional commit messages from your staged changes using Claude or Codex. Use the generated message with `git commit` (e.g. via the `git-ai` script) and optionally edit it in your editor before committing.

## Requirements

- **Go** — to build the binary
- **Claude** or **Codex** — installed and on your `PATH`

## Install

macOS/Linux:

```bash
make install
```

This installs the Go binary and [scripts/git-ai](scripts/git-ai) into `$(BINDIR)` (default `~/.local/bin`). Ensure that directory is on your `PATH`.

Windows (PowerShell):

```powershell
./scripts/install-windows.ps1
```

This installs the Go binary plus `git-ai.cmd`/`git-ai.ps1` into `$HOME\.local\bin` by default. Ensure that directory is on your `PATH`.

Configure a Git alias on Windows so `git ai` works:

```powershell
git config --global alias.ai "!git-ai"
```

Verify:

```powershell
git config --global --get alias.ai
```

Expected output:

```text
!git-ai
```

## Backends

The backend is auto-detected from your `PATH` (Claude preferred). Override with `GIT_AI_BACKEND`:

| Value    | Provider             |
|----------|----------------------|
| `claude` | Anthropic Claude CLI |
| `codex`  | OpenAI Codex CLI     |

```bash
# Auto-detect (claude preferred, falls back to codex)
git ai

# Force a specific backend
GIT_AI_BACKEND=codex git ai
```

PowerShell backend override:

```powershell
$env:GIT_AI_BACKEND='codex'; git ai
```

## Get started

1. Stage your changes: `git add ...`
2. Run: `git ai` (or `git-ai` if not using a git alias)
3. The backend drafts a conventional commit message and opens your editor so you can confirm or edit, then commit.
