//go:build windows

package codex

import "os/exec"

func setProcessGroup(_ *exec.Cmd) {}
