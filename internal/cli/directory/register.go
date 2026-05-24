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

// NewRegisterCmd creates the directory register command.
func NewRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a profile with the AGNTCY Directory",
		Long: `Register an agent profile with the AGNTCY Directory service.

Generates an OASF record from the profile and pushes it to the Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory register`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — short-circuit when --offline is set; AGNTCY Directory
			// registration is a network call. (Refactored from the inline
			// check landed in T-0386 to use the shared accessor.)
			if globals.IsOffline() {
				return fmt.Errorf("directory register: %w", globals.ErrOffline)
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

			if !core.ProfileHasCapability(profile, "agntcy-directory") {
				return fmt.Errorf("agntcy-directory capability not enabled for profile %s; enable it first", profileID)
			}

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: profileID})

			client, err := discovery.NewClient(profile.Directory)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: profileID, OK: &okFalse})
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()
			okConnect := true
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: profileID, OK: &okConnect})

			r.Emit(ctx, progress.Event{Phase: phasePublish, Item: profileID})
			record, err := client.Register(ctx, profile)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phasePublish, Item: profileID, OK: &okFalse})
				return fmt.Errorf("failed to register: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: phasePublish, Item: profileID, OK: &okTrue})

			formatted, err := discovery.FormatRecord(record)
			if err != nil {
				return fmt.Errorf("failed to format record: %w", err)
			}

			fmt.Printf("Registered profile %s with AGNTCY Directory\n", profileID)
			fmt.Printf("OASF Record:\n%s\n", formatted)

			return nil
		},
	}

	clinote.AddFlag(cmd) // T-1291

	// T-0648 — kit/cli signature annotations. Push to remote directory
	// (write-shared); register-if-not-exists is idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	// T-0656 — register pushes the local OASF record to the upstream
	// directory; preview would require the same wire round-trip.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "register pushes the locally-rendered OASF record to the upstream AGNTCY Directory; previewing would require the same wire round-trip that performs the publish."); err != nil {
		panic(err)
	}

	return cmd
}
