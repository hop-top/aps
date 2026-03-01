package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/core"
)

// NewDeregisterCmd creates the directory deregister command.
func NewDeregisterCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "deregister",
		Short: "Remove a profile from the AGNTCY Directory",
		Long: `Remove an agent profile's record from the AGNTCY Directory.

Example:
  aps directory deregister --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
				return fmt.Errorf("failed to deregister: %w", err)
			}

			fmt.Printf("Deregistered profile %s from AGNTCY Directory\n", profileID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
