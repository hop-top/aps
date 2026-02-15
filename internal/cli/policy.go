package cli

import "oss-aps-cli/internal/cli/policy"

func init() {
	rootCmd.AddCommand(policy.NewPolicyCmd())
}
