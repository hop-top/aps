package cli

import "hop.top/aps/internal/cli/service"

func init() {
	rootCmd.AddCommand(service.NewServiceCmd())
}
