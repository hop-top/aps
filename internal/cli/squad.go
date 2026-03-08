package cli

import "oss-aps-cli/internal/cli/squad"

func init() {
	rootCmd.AddCommand(squad.NewSquadCmd())
}
