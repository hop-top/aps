package cli

import "hop.top/aps/internal/cli/policy"

func init() {
	rootCmd.AddCommand(policy.NewPolicyCmd())
}
