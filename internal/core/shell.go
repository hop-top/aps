package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DetectShell attempts to find the user's preferred shell
func DetectShell() string {
	// 1. Check SHELL environment variable (standard on Unix)
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// 2. Fallback based on OS
	if runtime.GOOS == "windows" {
		// Check COMSPEC (usually cmd.exe)
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			return comspec
		}
		return "powershell.exe" // Reasonable modern default
	}

	// 3. Fallback for Unix-like
	// Try standard paths
	candidates := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return "sh" // Last resort
}

// IsCommandAvailable checks if a command name exists in the system PATH
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetShellName returns the base name of the shell (e.g., "zsh" from "/bin/zsh")
func GetShellName(path string) string {
	return filepath.Base(path)
}
