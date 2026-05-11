package acp

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

func NewToggleCmd() *cobra.Command {
	var (
		profileID string
		enabled   string
		transport string
		host      string
		port      string
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
  aps acp toggle --profile worker --enabled=off     # Force disable`,
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
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport (stdio, ws, websocket)")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Listen host for network transports")
	cmd.Flags().StringVar(&port, "port", "8088", "Listen port for network transports")

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

	transport = normalizeACPTransport(transport)
	if !validACPTransport(transport) {
		return fmt.Errorf("invalid transport %q: use stdio, ws, or websocket", transport)
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

func normalizeACPTransport(transport string) string {
	transport = strings.ToLower(strings.TrimSpace(transport))
	if transport == "" {
		return "stdio"
	}
	if transport == "websocket" {
		return "ws"
	}
	return transport
}

func validACPTransport(transport string) bool {
	switch normalizeACPTransport(transport) {
	case "stdio", "ws":
		return true
	default:
		return false
	}
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
