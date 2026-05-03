package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
)

// NewDeleteCmd creates the directory delete command. Pairs with the
// existing 'register' verb; 'delete' is the canonical removal verb
// across the aps surface (cli-conventions §3.2).
func NewDeleteCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Remove a profile from the AGNTCY Directory",
		Long: `Remove an agent profile's record from the AGNTCY Directory.

Example:
  aps directory delete --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate Directory deregistration on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory delete: %w", globals.ErrOffline)
			}

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			client, err := discovery.NewClient(profile.Directory)
			if err != nil {
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()

			if err := client.Deregister(cmd.Context(), profileID); err != nil {
				return fmt.Errorf("failed to delete record: %w", err)
			}

			fmt.Printf("Deleted profile %s from AGNTCY Directory\n", profileID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
