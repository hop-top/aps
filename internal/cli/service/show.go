package service

import (
	"fmt"
	"sort"

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
			if len(service.Options) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "options:")
				keys := make([]string, 0, len(service.Options))
				for key := range service.Options {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					value := service.Options[key]
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", key, value)
				}
			}
			runtime := core.DescribeServiceRuntime(service)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "receives: %s\n", runtime.Receives)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "executes: %s\n", runtime.Executes)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "replies: %s\n", runtime.Replies)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "maturity: %s\n", runtime.Maturity)
			return nil
		},
	}
}

func newRoutesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "routes <service-id>",
		Short: "Show reachable routes for a persisted service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			runtime := core.DescribeServiceRuntime(service)
			if len(runtime.Routes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "routes: none")
				return nil
			}
			for _, route := range runtime.Routes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", route)
			}
			return nil
		},
	}
}
