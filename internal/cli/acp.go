package cli

import (
	"hop.top/aps/internal/cli/acp"
)

func init() {
	rootCmd.AddCommand(acp.NewACPCmd())
}
