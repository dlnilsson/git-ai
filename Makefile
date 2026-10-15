BINDIR ?= $(HOME)/.local/bin
MANDIR ?= $(HOME)/.local/share/man/man1

.PHONY: install
install:
	go install ./cmd/git-cc-ai
	install -d -m 755 "$(BINDIR)"
	install -m 755 scripts/git-ai "$(BINDIR)/git-ai"
	install -d -m 755 "$(MANDIR)"
	{ printf '.TH GIT-AI 1\n.SH NAME\ngit-ai \\- generate commit messages from staged changes\n.SH DESCRIPTION\n.nf\n'; \
	  go run ./cmd/git-cc-ai --help 2>&1 | sed -e 's/\\/\\e/g' -e 's/—/\\-/g' -e "s/^[.']/\\\\\&&/"; \
	  printf '.fi\n'; } > "$(MANDIR)/git-ai.1"
