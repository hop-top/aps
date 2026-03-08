package cli

import "hop.top/aps/internal/cli/adapter"

func init() {
	rootCmd.AddCommand(adapter.NewAdapterCmd())
}
