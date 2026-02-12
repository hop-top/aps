package capability

import (
	"fmt"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/capability"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newEnableCmd() *cobra.Command {
	return &cobra.Command{
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
				return fmt.Errorf("profile '%s' not found", profileID)
			}

			if core.ProfileHasCapability(profile, capName) {
				fmt.Println(dimStyle.Render(fmt.Sprintf(
					"'%s' already enabled on '%s'", capName, profileID)))
				return nil
			}

			if err := core.AddCapabilityToProfile(profileID, capName); err != nil {
				return err
			}

			fmt.Println(styles.StatusDot(true) + " " +
				successStyle.Render("Enabled") + " " +
				boldStyle.Render(capName) + " on " + profileID)
			return nil
		},
	}
}

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <profile> <capability>",
		Short: "Disable a capability on a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID, capName := args[0], args[1]

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("profile '%s' not found", profileID)
			}

			if !core.ProfileHasCapability(profile, capName) {
				fmt.Println(dimStyle.Render(fmt.Sprintf(
					"'%s' not enabled on '%s'", capName, profileID)))
				return nil
			}

			if err := core.RemoveCapabilityFromProfile(
				profileID, capName); err != nil {
				return err
			}

			fmt.Println(styles.StatusDot(false) + " " +
				successStyle.Render("Disabled") + " " +
				boldStyle.Render(capName) + " on " + profileID)
			return nil
		},
	}
}
