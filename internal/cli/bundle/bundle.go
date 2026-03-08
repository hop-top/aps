package bundle

import "github.com/spf13/cobra"

// NewBundleCmd creates the bundle command group.
func NewBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Aliases: []string{"bundles"},
		Short:   "Manage capability bundles",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newValidateCmd())

	return cmd
}
