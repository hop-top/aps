package core

import (
	"os"
	"path/filepath"
)

// FindLocalConfigDir searches for local config directories.
// Priority: .aps/ > .hop/aps/
// Walks up from cwd to root.
func FindLocalConfigDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		// Check .aps/ first (direct APS usage)
		if candidate := filepath.Join(dir, ".aps"); isDir(candidate) {
			return candidate
		}
		// Check .hop/aps/ (when running via hop)
		if candidate := filepath.Join(dir, ".hop", "aps"); isDir(candidate) {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// InitLocalConfig creates a .aps/ directory in the current directory.
func InitLocalConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	localDir := filepath.Join(dir, ".aps")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", err
	}

	return localDir, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
