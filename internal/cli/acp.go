package cli

import (
	"oss-aps-cli/internal/cli/acp"
)

func init() {
	rootCmd.AddCommand(acp.NewACPCmd())
}
