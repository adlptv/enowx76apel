// Package daemon runs the enowx server as a detached background process and
// controls it via a PID file, so the CLI can start/stop/restart a headless
// instance (e.g. on a VPS). Foreground running is unaffected — the daemon is
// opt-in via `enx start --daemon`.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// envMarker is set on the forked child so it knows to run the server rather than
// re-dispatch CLI commands.
const envMarker = "ENOWX_DAEMON"

// IsDaemon reports whether this process is the detached server child.
func IsDaemon() bool { return os.Getenv(envMarker) == "1" }

// pidPath / logPath live under the runtime dir.
func pidPath(runtimeDir string) string { return filepath.Join(runtimeDir, "enx.pid") }
func logPath(runtimeDir string) string { return filepath.Join(runtimeDir, "enx.log") }

// Start forks a detached copy of this binary that runs the server in the
// background, writes its PID, and returns the pid. extraArgs are passed through
// (e.g. so the child re-runs as `enx start`).
func Start(runtimeDir string) (int, error) {
	if IsRunning(runtimeDir) {
		pid, _ := GetPID(runtimeDir)
		return pid, fmt.Errorf("already running (pid %d)", pid)
	}
	self, err := os.Executable()
	if err != nil {
		return 0, err
	}
	logFile, err := os.OpenFile(logPath(runtimeDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log: %w", err)
	}
	defer logFile.Close()

	// The child runs the server: pass `start` and the daemon env marker.
	cmd := exec.Command(self, "start")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), envMarker+"=1")
	cmd.SysProcAttr = daemonSysProcAttr() // detach (new session / detached process)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start daemon: %w", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath(runtimeDir), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return 0, fmt.Errorf("write pid: %w", err)
	}
	// Let the parent exit; the child is reparented to init.
	_ = cmd.Process.Release()
	return pid, nil
}

// Stop asks the daemon to exit cleanly (SIGTERM), then force-kills if it doesn't
// exit within a grace period. Removes the PID file.
func Stop(runtimeDir string) error {
	pid, err := GetPID(runtimeDir)
	if err != nil {
		return fmt.Errorf("not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidPath(runtimeDir))
		return fmt.Errorf("not running")
	}
	_ = requestStop(pid, proc)

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(runtimeDir) {
			_ = os.Remove(pidPath(runtimeDir))
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = forceKill(pid, proc)
	time.Sleep(300 * time.Millisecond)
	_ = os.Remove(pidPath(runtimeDir))
	if IsRunning(runtimeDir) {
		return fmt.Errorf("process %d still running after force stop", pid)
	}
	return nil
}

// IsRunning reports whether the daemon PID is a live process.
func IsRunning(runtimeDir string) bool {
	pid, err := GetPID(runtimeDir)
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return isProcessAlive(pid)
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// GetPID reads the daemon's pid from the PID file.
func GetPID(runtimeDir string) (int, error) {
	b, err := os.ReadFile(pidPath(runtimeDir))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

// WritePID records the current process as the running server (used when the
// server starts, foreground or daemon child, so `enx status/stop` can find it).
func WritePID(runtimeDir string) {
	_ = os.WriteFile(pidPath(runtimeDir), []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// RemovePID clears the PID file (call on clean shutdown).
func RemovePID(runtimeDir string) { _ = os.Remove(pidPath(runtimeDir)) }
