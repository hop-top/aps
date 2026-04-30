package cli

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
)

// registerHints wires contextual next-step hints for key commands.
// Lookup key matches `cmd.Name()` or `<parent> <name>` (see
// renderPostRunHintsFor); use the namespaced form for subcommands so
// "list" or "show" don't collide across command groups.
func registerHints(hints *output.HintSet) {
	hints.Register("profile list", output.Hint{
		Message: "Run `aps profile show <id>` for details.",
	})
	// Note: `aps workspace list` is not yet implemented (only `activity`
	// and `sync` exist under workspace). When list lands, register:
	//   hints.Register("workspace list", output.Hint{
	//       Message: "Run `aps workspace show <id>` for details.",
	//   })
}

// renderPostRunHintsFor renders contextual hints after command output.
// Called from the root PersistentPostRunE. Takes the kit root explicitly
// to avoid an init cycle with the package-level `root` symbol.
func renderPostRunHintsFor(cmd *cobra.Command, r *kitcli.Root) {
	format := r.Viper.GetString("format")

	// Build the command path key for hint lookup. For "aps profile list"
	// cmd.Name() is "list" — also try "<parent> <name>" for namespaced
	// hints to avoid cross-group collisions.
	keys := []string{cmd.Name()}
	if cmd.Parent() != nil && cmd.Parent() != r.Cmd {
		keys = append(keys, cmd.Parent().Name()+" "+cmd.Name())
	}

	for _, key := range keys {
		registered := r.Hints.Lookup(key)
		if len(registered) == 0 {
			continue
		}
		output.RenderHints(
			cmd.OutOrStdout(),
			registered,
			format,
			r.Viper,
			r.Theme.Muted,
		)
	}
}
