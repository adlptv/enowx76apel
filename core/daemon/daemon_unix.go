//go:build !windows

package daemon

import (
	"errors"
	"os"
	"syscall"
)

// daemonSysProcAttr detaches the child into its own session so it survives the
// parent (and terminal) exiting.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func isProcessAlive(_ int) bool { return true } // unix uses signal(0); see IsRunning

// requestStop asks the process (group) to shut down cleanly.
func requestStop(pid int, proc *os.Process) error {
	if pid > 0 {
		if err := syscall.Kill(-pid, syscall.SIGTERM); err == nil || errors.Is(err, syscall.ESRCH) {
			return nil
		}
	}
	return proc.Signal(syscall.SIGTERM)
}

// forceKill terminates the process (group) hard.
func forceKill(pid int, proc *os.Process) error {
	if pid > 0 {
		if err := syscall.Kill(-pid, syscall.SIGKILL); err == nil || errors.Is(err, syscall.ESRCH) {
			return nil
		}
	}
	return proc.Kill()
}
