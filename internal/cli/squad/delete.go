package squad

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a squad",
		Long: `Remove the named squad from the local squad store. The squad
ID is the slug shown by aps squad list, not the human-readable
name. Member profile records are not touched — deleting a squad
only drops the squad entry and any topology it carried.

Destructive: the squad record is removed irreversibly. The
destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would only restate the
squad ID. Idempotent on the pair. Use --note to attach an audit
reason that flows to the event bus alongside the mutation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0])
		},
	}
	clinote.AddFlag(cmd) // T-1291

	// T-0654 — squad delete is an irreversible local mutation on the
	// squad store. Delete-by-id is idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	kitcli.SetDestructiveToken(cmd)
	// T-0656 — destructive-token confirm already gates the apply path;
	// preview would only restate the squad ID being removed.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "delete removes the squad entry from the local squad store; the destructive-token confirm flow already gates the apply path, and preview would only restate the squad ID."); err != nil {
		panic(err)
	}
	return cmd
}

func runDelete(id string) error {
	if err := defaultManager.Delete(id); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleted squad %q\n", id)
	return nil
}
