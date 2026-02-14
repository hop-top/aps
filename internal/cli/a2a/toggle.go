package a2a

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
)

func NewToggleCmd() *cobra.Command {
	var (
		profileID string
		enabled   string
		protocol  string
		host      string
		port      string
		url       string
	)

	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Enable or disable A2A for a profile",
		Long: `Enable or disable A2A (Agent-to-Agent) protocol for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps a2a toggle --profile worker                    # Toggle A2A
  aps a2a toggle --profile worker --enabled=on      # Force enable
  aps a2a toggle --profile worker --enabled=off     # Force disable
  aps a2a toggle --profile worker --protocol=grpc --port=9000`,
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
				if core.ProfileHasCapability(profile, "a2a") {
					action = "disable"
				} else {
					action = "enable"
				}
			}

			// Execute action
			if action == "enable" {
				if err := enableA2A(profile, protocol, host, port, url); err != nil {
					return err
				}
			} else {
				if err := disableA2A(profile); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Enable (on), disable (off), or toggle (omit or blank)")
	cmd.Flags().StringVar(&protocol, "protocol", "jsonrpc", "Protocol binding (jsonrpc, grpc, http)")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Listen host")
	cmd.Flags().StringVar(&port, "port", "8081", "Listen port")
	cmd.Flags().StringVar(&url, "url", "", "Public endpoint URL (defaults to http://{host}:{port})")

	return cmd
}

func enableA2A(profile *core.Profile, protocol, host, port, publicURL string) error {
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

	// Add "a2a" capability (deduplicates automatically)
	if err := core.AddCapabilityToProfile(profileID, "a2a"); err != nil {
		return fmt.Errorf("failed to add A2A capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Build listen address
	listenAddr := fmt.Sprintf("%s:%s", host, port)

	// Build public endpoint
	publicEndpoint := publicURL
	if publicEndpoint == "" {
		publicEndpoint = fmt.Sprintf("http://%s:%s", host, port)
	}

	// Validate protocol binding
	validBindings := map[string]bool{
		"jsonrpc": true,
		"grpc":    true,
		"http":    true,
	}
	if !validBindings[protocol] {
		return fmt.Errorf("invalid protocol binding: %s (use: jsonrpc, grpc, http)", protocol)
	}

	// Create A2A configuration
	profile.A2A = &core.A2AConfig{
		ProtocolBinding: protocol,
		ListenAddr:      listenAddr,
		PublicEndpoint:  publicEndpoint,
	}

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("A2A enabled for profile: %s\n", profileID)
	fmt.Printf("Protocol binding: %s\n", protocol)
	fmt.Printf("Listen address: %s\n", listenAddr)
	fmt.Printf("Public endpoint: %s\n", publicEndpoint)

	return nil
}

func disableA2A(profile *core.Profile) error {
	profileID := profile.ID

	// Remove "a2a" capability
	if err := core.RemoveCapabilityFromProfile(profileID, "a2a"); err != nil {
		return fmt.Errorf("failed to remove A2A capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Clear A2A configuration
	profile.A2A = nil

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("A2A disabled for profile: %s\n", profileID)

	return nil
}
