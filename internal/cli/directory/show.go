package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
)

// NewShowCmd creates the directory show command.
func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a profile's OASF record",
		Long: `Display the OASF record for a profile as it would appear in the Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate Directory record fetch on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory show: %w", globals.ErrOffline)
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

			ctx := cmd.Context()
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: profileID})

			client, err := discovery.NewClient(profile.Directory)
			if err != nil {
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()

			r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: profileID})
			record, err := client.Show(ctx, profileID)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: profileID, OK: &okFalse})
				return fmt.Errorf("failed to get record: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: profileID, OK: &okTrue})

			formatted, err := discovery.FormatRecord(record)
			if err != nil {
				return fmt.Errorf("failed to format record: %w", err)
			}

			fmt.Println(formatted)

			return nil
		},
	}

	// T-0648 — kit/cli signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}
