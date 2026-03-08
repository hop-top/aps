package cli

import (
	"hop.top/aps/internal/cli/collab"
)

func init() {
	rootCmd.AddCommand(collab.NewCollabCmd())
}
