//go:build !windows

package providers

import (
	"os"
	"os/exec"
	"syscall"
)

func forwardToProcessGroup(cmd *exec.Cmd, sig os.Signal) bool {
	if sig != os.Interrupt && sig != syscall.SIGTERM {
		return false
	}

	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		return false
	}

	_ = syscall.Kill(-cmd.Process.Pid, sysSig)
	return true
}
