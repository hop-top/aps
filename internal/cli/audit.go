package cli

import "oss-aps-cli/internal/cli/audit"

func init() {
	rootCmd.AddCommand(audit.NewAuditCmd())
}
