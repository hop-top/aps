package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
)

// NewShowCmd creates the directory show command.
func NewShowCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a profile's OASF record",
		Long: `Display the OASF record for a profile as it would appear in the Directory.

Example:
  aps directory show --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate Directory record fetch on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory show: %w", globals.ErrOffline)
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

			record, err := client.Show(cmd.Context(), profileID)
			if err != nil {
				return fmt.Errorf("failed to get record: %w", err)
			}

			formatted, err := discovery.FormatRecord(record)
			if err != nil {
				return fmt.Errorf("failed to format record: %w", err)
			}

			fmt.Println(formatted)

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
