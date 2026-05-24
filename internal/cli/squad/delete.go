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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0])
		},
	}
	clinote.AddFlag(cmd) // T-1291

	// T-0648 — kit 0.4 signature annotations. Squad deletion mutates
	// the local squad store and is naturally idempotent (delete-by-id).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runDelete(id string) error {
	if err := defaultManager.Delete(id); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleted squad %q\n", id)
	return nil
}
