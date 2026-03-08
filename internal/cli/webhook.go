package cli

import (
	"hop.top/aps/internal/cli/webhook"
)

func init() {
	rootCmd.AddCommand(webhook.NewWebhookCmd())
}
