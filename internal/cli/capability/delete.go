package capability

import (
	"fmt"

	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a capability",
		Args:  cobra.ExactArgs(1),
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

	return cmd
}
