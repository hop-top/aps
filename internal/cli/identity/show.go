package identity

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	idpkg "hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/core"
)

// NewShowCmd creates the identity show command.
func NewShowCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show identity for a profile",
		Long: `Display the DID and identity configuration for a profile.

Example:
  aps identity show --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
