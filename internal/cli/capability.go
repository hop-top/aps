package cli

import "hop.top/aps/internal/cli/capability"

func init() {
	rootCmd.AddCommand(capability.NewCapabilityCmd())
}
