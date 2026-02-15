package cli

import (
	"oss-aps-cli/internal/cli/workspace"
)

func init() {
	rootCmd.AddCommand(workspace.NewWorkspaceCmd())
}
