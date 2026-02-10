package providers

import (
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

type Registry struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	stopSpinner func()
	interrupted bool
}

func (r *Registry) Register(cmd *exec.Cmd, stopSpinner func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmd = cmd
	r.stopSpinner = stopSpinner
	r.interrupted = false
}

func (r *Registry) Unregister() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmd = nil
	r.stopSpinner = nil
}

func (r *Registry) WasInterrupted() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.interrupted
}

func (r *Registry) ForwardSignal(sig os.Signal) {
	r.mu.Lock()
	cmd := r.cmd
	if sig == os.Interrupt {
		r.interrupted = true
	}
	r.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS != "windows" && (sig == os.Interrupt || sig == syscall.SIGTERM) {
		_ = syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal))
		return
	}
	_ = cmd.Process.Signal(sig)
}

func (r *Registry) StopSpinnerIfSet() {
	r.mu.Lock()
	stop := r.stopSpinner
	r.stopSpinner = nil
	r.mu.Unlock()
	if stop != nil {
		stop()
	}
}
