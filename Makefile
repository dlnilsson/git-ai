BINDIR ?= $(HOME)/.local/bin

.PHONY: install
install:
	go install
	install -d -m 755 "$(BINDIR)"
	install -m 755 scripts/git-ai "$(BINDIR)/git-ai"
