package directory

import (
	"github.com/spf13/cobra"
)

// NewDirectoryCmd creates the directory command group.
func NewDirectoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "directory",
		Aliases: []string{"dir"},
		Short:   "Manage AGNTCY Directory registration and discovery",
		Long: `Manage agent registration and discovery via the AGNTCY Directory.

Register profiles to make them discoverable by other agents,
discover agents by capability, and manage directory records.`,
	}

	cmd.AddCommand(NewRegisterCmd())
	cmd.AddCommand(NewDiscoverCmd())
	cmd.AddCommand(NewDeleteCmd())
	cmd.AddCommand(NewShowCmd())

	return cmd
}
