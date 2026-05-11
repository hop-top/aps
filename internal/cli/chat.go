package cli

import "hop.top/aps/internal/cli/chat"

func init() {
	rootCmd.AddCommand(chat.NewCommand())
}
