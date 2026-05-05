package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
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

	return cmd
}
