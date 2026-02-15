package cli

import (
	"oss-aps-cli/internal/cli/collab"
)

func init() {
	rootCmd.AddCommand(collab.NewCollabCmd())
}
