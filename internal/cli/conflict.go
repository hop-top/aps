package cli

import "hop.top/aps/internal/cli/conflict"

func init() {
	rootCmd.AddCommand(conflict.NewConflictCmd())
}
