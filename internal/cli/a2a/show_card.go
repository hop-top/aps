package a2a

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/cli/globals"
	kitcli "hop.top/kit/go/console/cli"
)

func NewShowCardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the Agent Card for a profile",
		Long:  `Display the A2A Agent Card for a specified profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0648 — read --profile / --format from the tool-level
			// globals; no local shadow.
			profileID := globals.Profile()
			if profileID == "" {
				return errors.New("--profile is required")
			}

			profile, err := loadProfile(profileID)
			if err != nil {
				return err
			}

			card, err := a2apkg.GenerateAgentCardFromProfile(profile)
			if err != nil {
				return fmt.Errorf("failed to generate agent card: %w", err)
			}

			switch globals.Format() {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(card); err != nil {
					return fmt.Errorf("encode card: %w", err)
				}
				return nil
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

	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}
