package observability

import (
	"github.com/spf13/cobra"
)

// NewObservabilityCmd creates the observability command group.
func NewObservabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "observability",
		Aliases: []string{"otel", "o11y"},
		Short:   "Manage OpenTelemetry observability",
		Long: `Manage OpenTelemetry observability for agent profiles.

Enables distributed tracing and metrics export via OpenTelemetry.
Supports OTLP (gRPC) and stdout exporters.`,
	}

	cmd.AddCommand(NewToggleCmd())

	return cmd
}
