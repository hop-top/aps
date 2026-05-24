package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newInstallCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "install <source> --name <name>",
		Short: "Install a capability from a source directory or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if err := capability.Install(name, args[0]); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Installed") + " " +
				boldStyle.Render(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the capability")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — install copies the bundled capability into the user dir;
	// the destination is a pure function of the --name flag.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "install copies the bundled capability into the user capability directory under a path derived from --name; previewing would only restate that path."); err != nil {
		panic(err)
	}
	return cmd
}
