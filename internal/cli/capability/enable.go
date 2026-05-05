package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
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
	return cmd
}
