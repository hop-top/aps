//go:build !windows

package adapter

import (
	"os"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func termSignal() os.Signal { return syscall.SIGTERM }
func killSignal() os.Signal { return syscall.SIGKILL }
