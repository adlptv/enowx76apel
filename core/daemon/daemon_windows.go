//go:build windows

package daemon

import (
	"os"
	"syscall"
	"unsafe"
)

// daemonSysProcAttr detaches the child (DETACHED_PROCESS) so it keeps running
// after the launching console closes.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008}
}

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = kernel32.NewProc("OpenProcess")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
)

const (
	processQueryLimitedInfo = 0x1000
	stillActive             = 259
)

func isProcessAlive(pid int) bool {
	handle, _, _ := procOpenProcess.Call(processQueryLimitedInfo, 0, uintptr(pid))
	if handle == 0 {
		return false
	}
	defer procCloseHandle.Call(handle)
	var code uint32
	r, _, _ := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&code)))
	return r != 0 && code == stillActive
}

// requestStop / forceKill: Windows has no SIGTERM, so both just kill the process.
func requestStop(_ int, proc *os.Process) error { return proc.Kill() }
func forceKill(_ int, proc *os.Process) error   { return proc.Kill() }
