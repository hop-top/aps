package core

import (
	"os"
	"path/filepath"
)

// GetDataDir returns the XDG data directory for APS.
// Priority: $APS_DATA_PATH > $XDG_DATA_HOME/aps > ~/.local/share/aps
func GetDataDir() (string, error) {
	if apsData := os.Getenv("APS_DATA_PATH"); apsData != "" {
		return apsData, nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "aps"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "aps"), nil
}

// GetCacheDir returns the XDG cache directory for APS.
// Priority: $XDG_CACHE_HOME/aps > ~/.cache/aps
func GetCacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "aps"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "aps"), nil
}

// GetStateDir returns the XDG state directory for APS.
// Priority: $XDG_STATE_HOME/aps > ~/.local/state/aps
func GetStateDir() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "aps"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "aps"), nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
