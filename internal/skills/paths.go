package skills

import (
	"os"
	"path/filepath"
	"runtime"

	"hop.top/aps/internal/core"
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
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return ""
	}
	return filepath.Join(profileDir, "skills")
}

// getGlobalSkillsPath returns global APS skills directory.
//
// Delegates to core.GetDataDir() so the lookup honours
// $APS_DATA_PATH > $XDG_DATA_HOME/aps > ~/.local/share/aps on every
// platform, matching the rest of the aps data layout (see the
// JUSTIFIED note in internal/core/paths.go). Previously this picked
// OS-native data dirs (~/Library/Application Support on darwin,
// %LOCALAPPDATA% on windows), which silently ignored the env vars
// that the rest of aps respects — surfaced by T-0677 when the
// skills_test fixture only worked on Linux.
func getGlobalSkillsPath() string {
	dataDir, err := core.GetDataDir()
	if err != nil {
		// Best-effort fallback if home cannot be resolved. Keeping
		// the literal join here matches GetDataDir's own structure.
		return filepath.Join("aps", "skills")
	}
	return filepath.Join(dataDir, "skills")
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
	detected := []string{}
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
