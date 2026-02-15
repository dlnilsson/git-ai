//go:build windows

package claude

import "os/exec"

func setProcessGroup(_ *exec.Cmd) {}
