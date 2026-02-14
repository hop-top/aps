package webhook

import (
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
)

// NewServerCmd creates the `aps webhook server` command
func NewServerCmd() *cobra.Command {
	var (
		addr      string
		secret    string
		dryRun    bool
		eventMaps []string
		allowList []string
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a webhook server",
		Long: `Start a webhook server to receive and process webhook events.

The server listens on the specified address and processes incoming webhook
events by mapping them to profile actions.

Examples:
  aps webhook server
  aps webhook server --addr 0.0.0.0:9000
  aps webhook server --secret my-secret --event-map github=profile:action`,
		RunE: func(cmd *cobra.Command, args []string) error {
			eventMap := make(map[string]string)
			for _, m := range eventMaps {
				// Parse event map (event=profile:action)
				var event, mapping string
				fmt.Sscanf(m, "%[^=]=%s", &event, &mapping)
				if event == "" || mapping == "" {
					return fmt.Errorf("invalid event-map format '%s', expected event=profile:action", m)
				}
				eventMap[event] = mapping
			}

			config := core.WebhookServerConfig{
				Addr:        addr,
				EventMap:    eventMap,
				AllowEvents: allowList,
				Secret:      secret,
				DryRun:      dryRun,
			}

			if err := core.ServeWebhooks(config); err != nil {
				return fmt.Errorf("webhook server failed: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "Address to listen on")
	cmd.Flags().StringVar(&secret, "secret", "", "Shared secret for HMAC validation")
	cmd.Flags().StringSliceVar(&eventMaps, "event-map", nil, "Map event to action (event=profile:action)")
	cmd.Flags().StringSliceVar(&allowList, "allow-event", nil, "Allowed event types")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Log events without executing")

	return cmd
}
