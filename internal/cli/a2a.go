package cli

import (
	"hop.top/aps/internal/cli/a2a"
)

func init() {
	rootCmd.AddCommand(a2a.NewA2ACmd())
}
