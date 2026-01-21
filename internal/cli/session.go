package cli

import (
	"oss-aps-cli/internal/cli/session"

	"github.com/spf13/cobra"
)

func init() {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
	}

	sessionCmd.AddCommand(session.NewListCmd())
	rootCmd.AddCommand(sessionCmd)
}
