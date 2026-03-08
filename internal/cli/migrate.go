package cli

import "hop.top/aps/internal/cli/migrate"

func init() {
	rootCmd.AddCommand(migrate.NewMigrateCmd())
}
