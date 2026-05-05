package voice

import (
	"errors"
	"os"
	"syscall"
)

// looksAlive reports whether pid refers to a process the calling user
// can signal. Mirrors kit/console/ps's internal isProcessAlive: signal
// 0 is the canonical liveness probe on POSIX systems. EPERM means the
// process exists but is owned by another user — still alive.
func looksAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}
