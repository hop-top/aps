package identity

import (
	"github.com/spf13/cobra"
)

// NewIdentityCmd creates the identity command group.
func NewIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identity",
		Aliases: []string{"id"},
		Short:   "Manage DID-based agent identity",
		Long: `Manage decentralized identifiers (DIDs) and verifiable credentials for agent profiles.

Initialize identity to generate a DID and Ed25519 key pair. Issue and verify
badges (Verifiable Credentials) to attest agent capabilities.`,
	}

	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewVerifyCmd())
	cmd.AddCommand(NewBadgeCmd())

	return cmd
}
