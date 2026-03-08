package cli

import (
	"hop.top/aps/internal/cli/workspace"
)

func init() {
	rootCmd.AddCommand(workspace.NewWorkspaceCmd())
}
