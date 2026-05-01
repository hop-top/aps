package a2a

import "github.com/spf13/cobra"

// NewCardCmd returns the `a2a card` mid-level command grouping
// agent card operations (show, fetch).
func NewCardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Manage A2A agent cards",
		Long:  `Show local profile cards or fetch remote agent cards.`,
	}

	cmd.AddCommand(NewShowCardCmd())
	cmd.AddCommand(NewFetchCardCmd())

	return cmd
}
