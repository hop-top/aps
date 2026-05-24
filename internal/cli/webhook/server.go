package webhook

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/core"
)

// NewServerCmd creates the `aps webhook server` command
func NewServerCmd() *cobra.Command {
	var (
		addr      string
		secret    string
		eventMaps []string
		allowList []string
	)

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
			// --profile and --dry-run are persistent globals (declared
			// in root.go Globals; kit/cli auto-registers --dry-run).
			// Read from the inherited flag set instead of redeclaring
			// locally (signature validator local-globals check, T-0648).
			profileID, _ := cmd.Flags().GetString("profile")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
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
				event, mapping, ok := strings.Cut(m, "=")
				if !ok || event == "" || mapping == "" {
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
	// Long-running daemon — kit's 6-tier ladder reserves interactive
	// for session-bound / serve commands. Each invocation starts a
	// fresh listener, so not idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectInteractive)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)

	return cmd
}
