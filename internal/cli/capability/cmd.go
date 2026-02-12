package capability

import "github.com/spf13/cobra"

// NewCapabilityCmd creates the capability command group
func NewCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "capability",
		Aliases: []string{"cap"},
		Short:   "Manage capabilities (tools, configs, dotfiles)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newAdoptCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newPatternsCmd())
	cmd.AddCommand(newEnableCmd())
	cmd.AddCommand(newDisableCmd())

	return cmd
}
