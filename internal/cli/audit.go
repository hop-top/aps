package cli

import "hop.top/aps/internal/cli/audit"

func init() {
	rootCmd.AddCommand(audit.NewAuditCmd())
}
