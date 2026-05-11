package service

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <service-id>",
		Short: "Show a persisted service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "id: %s\n", service.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "type: %s\n", service.Type)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "adapter: %s\n", service.Adapter)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", service.Profile)
			if service.Description != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "description: %s\n", service.Description)
			}
			return nil
		},
	}
}
