package policy

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

func newTrustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Manage inbound trust verification policy",
		Long:  `Configure trust verification for inbound A2A requests using DID-based identity.`,
	}

	cmd.AddCommand(newTrustSetCmd())
	cmd.AddCommand(newTrustShowCmd())

	return cmd
}

func newTrustSetCmd() *cobra.Command {
	var (
		profileID       string
		requireIdentity bool
		allowedIssuers  string
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set trust verification policy for a profile",
		Long: `Configure trust verification for inbound A2A requests.

When require-identity is set, the A2A server will reject requests
without a valid X-Agent-DID header. Optionally restrict to specific
DIDs via --allowed-issuers.

Examples:
  aps policy trust set --profile worker --require-identity
  aps policy trust set --profile worker --require-identity --allowed-issuers "did:key:z6Mk..."
  aps policy trust set --profile worker --require-identity=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			// Add capability
			if err := core.AddCapabilityToProfile(profileID, "agntcy-trust"); err != nil {
				return fmt.Errorf("failed to add trust capability: %w", err)
			}

			profile, err = core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to reload profile: %w", err)
			}

			var issuers []string
			if allowedIssuers != "" {
				for _, s := range strings.Split(allowedIssuers, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						issuers = append(issuers, s)
					}
				}
			}

			profile.Trust = &core.TrustConfig{
				RequireIdentity: requireIdentity,
				AllowedIssuers:  issuers,
			}

			if err := core.SaveProfile(profile); err != nil {
				return fmt.Errorf("failed to save profile: %w", err)
			}

			fmt.Printf("Trust policy set for profile: %s\n", profileID)
			fmt.Printf("Require identity: %t\n", requireIdentity)
			if len(issuers) > 0 {
				fmt.Printf("Allowed issuers:  %s\n", strings.Join(issuers, ", "))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&requireIdentity, "require-identity", false, "Require X-Agent-DID header on inbound requests")
	cmd.Flags().StringVar(&allowedIssuers, "allowed-issuers", "", "Comma-separated list of allowed DIDs")

	return cmd
}

func newTrustShowCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show trust verification policy for a profile",
		Long: `Display the current trust verification policy configuration.

Example:
  aps policy trust show --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			if profile.Trust == nil {
				fmt.Printf("No trust policy configured for profile: %s\n", profileID)
				return nil
			}

			fmt.Printf("Trust policy for profile: %s\n", profileID)
			fmt.Printf("Require identity: %t\n", profile.Trust.RequireIdentity)
			if len(profile.Trust.AllowedIssuers) > 0 {
				fmt.Printf("Allowed issuers:\n")
				for _, issuer := range profile.Trust.AllowedIssuers {
					fmt.Printf("  - %s\n", issuer)
				}
			} else {
				fmt.Printf("Allowed issuers:  (any)\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
