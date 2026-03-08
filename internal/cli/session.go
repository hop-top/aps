package cli

import (
	"hop.top/aps/internal/cli/session"

	"github.com/spf13/cobra"
)

func init() {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
	}

	sessionCmd.AddCommand(session.NewListCmd())
	sessionCmd.AddCommand(session.NewAttachCmd())
	sessionCmd.AddCommand(session.NewDetachCmd())
	sessionCmd.AddCommand(session.NewInspectCmd())
	sessionCmd.AddCommand(session.NewLogsCmd())
	sessionCmd.AddCommand(session.NewTerminateCmd())
	sessionCmd.AddCommand(session.NewDeleteCmd())

	rootCmd.AddCommand(sessionCmd)
}
