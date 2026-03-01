package identity

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	idpkg "hop.top/aps/internal/agntcy/identity"
)

// NewVerifyCmd creates the identity verify command.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <did>",
		Short: "Verify and resolve a DID",
		Long: `Resolve a DID to its DID Document and display verification details.

Example:
  aps identity verify did:key:z6Mk...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			did := args[0]

			doc, err := idpkg.ResolveDID(did)
			if err != nil {
				return fmt.Errorf("failed to resolve DID: %w", err)
			}

			fmt.Printf("DID: %s\n", did)
			fmt.Printf("Resolved: true\n")

			data, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format DID document: %w", err)
			}

			fmt.Printf("\nDID Document:\n%s\n", string(data))

			return nil
		},
	}

	return cmd
}
