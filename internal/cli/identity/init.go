package identity

import (
	"fmt"

	"github.com/spf13/cobra"

	idpkg "hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/core"
)

// NewInitCmd creates the identity init command.
func NewInitCmd() *cobra.Command {
	var (
		profileID string
		method    string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize identity for a profile",
		Long: `Generate a DID and Ed25519 key pair for a profile.

Supported DID methods:
  did:key  — Self-describing, no network required (default)
  did:web  — Web-based, requires hosting a DID document

Examples:
  aps identity init --profile worker
  aps id init --profile worker --method did:web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			if profile.Identity != nil && profile.Identity.DID != "" {
				return fmt.Errorf("identity already initialized for profile %s (DID: %s); remove it first to reinitialize", profileID, profile.Identity.DID)
			}

			did, keyPath, err := idpkg.GenerateDID(method, profileID)
			if err != nil {
				return fmt.Errorf("failed to generate DID: %w", err)
			}

			// Add capability
			if err := core.AddCapabilityToProfile(profileID, "agntcy-identity"); err != nil {
				return fmt.Errorf("failed to add identity capability: %w", err)
			}

			// Reload and save identity config
			profile, err = core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to reload profile: %w", err)
			}

			profile.Identity = &core.IdentityConfig{
				DID:     did,
				KeyPath: keyPath,
			}

			if err := core.SaveProfile(profile); err != nil {
				return fmt.Errorf("failed to save profile: %w", err)
			}

			fmt.Printf("Identity initialized for profile: %s\n", profileID)
			fmt.Printf("DID:      %s\n", did)
			fmt.Printf("Key path: %s\n", keyPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&method, "method", "did:key", "DID method (did:key, did:web)")

	return cmd
}
