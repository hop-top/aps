package cli

import (
	"fmt"
	"os"
	"strings"

	"oss-aps-cli/internal/core"

	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhook server",
}

var webhookServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the webhook server",
	Run: func(cmd *cobra.Command, args []string) {
		addr, _ := cmd.Flags().GetString("addr")
		secret, _ := cmd.Flags().GetString("secret")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		eventMapFlags, _ := cmd.Flags().GetStringSlice("event-map")
		allowEvents, _ := cmd.Flags().GetStringSlice("allow-event")

		// Parse event map
		eventMap := make(map[string]string)
		for _, m := range eventMapFlags {
			parts := strings.SplitN(m, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Invalid event-map format '%s', expected event=profile:action\n", m)
				os.Exit(1)
			}
			eventMap[parts[0]] = parts[1]
		}

		config := core.WebhookServerConfig{
			Addr:        addr,
			EventMap:    eventMap,
			AllowEvents: allowEvents,
			Secret:      secret,
			DryRun:      dryRun,
		}

		if err := core.ServeWebhooks(config); err != nil {
			fmt.Fprintf(os.Stderr, "Webhook server failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.AddCommand(webhookServeCmd)

	webhookServeCmd.Flags().String("addr", "127.0.0.1:8080", "Address to listen on")
	webhookServeCmd.Flags().String("secret", "", "Shared secret for HMAC validation")
	webhookServeCmd.Flags().StringSlice("event-map", nil, "Map event to action (event=profile:action)")
	webhookServeCmd.Flags().StringSlice("allow-event", nil, "Allowed event types")
	webhookServeCmd.Flags().Bool("dry-run", false, "Log events without executing")
}
