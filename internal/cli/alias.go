// Package cli — alias.go wires both the shell-alias *generator* (legacy
// `aps alias`, now `aps alias shell`) and the kit/console/alias YAML
// user-shorthand *store* under a single `alias` parent command.
//
// Two orthogonal concerns share the verb:
//
//   - aps alias shell     → emit shell-source-able alias lines for every
//                           profile (eval "$(aps alias shell)").
//   - aps alias add/list/remove → manage user shorthands stored at
//                           $XDG_CONFIG_HOME/aps/aliases.yaml.
//
// T-0388.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/alias"
	"hop.top/kit/go/core/xdg"
)

// aliasShellCmd is the legacy shell-alias generator, now nested under
// `alias` as `alias shell`. Behaviour preserved verbatim from the
// pre-T-0388 top-level `alias` command.
var aliasShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Generate shell aliases for profiles",
	Long: `Generate shell aliases for all available profiles.
Add the following to your shell configuration file:

  eval "$(aps alias shell)"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := core.ListProfiles()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		// Detect shell to decide alias format
		shellPath := core.DetectShell()
		shellName := core.GetShellName(shellPath)

		for _, p := range profiles {
			// Check for conflicts
			if core.IsCommandAvailable(p) {
				fmt.Fprintf(os.Stderr, "WARNING: Skipping alias for '%s' because a command with that name already exists in PATH.\n", p)
				continue
			}

			// Format alias based on shell
			// For now, standard POSIX alias works for zsh/bash/fish
			switch shellName {
			case "powershell", "pwsh":
				fmt.Printf("function %s { aps %s @args }\n", p, p)
			default:
				fmt.Printf("alias %s='aps %s'\n", p, p)
			}
		}
		return nil
	},
}

// newAliasStore returns a kit alias.Store rooted at
// $XDG_CONFIG_HOME/aps/aliases.yaml. Errors propagate so the caller can
// abort wiring rather than silently ignoring config dir resolution
// failures.
func newAliasStore() (*alias.Store, error) {
	dir, err := xdg.ConfigDir("aps")
	if err != nil {
		return nil, fmt.Errorf("resolve aps config dir: %w", err)
	}
	store := alias.NewStore(filepath.Join(dir, "aliases.yaml"))
	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("load aps aliases: %w", err)
	}
	return store, nil
}

func init() {
	store, err := newAliasStore()
	if err != nil {
		// Surface the failure on stderr but don't crash CLI startup; the
		// alias subcommand simply won't be registered. Other commands
		// remain available.
		fmt.Fprintf(os.Stderr, "aps: alias store unavailable: %v\n", err)
		return
	}

	aliasCmd := root.AliasCmd(store)
	// Deprecation breadcrumb for users who relied on `aps alias` being
	// the shell generator. Removed after one release.
	aliasCmd.Long = `Manage user-defined command aliases (YAML-backed) and
generate per-profile shell aliases.

Subcommands:
  add | list | remove   Manage YAML aliases at $XDG_CONFIG_HOME/aps/aliases.yaml
  shell                 Print shell-source-able alias lines (legacy ` + "`aps alias`" + `)

Note: the bare ` + "`aps alias`" + ` form previously printed shell aliases.
That behaviour has moved to ` + "`aps alias shell`" + `; this notice will be
removed after one release.`

	aliasCmd.AddCommand(aliasShellCmd)
	rootCmd.AddCommand(aliasCmd)
}
