package e2e

import (
	"os"
	"path/filepath"
	"strings"
)

// sandboxEnv returns an environment for invoking the aps binary that
// points HOME and every XDG_*_HOME at home and strips APS_DATA_PATH
// from the parent, so tests cannot read or write the user's real
// ~/.local/share/aps. hop.top/aps/internal/core.GetDataDir resolves to
// $APS_DATA_PATH > $XDG_DATA_HOME/aps > ~/.local/share/aps.
func sandboxEnv(home string) []string {
	return sandboxEnvWith(home, nil)
}

// sandboxEnvWith is sandboxEnv with extra overrides layered on top.
// Keys present in extra always win, including over the sandbox's own
// HOME/XDG defaults (use sparingly).
func sandboxEnvWith(home string, extra map[string]string) []string {
	overridden := map[string]bool{
		"HOME":            true,
		"USERPROFILE":     true,
		"XDG_DATA_HOME":   true,
		"XDG_CONFIG_HOME": true,
		"XDG_CACHE_HOME":  true,
		"XDG_STATE_HOME":  true,
		"APS_DATA_PATH":   true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
	}
	// Replace sandbox defaults if extra overrides them.
	if len(extra) > 0 {
		filtered := env[:0]
		for _, e := range env {
			key, _, _ := strings.Cut(e, "=")
			if _, ok := extra[key]; !ok {
				filtered = append(filtered, e)
			}
		}
		env = filtered
	}
	for k, v := range extra {
		overridden[k] = true
		env = append(env, k+"="+v)
	}
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if !overridden[key] {
			env = append(env, e)
		}
	}
	return env
}
