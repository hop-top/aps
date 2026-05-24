package a2a

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// NewCardCmd returns the `a2a card` mid-level command grouping
// agent card operations (show, fetch).
func NewCardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Manage A2A agent cards",
		Long:  `Show local profile cards or fetch remote agent cards.`,
	}

	// T-0648 — intermediate grouping node; leaves below sit at depth 3.
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(NewShowCardCmd())
	cmd.AddCommand(NewFetchCardCmd())

	return cmd
}
