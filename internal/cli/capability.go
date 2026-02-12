package cli

import "oss-aps-cli/internal/cli/capability"

func init() {
	rootCmd.AddCommand(capability.NewCapabilityCmd())
}
