package cli

import "hop.top/aps/internal/cli/directory"

func init() {
	rootCmd.AddCommand(directory.NewDirectoryCmd())
}
