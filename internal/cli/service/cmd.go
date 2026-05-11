package service

import "github.com/spf13/cobra"

func NewServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage profile-facing services",
	}

	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newRoutesCmd())

	return cmd
}
