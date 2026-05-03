package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
)

// NewRegisterCmd creates the directory register command.
func NewRegisterCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a profile with the AGNTCY Directory",
		Long: `Register an agent profile with the AGNTCY Directory service.

Generates an OASF record from the profile and pushes it to the Directory.

Example:
  aps directory register --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — short-circuit when --offline is set; AGNTCY Directory
			// registration is a network call. (Refactored from the inline
			// check landed in T-0386 to use the shared accessor.)
			if globals.IsOffline() {
				return fmt.Errorf("directory register: %w", globals.ErrOffline)
			}

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			if !core.ProfileHasCapability(profile, "agntcy-directory") {
				return fmt.Errorf("agntcy-directory capability not enabled for profile %s; enable it first", profileID)
			}

			client, err := discovery.NewClient(profile.Directory)
			if err != nil {
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()

			record, err := client.Register(cmd.Context(), profile)
			if err != nil {
				return fmt.Errorf("failed to register: %w", err)
			}

			formatted, err := discovery.FormatRecord(record)
			if err != nil {
				return fmt.Errorf("failed to format record: %w", err)
			}

			fmt.Printf("Registered profile %s with AGNTCY Directory\n", profileID)
			fmt.Printf("OASF Record:\n%s\n", formatted)

			return nil
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
