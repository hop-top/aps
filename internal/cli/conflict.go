package cli

import "oss-aps-cli/internal/cli/conflict"

func init() {
	rootCmd.AddCommand(conflict.NewConflictCmd())
}
