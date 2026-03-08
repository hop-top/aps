package capability

import (
	"fmt"

	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
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

	return cmd
}
