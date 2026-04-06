//go:build windows

package adapter

import (
	"os"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func termSignal() os.Signal { return os.Kill }
func killSignal() os.Signal { return os.Kill }
