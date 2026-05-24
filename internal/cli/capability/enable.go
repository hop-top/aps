package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/runtime/domain"
)

func newEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <profile> <capability>",
		Short: "Enable a capability on a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID, capName := args[0], args[1]

			if !capability.Exists(capName) {
				return fmt.Errorf("capability '%s' does not exist", capName)
			}

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("%w: profile '%s'", domain.ErrNotFound, profileID)
			}

			if core.ProfileHasCapability(profile, capName) {
				fmt.Println(dimStyle.Render(fmt.Sprintf(
					"'%s' already enabled on '%s'", capName, profileID)))
				return nil
			}

			// T-1291 — attach --note before the cap-add mutation.
			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			if err := core.AddCapabilityToProfileWithContext(ctx, profileID, capName); err != nil {
				return err
			}

			fmt.Println(styles.StatusDot(true) + " " +
				successStyle.Render("Enabled") + " " +
				boldStyle.Render(capName) + " on " + profileID)
			return nil
		},
	}
	clinote.AddFlag(cmd) // T-1291
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — enable appends one capability to a profile's manifest;
	// the result is fully determined by the two positional arguments.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "enable appends a single capability entry to the profile's manifest; the result is fully determined by the two positional arguments."); err != nil {
		panic(err)
	}
	return cmd
}

func newDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <profile> <capability>",
		Short: "Disable a capability on a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID, capName := args[0], args[1]

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("%w: profile '%s'", domain.ErrNotFound, profileID)
			}

			if !core.ProfileHasCapability(profile, capName) {
				fmt.Println(dimStyle.Render(fmt.Sprintf(
					"'%s' not enabled on '%s'", capName, profileID)))
				return nil
			}

			// T-1291 — attach --note before the cap-remove mutation.
			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			if err := core.RemoveCapabilityFromProfileWithContext(
				ctx, profileID, capName); err != nil {
				return err
			}

			fmt.Println(styles.StatusDot(false) + " " +
				successStyle.Render("Disabled") + " " +
				boldStyle.Render(capName) + " on " + profileID)
			return nil
		},
	}
	clinote.AddFlag(cmd) // T-1291
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — disable drops one capability from a profile's manifest;
	// the result is fully determined by the two positional arguments.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "disable drops a single capability entry from the profile's manifest; the result is fully determined by the two positional arguments."); err != nil {
		panic(err)
	}
	return cmd
}
