package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
)

// NewDeleteCmd creates the directory delete command. Pairs with the
// existing 'register' verb; 'delete' is the canonical removal verb
// across the aps surface (cli-conventions §3.2).
func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Remove a profile from the AGNTCY Directory",
		Long: `Remove an agent profile's record from the AGNTCY Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory delete`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate Directory deregistration on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory delete: %w", globals.ErrOffline)
			}

			// T-0648 — read --profile from the tool-level global.
			profileID := globals.Profile()
			if profileID == "" {
				return fmt.Errorf("--profile is required")
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

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			if err := client.Deregister(ctx, profileID); err != nil {
				return fmt.Errorf("failed to delete record: %w", err)
			}

			fmt.Printf("Deleted profile %s from AGNTCY Directory\n", profileID)

			return nil
		},
	}

	clinote.AddFlag(cmd) // T-1291

	// T-0648 — kit/cli signature annotations. Deregistration is
	// destructive (removes shared upstream record); delete-by-name is
	// idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}
