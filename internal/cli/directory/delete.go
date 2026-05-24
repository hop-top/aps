package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
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

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: profileID})

			client, err := discovery.NewClient(profile.Directory)
			if err != nil {
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()

			r.Emit(ctx, progress.Event{Phase: phaseDelete, Item: profileID})
			if err := client.Deregister(ctx, profileID); err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseDelete, Item: profileID, OK: &okFalse})
				return fmt.Errorf("failed to delete record: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: phaseDelete, Item: profileID, OK: &okTrue})

			fmt.Printf("Deleted profile %s from AGNTCY Directory\n", profileID)

			return nil
		},
	}

	clinote.AddFlag(cmd) // T-1291

	// T-0654 — Deregistration removes a shared upstream AGNTCY Directory
	// record; the loss propagates beyond the caller's local scope, so
	// classify as destructive-shared. Delete-by-name is idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	kitcli.SetDestructiveToken(cmd)
	// T-0656 — destructive-token confirm already gates the upstream
	// deregistration call; preview would need the same wire round-trip.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "delete is a deregistration call to the upstream AGNTCY Directory; previewing would require the same network round-trip that performs the deregistration."); err != nil {
		panic(err)
	}

	return cmd
}
