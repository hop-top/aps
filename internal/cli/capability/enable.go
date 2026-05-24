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
		Long: `Add <capability> to the capabilities list of the named profile at
$APS_DATA_PATH/profiles/<profile>/. The capability must already
exist as a builtin or be installed under $APS_DATA_PATH/capabilities/;
the profile must exist as well. Already-enabled pairings are a no-op
that prints a dim notice and leaves the manifest untouched.

Local write to the profile manifest only; pair with aps capability
disable to undo. Idempotent on the pair. --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.`,
		Args: cobra.ExactArgs(2),
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
		Long: `Drop <capability> from the capabilities list of the named
profile at $APS_DATA_PATH/profiles/<profile>/. Removes only the
profile-level linkage — the underlying capability record under
$APS_DATA_PATH/capabilities/<name>/ is left intact and remains
available to other profiles. Not-enabled pairings are a no-op
that prints a dim notice.

Local write to the profile manifest only; pair with aps capability
enable to restore. Idempotent on the pair. --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.`,
		Args: cobra.ExactArgs(2),
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
