package webhook

import (
	"fmt"
	"os"

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

	var profileID string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a webhook server",
		Long: `Start a webhook server to receive and process webhook events.

The server listens on the specified address and processes incoming webhook
events by mapping them to profile actions.

If --profile is provided and webhooks are not enabled, will auto-enable them.

Examples:
  aps webhook server
  aps webhook server --addr 0.0.0.0:9000
  aps webhook server --profile worker --secret my-secret
  aps webhook server --secret my-secret --event-map github=profile:action`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Auto-enable webhooks if profile provided and not already configured
			if profileID != "" {
				profile, err := core.LoadProfile(profileID)
				if err != nil {
					return fmt.Errorf("failed to load profile %s: %w", profileID, err)
				}

				if !core.ProfileHasCapability(profile, "webhooks") {
					fmt.Fprintf(os.Stderr, "Webhooks not enabled for profile %s, auto-enabling...\n", profileID)
					if err := enableWebhook(profile); err != nil {
						return fmt.Errorf("failed to auto-enable webhooks: %w", err)
					}
					fmt.Fprintf(os.Stderr, "Webhooks enabled\n\n")
				}
			}

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
	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (optional, auto-enables webhooks if not configured)")

	return cmd
}
