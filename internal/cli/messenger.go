package cli

import (
	"oss-aps-cli/internal/cli/device"
)

func init() {
	rootCmd.AddCommand(device.NewMessengerCmd())
}
