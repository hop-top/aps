package identity

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	idpkg "hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
)

// NewBadgeCmd creates the badge command group with issue and verify subcommands.
func NewBadgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "badge",
		Short: "Manage agent badges (Verifiable Credentials)",
		Long:  `Issue and verify agent badges (Verifiable Credentials) for capability attestation.`,
	}

	cmd.AddCommand(newBadgeIssueCmd())
	cmd.AddCommand(newBadgeVerifyCmd())

	// T-0648 — intermediate node for depth-3 leaves below.
	kitcli.SetHierarchical(cmd)

	return cmd
}

func newBadgeIssueCmd() *cobra.Command {
	var capability string

	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue a badge for a capability",
		Long: `Issue a signed Verifiable Credential attesting an agent's capability.

Profile is supplied via the tool-level --profile global:
  aps --profile worker identity badge issue --capability invoice-processing`,
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

			badge, err := idpkg.IssueBadge(profile, capability)
			if err != nil {
				return fmt.Errorf("failed to issue badge: %w", err)
			}

			// Save badge
			profileDir, err := core.GetProfileDir(profileID)
			if err != nil {
				return fmt.Errorf("failed to get profile directory: %w", err)
			}

			badgePath := filepath.Join(profileDir, "badges", capability+".json")
			if err := idpkg.SaveBadge(badge, badgePath); err != nil {
				return fmt.Errorf("failed to save badge: %w", err)
			}

			// Update profile's badge list
			profile.Identity.Badges = appendUnique(profile.Identity.Badges, badgePath)
			if err := core.SaveProfile(profile); err != nil {
				return fmt.Errorf("failed to update profile: %w", err)
			}

			fmt.Printf("Badge issued for profile: %s\n", profileID)
			fmt.Printf("Capability: %s\n", capability)
			fmt.Printf("Issuer:     %s\n", badge.Issuer)
			fmt.Printf("Saved to:   %s\n", badgePath)

			return nil
		},
	}

	cmd.Flags().StringVar(&capability, "capability", "", "Capability to attest (required)")
	cmd.MarkFlagRequired("capability")

	// T-0648 — kit/cli signature annotations. issue mints a new
	// credential per call → not naturally idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)

	return cmd
}

func newBadgeVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <badge-file>",
		Short: "Verify a badge file",
		Long: `Verify a badge (Verifiable Credential) file's signature and contents.

Example:
  aps identity badge verify ~/.agents/profiles/worker/badges/invoice-processing.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			badgePath := args[0]

			result, err := idpkg.VerifyBadge(badgePath)
			if err != nil {
				return fmt.Errorf("verification error: %w", err)
			}

			fmt.Printf("Badge:      %s\n", badgePath)
			fmt.Printf("Valid:      %t\n", result.Valid)
			fmt.Printf("Issuer:     %s\n", result.Issuer)
			fmt.Printf("Capability: %s\n", result.Capability)
			fmt.Printf("Subject:    %s\n", result.Subject)

			if result.Error != "" {
				fmt.Printf("Error:      %s\n", result.Error)
			}

			if !result.Valid {
				return fmt.Errorf("badge verification failed")
			}

			return nil
		},
	}

	// T-0648 — kit/cli signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
