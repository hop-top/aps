package acp

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

func NewToggleCmd() *cobra.Command {
	var (
		profileID  string
		enabled    string
		transport  string
		host       string
		port       string
	)

	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Enable or disable ACP for a profile",
		Long: `Enable or disable ACP (Agent Client Protocol) for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps acp toggle --profile worker                    # Toggle ACP
  aps acp toggle --profile worker --enabled=on      # Force enable
  aps acp toggle --profile worker --enabled=off     # Force disable
  aps acp toggle --profile worker --transport=http --port=8088`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load profile
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			// Determine action: enable, disable, or toggle
			action := ""

			if cmd.Flags().Changed("enabled") {
				// Flag was explicitly provided
				if enabled == "" || enabled == "on" {
					action = "enable"
				} else if enabled == "off" {
					action = "disable"
				} else {
					return fmt.Errorf("invalid value for --enabled: %s (use: on, off, or omit for toggle)", enabled)
				}
			} else {
				// Flag not provided - toggle based on current state
				if profile.ACP != nil && profile.ACP.Enabled {
					action = "disable"
				} else {
					action = "enable"
				}
			}

			// Execute action
			if action == "enable" {
				if err := enableACP(profile, transport, host, port); err != nil {
					return err
				}
			} else {
				if err := disableACP(profile); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Enable (on), disable (off), or toggle (omit or blank)")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport (stdio, http, ws)")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Listen host (for http/ws)")
	cmd.Flags().StringVar(&port, "port", "8088", "Listen port (for http/ws)")

	return cmd
}

func enableACP(profile *core.Profile, transport, host, port string) error {
	profileID := profile.ID

	// Validate port is numeric
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid port number: %s", port)
	}

	// Validate host can be parsed
	if host != "" && net.ParseIP(host) == nil {
		// Could be a hostname, not just IP
		// For now, allow it to pass through
	}

	// Validate transport
	validTransports := map[string]bool{
		"stdio": true,
		"http":  true,
		"ws":    true,
	}
	if !validTransports[transport] {
		return fmt.Errorf("invalid transport: %s (use: stdio, http, ws)", transport)
	}

	// Add "agent-protocol" capability (deduplicates automatically)
	if err := core.AddCapabilityToProfile(profileID, "agent-protocol"); err != nil {
		return fmt.Errorf("failed to add ACP capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Parse port as integer
	portInt, _ := strconv.Atoi(port)

	// Create or update ACP configuration
	profile.ACP = &core.ACPConfig{
		Enabled:    true,
		Transport:  transport,
		ListenAddr: host,
		Port:       portInt,
	}

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("ACP enabled for profile: %s\n", profileID)
	fmt.Printf("Transport: %s\n", transport)
	if transport != "stdio" {
		fmt.Printf("Listen address: %s:%s\n", host, port)
	}

	return nil
}

func disableACP(profile *core.Profile) error {
	profileID := profile.ID

	// Remove "agent-protocol" capability
	if err := core.RemoveCapabilityFromProfile(profileID, "agent-protocol"); err != nil {
		return fmt.Errorf("failed to remove ACP capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Clear ACP configuration
	profile.ACP = nil

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("ACP disabled for profile: %s\n", profileID)

	return nil
}
