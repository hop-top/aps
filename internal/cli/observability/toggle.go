package observability

import (
	"fmt"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/core"
)

// NewToggleCmd creates the observability toggle command.
func NewToggleCmd() *cobra.Command {
	var (
		enabled      string
		exporter     string
		endpoint     string
		samplingRate float64
	)

	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Enable or disable observability for a profile",
		Long: `Enable or disable OpenTelemetry observability for a profile.

Without --enabled flag, toggles the current state.
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps observability toggle --profile worker                          # Toggle
  aps observability toggle --profile worker --enabled=on             # Force enable
  aps observability toggle --profile worker --enabled=off            # Force disable
  aps o11y toggle --profile worker --exporter=otlp --endpoint=localhost:4317
  aps otel toggle --profile worker --exporter=stdout --sampling-rate=0.5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --profile is a persistent global (declared in root.go
			// Globals). Read it from the inherited flag set instead of
			// redeclaring as a local (signature validator local-globals
			// check, T-0648).
			profileID, _ := cmd.Flags().GetString("profile")
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			action := ""
			if cmd.Flags().Changed("enabled") {
				if enabled == "" || enabled == "on" {
					action = "enable"
				} else if enabled == "off" {
					action = "disable"
				} else {
					return fmt.Errorf("invalid value for --enabled: %s (use: on, off, or omit for toggle)", enabled)
				}
			} else {
				if core.ProfileHasCapability(profile, "agntcy-observability") {
					action = "disable"
				} else {
					action = "enable"
				}
			}

			if action == "enable" {
				return enableObservability(profile, exporter, endpoint, samplingRate)
			}
			return disableObservability(profile)
		},
	}

	cmd.Flags().StringVar(&enabled, "enabled", "", "Enable (on), disable (off), or toggle (omit)")
	cmd.Flags().StringVar(&exporter, "exporter", "stdout", "Exporter type (otlp, stdout, none)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "OTLP collector endpoint (e.g. localhost:4317)")
	cmd.Flags().Float64Var(&samplingRate, "sampling-rate", 1.0, "Trace sampling rate (0.0–1.0)")
	// Toggling sets capability + observability config on the profile —
	// local write. Re-running with the same --enabled state is a no-op,
	// hence idempotent yes.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}

func enableObservability(profile *core.Profile, exporter, endpoint string, samplingRate float64) error {
	profileID := profile.ID

	validExporters := map[string]bool{"otlp": true, "stdout": true, "none": true}
	if !validExporters[exporter] {
		return fmt.Errorf("invalid exporter: %s (use: otlp, stdout, none)", exporter)
	}

	if samplingRate < 0 || samplingRate > 1 {
		return fmt.Errorf("sampling rate must be between 0.0 and 1.0, got: %f", samplingRate)
	}

	if err := core.AddCapabilityToProfile(profileID, "agntcy-observability"); err != nil {
		return fmt.Errorf("failed to add observability capability: %w", err)
	}

	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	profile.Observability = &core.ObservabilityConfig{
		Exporter:     exporter,
		Endpoint:     endpoint,
		SamplingRate: samplingRate,
	}

	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("Observability enabled for profile: %s\n", profileID)
	fmt.Printf("Exporter: %s\n", exporter)
	if endpoint != "" {
		fmt.Printf("Endpoint: %s\n", endpoint)
	}
	fmt.Printf("Sampling rate: %.2f\n", samplingRate)

	return nil
}

func disableObservability(profile *core.Profile) error {
	profileID := profile.ID

	if err := core.RemoveCapabilityFromProfile(profileID, "agntcy-observability"); err != nil {
		return fmt.Errorf("failed to remove observability capability: %w", err)
	}

	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	profile.Observability = nil

	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("Observability disabled for profile: %s\n", profileID)

	return nil
}
