package cli

import (
	"oss-aps-cli/internal/cli/adapter"
)

func init() {
	rootCmd.AddCommand(adapter.NewMessengerCmd())
}
