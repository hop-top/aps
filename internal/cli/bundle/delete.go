package bundle

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/prompt"
	corebundle "hop.top/aps/internal/core/bundle"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a user-defined bundle (refuses on built-ins)",
		Long: `Remove the user bundle file at aps/bundles/<name>.yaml under
the user config directory. The command refuses to touch built-in
bundles — if the named bundle is a built-in with no user override,
it suggests aps bundle edit to create an override first. The user
is prompted for confirmation unless --force is passed.

Destructive: the user bundle YAML file is removed irreversibly.
The destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would only restate the
bundle name. Idempotent on already-absent records — re-running on
a missing user bundle reports the bundle as not found.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false,
		"Skip confirmation prompt")
	clinote.AddFlag(cmd) // T-1291

	// T-0654 — bundle delete removes the user-bundle file on disk;
	// irreversible local mutation. Delete-by-name is idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	kitcli.SetDestructiveToken(cmd)
	// T-0656 — destructive-token confirm already gates the apply path;
	// preview would only restate the user-bundle path being removed.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "delete removes the user-bundle file on disk; the destructive-token confirm flow already requires explicit acknowledgement, and preview would only restate the bundle name."); err != nil {
		panic(err)
	}
	return cmd
}

func runDelete(name string, force bool) error {
	path, err := userBundlePath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Not a user bundle — check if it's a built-in.
		if isBuiltin(name) {
			return fmt.Errorf(
				"cannot delete built-in bundle; use 'aps bundle edit' to create a user override")
		}
		return fmt.Errorf("bundle %q not found", name)
	}

	if !force {
		confirmed, err := prompt.Confirm(
			fmt.Sprintf("Delete user bundle %q at %s?", name, path))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(dimStyle.Render("Aborted."))
			return nil
		}
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete bundle file: %w", err)
	}

	fmt.Printf("%s bundle %q\n", successStyle.Render("Deleted"), name)
	return nil
}

func isBuiltin(name string) bool {
	builtins, err := corebundle.LoadBuiltins()
	if err != nil {
		return false
	}
	for _, b := range builtins {
		if b.Name == name {
			return true
		}
	}
	return false
}
