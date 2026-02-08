---
name: git-conventional-commit
description: Generate Conventional Commit messages from staged git changes using Conventional Commits 1.0.0. Use when asked to draft, infer, or validate commit messages; when summarizing staged diffs; or when the user wants Conventional Commits formatted output.
---

# Git Conventional Commit

## Overview

Generate Conventional Commit messages for staged changes. Use the helper script or follow the manual workflow. Keep the subject concise and imperative.

## Workflow

1. Verify there are staged changes (git diff --staged).
2. Determine the type and optional scope.
3. Write the summary line: type(scope)!: summary
4. Add a body if needed (what and why).
5. Add a footer for breaking changes (BREAKING CHANGE: ...).

## Types

Use one of these Conventional Commit types:
- feat: new feature
- fix: bug fix
- docs: documentation only
- style: formatting only (no logic changes)
- refactor: code change that is not a fix or feature
- perf: performance improvement
- test: add or update tests
- build: build system or dependencies
- ci: CI configuration and scripts
- chore: other changes that do not modify src or tests
- revert: revert a previous commit

## Helper Binary

Use scripts/git_ai_commit.go (or the compiled git-ai binary) to draft a message from staged changes, or scripts/propose_commit_message.ps1 to auto-run it from the current repo.

Build the binary once and keep it on PATH:

~~~
go build -o "$(go env GOPATH)/bin/git-ai" scripts/git_ai_commit.go
~~~

~~~
git-ai
git-ai -type fix -scope api -summary "handle nil user"
git-ai -breaking -breaking-desc "remove legacy auth"
git-ai -use-codex -skill-path "C:\Users\Daniel\.codex\skills\local\git-conventional-commit\SKILL.md"
~~~

Flags:
- codex-cmd: codex command name or path (default: codex)
- codex-args: args for codex invocation (default: exec --skip-git-repo-check --json)
- skill-path: path to SKILL.md to append after the built-in Conventional Commits 1.0.0 spec
- no-spinner: disable the CLI spinner while Codex runs

The staged diff is provided to Codex and the output is used as the full commit message.

### Git Alias

Configure a git alias that pipes the suggested message into the commit editor:

~~~
git config --global alias.ai '!git-ai | git commit -F - --edit'
~~~

Use a skill path if you want to load these instructions explicitly:

~~~
git config --global alias.ai '!git-ai -skill-path "C:\Users\Daniel\.codex\skills\local\git-conventional-commit\SKILL.md" | git commit -F - --edit'
~~~

## Validation

- Use lower-case summary text and avoid a trailing period.
- Keep the summary under 72 characters when possible.
- Only use ! when introducing a breaking change.

## References

Read references/conventional-commits.md for the core specification details and examples.


