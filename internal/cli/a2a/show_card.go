package a2a

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	a2apkg "oss-aps-cli/internal/a2a"
)

func NewShowCardCmd() *cobra.Command {
	var (
		profileID string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "show-card",
		Short: "Show the Agent Card for a profile",
		Long:  `Display the A2A Agent Card for a specified profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := loadProfile(profileID)
			if err != nil {
				return err
			}

			card, err := a2apkg.GenerateAgentCardFromProfile(profile)
			if err != nil {
				return fmt.Errorf("failed to generate agent card: %w", err)
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(card)
			default:
				fmt.Printf("Agent Card for Profile: %s\n", profile.ID)
				fmt.Printf("Display Name: %s\n", profile.DisplayName)
				fmt.Printf("URL: %s\n", card.URL)
				fmt.Printf("Transport: %s\n", card.PreferredTransport)
				if card.Description != "" {
					fmt.Printf("Description: %s\n", card.Description)
				}
				fmt.Printf("\nCapabilities:\n")
				fmt.Printf("  - Streaming: %v\n", card.Capabilities.Streaming)
				fmt.Printf("  - Push Notifications: %v\n", card.Capabilities.PushNotifications)
				fmt.Printf("  - State Transition History: %v\n", card.Capabilities.StateTransitionHistory)
				if len(card.Capabilities.Extensions) > 0 {
					fmt.Printf("  - Extensions: %d\n", len(card.Capabilities.Extensions))
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
