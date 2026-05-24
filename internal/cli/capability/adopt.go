package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newAdoptCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "adopt <path> --name <name>",
		Short: "Adopt an existing file/dir (move to APS + symlink back)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if err := capability.Adopt(args[0], name); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Adopted") + " " +
				args[0] + " as " + boldStyle.Render(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of capability")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — adopt copies the source bundle into the user capability
	// dir; the destination path is a pure function of the --name flag.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "adopt copies the source bundle into the user capability directory under a path derived from --name; previewing would only restate that path."); err != nil {
		panic(err)
	}
	return cmd
}
