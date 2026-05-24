package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
	kitcliconfig "hop.top/kit/go/console/cli/config"
	coreconfig "hop.top/kit/go/core/config"
)

// configCmd is the parent for `aps config <subcommand>`. It hosts kit's
// shared `config path` and `config paths` subcommands so aps participates
// in the §7.4 cross-tool convention (`<tool> config path|paths`). See
// ~/.ops/docs/cli-conventions-with-kit.md and kit/go/console/cli/config.
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
// highest-precedence first. The chain is composed by kit's
// PathsForToolWithMarkers: cwd marker(s) → walk-up to project root (stops
// at $HOME) → user (`$XDG_CONFIG_HOME/aps/config.yaml`) → system
// (`/etc/aps/config.yaml`) → synthetic `<defaults>` entry.
func apsConfigPathsResolver(cwd string) []kitcliconfig.ResolvedPath {
	raw := coreconfig.PathsForToolWithMarkers(cwd, "aps", apsProjectMarkers)
	out := make([]kitcliconfig.ResolvedPath, len(raw))
	for i, r := range raw {
		out[i] = kitcliconfig.ResolvedPath{
			Path:   r.Path,
			Source: r.Source,
			Scope:  r.Scope,
			Exists: r.Exists,
		}
	}
	return out
}

func init() {
	kitcliconfig.RegisterPathSubcommands(
		configCmd, "aps",
		kitcliconfig.WithResolver(apsConfigPathsResolver),
	)
	rootCmd.AddCommand(configCmd)
}
