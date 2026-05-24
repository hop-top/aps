package identity

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	idpkg "hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
)

// NewShowCmd creates the identity show command.
func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show identity for a profile",
		Long: `Display the DID and identity configuration for a profile.

Profile is supplied via the tool-level --profile global:
  aps --profile worker identity show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0648 — read --profile from the tool-level global.
			profileID := globals.Profile()
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			did, err := idpkg.LoadDID(profile)
			if err != nil {
				return fmt.Errorf("no identity configured: %w", err)
			}

			fmt.Printf("Profile:  %s\n", profileID)
			fmt.Printf("DID:      %s\n", did)

			if profile.Identity.KeyPath != "" {
				fmt.Printf("Key path: %s\n", profile.Identity.KeyPath)
			}

			if len(profile.Identity.Badges) > 0 {
				fmt.Printf("Badges:   %v\n", profile.Identity.Badges)
			}

			// Resolve DID document
			doc, err := idpkg.ResolveDID(did)
			if err == nil && doc != nil && len(doc.VerificationMethod) > 0 {
				data, _ := json.MarshalIndent(doc, "", "  ")
				fmt.Printf("\nDID Document:\n%s\n", string(data))
			}

			return nil
		},
	}

	// T-0648 — kit/cli signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}
