package webhook

import (
	"github.com/spf13/cobra"
)

// NewWebhookCmd creates the webhook command group
func NewWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage Webhook server",
		Long: `Manage Webhook server for event-driven task execution.

The webhook command group provides operations for:
- Enabling and configuring webhooks for a profile
- Starting a webhook server
- Managing webhook event mappings`,
	}

	cmd.AddCommand(NewToggleCmd())
	cmd.AddCommand(NewServerCmd())

	return cmd
}
