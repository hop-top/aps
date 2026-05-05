//go:build windows

package voice

import "syscall"

// detachSysProcAttr is a no-op on Windows; process group semantics
// differ and the upstream adapter manager uses an empty attr too.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
