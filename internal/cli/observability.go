package cli

import "hop.top/aps/internal/cli/observability"

func init() {
	rootCmd.AddCommand(observability.NewObservabilityCmd())
}
