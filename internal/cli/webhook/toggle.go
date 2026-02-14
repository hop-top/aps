package webhook

import (
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
)

func NewToggleCmd() *cobra.Command {
	var (
		profileID string
		enabled   string
	)

	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Enable or disable Webhook for a profile",
		Long: `Enable or disable Webhook server for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps webhook toggle --profile worker                    # Toggle Webhook
  aps webhook toggle --profile worker --enabled=on      # Force enable
  aps webhook toggle --profile worker --enabled=off     # Force disable`,
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
				if core.ProfileHasCapability(profile, "webhooks") {
					action = "disable"
				} else {
					action = "enable"
				}
			}

			// Execute action
			if action == "enable" {
				if err := enableWebhook(profile); err != nil {
					return err
				}
			} else {
				if err := disableWebhook(profile); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Enable (on), disable (off), or toggle (omit or blank)")

	return cmd
}

func enableWebhook(profile *core.Profile) error {
	profileID := profile.ID

	// Add "webhooks" capability (deduplicates automatically)
	if err := core.AddCapabilityToProfile(profileID, "webhooks"); err != nil {
		return fmt.Errorf("failed to add Webhook capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Initialize webhook config with empty allowed events
	profile.Webhooks = core.WebhookConfig{
		AllowedEvents: []string{},
	}

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("Webhook enabled for profile: %s\n", profileID)

	return nil
}

func disableWebhook(profile *core.Profile) error {
	profileID := profile.ID

	// Remove "webhooks" capability
	if err := core.RemoveCapabilityFromProfile(profileID, "webhooks"); err != nil {
		return fmt.Errorf("failed to remove Webhook capability: %w", err)
	}

	// Reload profile to get the updated version
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to reload profile: %w", err)
	}

	// Clear webhook configuration (set to empty struct)
	profile.Webhooks = core.WebhookConfig{}

	// Save profile
	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("Webhook disabled for profile: %s\n", profileID)

	return nil
}
