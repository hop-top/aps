package cli

import (
	"oss-aps-cli/internal/cli/webhook"
)

func init() {
	rootCmd.AddCommand(webhook.NewWebhookCmd())
}
