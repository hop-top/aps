package cli

import "hop.top/aps/internal/cli/identity"

func init() {
	rootCmd.AddCommand(identity.NewIdentityCmd())
}
