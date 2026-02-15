package cli

import (
	"oss-aps-cli/internal/cli/workspaces"
)

func init() {
	rootCmd.AddCommand(workspaces.NewWorkspacesCmd())
}
