package cli

import "hop.top/aps/internal/cli/squad"

func init() {
	rootCmd.AddCommand(squad.NewSquadCmd())
}
