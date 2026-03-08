package cli

import "hop.top/aps/internal/cli/bundle"

func init() {
	rootCmd.AddCommand(bundle.NewBundleCmd())
}
