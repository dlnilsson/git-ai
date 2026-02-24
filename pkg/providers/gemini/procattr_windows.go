//go:build windows

package gemini

import "os/exec"

func setProcessGroup(_ *exec.Cmd) {}
