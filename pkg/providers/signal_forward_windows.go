//go:build windows

package providers

import (
	"os"
	"os/exec"
)

func forwardToProcessGroup(_ *exec.Cmd, _ os.Signal) bool {
	return false
}
