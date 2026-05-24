package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a capability",
		Long: `Remove the on-disk capability directory at
$APS_DATA_PATH/capabilities/<name>/ along with any symlinks it
holds. Builtins cannot be deleted — use aps capability disable on a
profile to drop the linkage. If the capability has active links to
external targets the command refuses unless --force is passed, since
deleting an active source leaves dangling symlinks at the targets.

Destructive: the on-disk directory and its symlinks are removed
irreversibly. The destructive-token confirmation flow gates the
apply path, and --dry-run is opted out because preview would only
restate the capability name. Idempotent on already-absent records.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cap, err := capability.LoadCapability(name)
			if err != nil {
				return fmt.Errorf("capability '%s' not found", name)
			}

			if len(cap.Links) > 0 && !force {
				fmt.Println(warnStyle.Render(fmt.Sprintf(
					"Warning: '%s' has %d active links that will break.",
					name, len(cap.Links))))
				fmt.Println(dimStyle.Render("  Use --force to delete anyway."))
				return nil
			}

			if err := capability.Delete(name); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Deleted") + " " +
				boldStyle.Render(name))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip link warning")
	clinote.AddFlag(cmd) // T-1291

	// T-0654 — capability delete is an irreversible local mutation
	// (removes the on-disk capability + its symlinks); delete-by-name
	// is naturally idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	kitcli.SetDestructiveToken(cmd)
	// T-0656 — destructive-token confirm already gates the irreversible
	// disk delete; preview would only restate the capability name.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "delete removes the on-disk capability directory and its symlinks; the destructive-token confirm flow already gates the apply path, and preview would only restate the capability name."); err != nil {
		panic(err)
	}
	return cmd
}
