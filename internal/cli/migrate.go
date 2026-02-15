package cli

import "oss-aps-cli/internal/cli/migrate"

func init() {
	rootCmd.AddCommand(migrate.NewMigrateCmd())
}
