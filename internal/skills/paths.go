package skills

import (
	"os"
	"path/filepath"
	"runtime"
)

// SkillPaths manages hierarchical skill discovery paths
type SkillPaths struct {
	// Profile-specific path (highest priority)
	ProfilePath string

	// Global APS skills path
	GlobalPath string

	// User-configured paths from config
	UserPaths []string

	// Auto-detected IDE/TDE paths (opt-in)
	DetectedPaths []string
}

// NewSkillPaths creates a new SkillPaths with default locations
func NewSkillPaths(profileID string) *SkillPaths {
	return &SkillPaths{
		ProfilePath:   getProfileSkillsPath(profileID),
		GlobalPath:    getGlobalSkillsPath(),
		UserPaths:     []string{},
		DetectedPaths: []string{},
	}
}

// getProfileSkillsPath returns profile-specific skills directory
func getProfileSkillsPath(profileID string) string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".agents", "profiles", profileID, "skills")
}

// getGlobalSkillsPath returns global APS skills directory using XDG
func getGlobalSkillsPath() string {
	var dataHome string

	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		// XDG_DATA_HOME or ~/.local/share
		dataHome = os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			homeDir, _ := os.UserHomeDir()
			dataHome = filepath.Join(homeDir, ".local", "share")
		}

	case "darwin":
		// macOS: ~/Library/Application Support
		homeDir, _ := os.UserHomeDir()
		dataHome = filepath.Join(homeDir, "Library", "Application Support")

	case "windows":
		// Windows: %LOCALAPPDATA%
		dataHome = os.Getenv("LOCALAPPDATA")
		if dataHome == "" {
			homeDir, _ := os.UserHomeDir()
			dataHome = filepath.Join(homeDir, "AppData", "Local")
		}

	default:
		// Fallback to ~/.local/share
		homeDir, _ := os.UserHomeDir()
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataHome, "aps", "skills")
}

// AllPaths returns all skill paths in priority order (high to low)
func (sp *SkillPaths) AllPaths() []string {
	paths := []string{sp.ProfilePath}
	paths = append(paths, sp.GlobalPath)
	paths = append(paths, sp.UserPaths...)
	paths = append(paths, sp.DetectedPaths...)
	return paths
}

// DetectIDEPaths scans for IDE/TDE skill directories
func (sp *SkillPaths) DetectIDEPaths() []string {
	homeDir, _ := os.UserHomeDir()

	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			// Claude Code
			filepath.Join(homeDir, ".claude", "skills"),

			// Cursor
			filepath.Join(homeDir, ".cursor", "skills"),
			filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "skills"),

			// VS Code
			filepath.Join(homeDir, ".vscode", "skills"),
			filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "skills"),

			// Zed
			filepath.Join(homeDir, ".config", "zed", "skills"),

			// Windsurf
			filepath.Join(homeDir, ".windsurf", "skills"),
		}

	case "linux", "freebsd", "openbsd", "netbsd":
		candidates = []string{
			// Claude Code
			filepath.Join(homeDir, ".claude", "skills"),

			// Cursor
			filepath.Join(homeDir, ".cursor", "skills"),
			filepath.Join(homeDir, ".config", "Cursor", "User", "skills"),

			// VS Code
			filepath.Join(homeDir, ".vscode", "skills"),
			filepath.Join(homeDir, ".config", "Code", "User", "skills"),

			// Zed
			filepath.Join(homeDir, ".config", "zed", "skills"),

			// Windsurf
			filepath.Join(homeDir, ".windsurf", "skills"),
		}

	case "windows":
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")

		candidates = []string{
			// Claude Code
			filepath.Join(homeDir, ".claude", "skills"),

			// Cursor
			filepath.Join(homeDir, ".cursor", "skills"),
			filepath.Join(appData, "Cursor", "User", "skills"),

			// VS Code
			filepath.Join(homeDir, ".vscode", "skills"),
			filepath.Join(appData, "Code", "User", "skills"),

			// Zed
			filepath.Join(localAppData, "Zed", "skills"),

			// Windsurf
			filepath.Join(homeDir, ".windsurf", "skills"),
		}
	}

	// Filter to only existing directories
	var detected []string
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			detected = append(detected, path)
		}
	}

	return detected
}

// SuggestIDEPaths returns IDE paths that could be configured but aren't yet
func (sp *SkillPaths) SuggestIDEPaths() []string {
	detected := sp.DetectIDEPaths()

	// Filter out already configured paths
	suggestions := []string{}
	configured := make(map[string]bool)
	for _, p := range sp.UserPaths {
		configured[p] = true
	}
	for _, p := range sp.DetectedPaths {
		configured[p] = true
	}

	for _, path := range detected {
		if !configured[path] {
			suggestions = append(suggestions, path)
		}
	}

	return suggestions
}
