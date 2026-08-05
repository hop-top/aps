package cli

import "hop.top/aps/internal/cli/org"

func init() {
	rootCmd.AddCommand(org.NewOrgCmd())
}
