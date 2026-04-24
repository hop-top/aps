package cli

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/events"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

var profileSetWorkspaceCmd = &cobra.Command{
	Use:   "set-workspace <profile-id> <workspace-name>",
	Short: "Set workspace link for a profile",
	Long:  `Associate a profile with a workspace by name.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID, workspaceName := args[0], args[1]

		profile, err := core.LoadProfile(profileID)
		if err != nil {
			return fmt.Errorf("failed to load profile %s: %w", profileID, err)
		}

		profile.Workspace = &core.WorkspaceLink{
			Name: workspaceName,
		}

		if err := core.SaveProfile(profile); err != nil {
			return fmt.Errorf("failed to save profile %s: %w", profileID, err)
		}

		publishEvent(string(events.TopicProfileUpdated), "", events.ProfileUpdatedPayload{
			ProfileID: profileID,
			Fields:    []string{"workspace"},
		})

		fmt.Fprintf(os.Stdout, "%s workspace set to %s for profile %s\n",
			styles.StatusDot(true),
			styles.Bold.Render(workspaceName),
			styles.Bold.Render(profileID))
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileSetWorkspaceCmd)
}
