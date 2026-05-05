//go:build !windows

package voice

import "syscall"

// detachSysProcAttr returns SysProcAttr for spawning a backend that
// outlives the CLI invocation: a new process group so signals to the
// parent don't propagate to the child.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
