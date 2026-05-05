package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	kitcliconfig "hop.top/kit/go/console/cli/config"
	"hop.top/kit/go/core/xdg"
)

// configCmd is the parent for `aps config <subcommand>`. It hosts kit's
// shared `config path` and `config paths` subcommands so aps participates
// in the §7.4 cross-tool convention (`<tool> config path|paths`). See
// ~/.ops/docs/cli-conventions-with-kit.md and kit/go/console/cli/config.
//
// T-0457 — wire kit's RegisterPathSubcommands. Other config concerns
// (validate, set, get) are intentionally out of scope here; LoadConfig in
// internal/core already handles config loading.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect aps configuration",
	Args:  cobra.NoArgs,
}

// apsProjectMarkers lists the per-directory config files aps recognises,
// in highest-precedence-first order. Mirrors what LoadConfig consumes
// (.aps.yaml is the canonical project config; .aps/config.yaml and
// .hop/aps/config.yaml mirror FindLocalConfigDir layouts).
var apsProjectMarkers = []string{
	filepath.Join(".aps", "config.yaml"),
	filepath.Join(".hop", "aps", "config.yaml"),
	".aps.yaml",
}

// apsConfigPathsResolver returns the aps config precedence chain for cwd,
// highest-precedence first. Order: cwd marker(s) → walk-up to project
// root → user (`$XDG_CONFIG_HOME/aps/config.yaml`) → system
// (`/etc/aps/config.yaml`) → synthetic "default" entry. Walk-up stops at
// $HOME so user-scope discovery never escapes ~.
func apsConfigPathsResolver(cwd string) []kitcliconfig.ResolvedPath {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}

	out := make([]kitcliconfig.ResolvedPath, 0, len(apsProjectMarkers)*4+3)

	// 1) cwd marker(s).
	for _, m := range apsProjectMarkers {
		p := filepath.Join(abs, m)
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   p,
			Source: "cwd",
			Exists: regularFileExists(p),
		})
	}

	// 2) Walk up looking for project root. Stop at fs root or $HOME.
	home, _ := os.UserHomeDir()
	dir := abs
	const maxDepth = 32
	for depth := 0; depth < maxDepth; depth++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if home != "" && dir == home {
			break
		}
		for _, m := range apsProjectMarkers {
			p := filepath.Join(dir, m)
			out = append(out, kitcliconfig.ResolvedPath{
				Path:   p,
				Source: "project",
				Exists: regularFileExists(p),
			})
		}
	}

	// 3) User config: $XDG_CONFIG_HOME/aps/config.yaml.
	if userDir, err := xdg.ConfigDir("aps"); err == nil && userDir != "" {
		userPath := filepath.Join(userDir, "config.yaml")
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   userPath,
			Source: "user",
			Exists: regularFileExists(userPath),
		})
	}

	// 4) System config: /etc/aps/config.yaml.
	sysPath := filepath.Join("/etc", "aps", "config.yaml")
	out = append(out, kitcliconfig.ResolvedPath{
		Path:   sysPath,
		Source: "system",
		Exists: regularFileExists(sysPath),
	})

	// 5) Synthetic defaults entry — represents in-binary fallbacks.
	out = append(out, kitcliconfig.ResolvedPath{
		Path:   "<defaults>",
		Source: "default",
		Exists: true,
	})

	return out
}

// regularFileExists is a tiny helper around os.Stat that returns true
// only for regular files (and followed symlinks), false otherwise.
func regularFileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func init() {
	kitcliconfig.RegisterPathSubcommands(
		configCmd, "aps",
		kitcliconfig.WithResolver(apsConfigPathsResolver),
	)
	rootCmd.AddCommand(configCmd)
}
