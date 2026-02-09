BINDIR ?= $(HOME)/.local/bin

.PHONY: install
install:
	go install ./cmd/git-cc-ai
	install -d -m 755 "$(BINDIR)"
	install -m 755 scripts/git-ai "$(BINDIR)/git-ai"
