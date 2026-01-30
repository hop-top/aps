package cli

import (
	"oss-aps-cli/internal/cli/a2a"
)

func init() {
	rootCmd.AddCommand(a2a.NewA2ACmd())
}
