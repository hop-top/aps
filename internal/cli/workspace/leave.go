package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/prompt"
)

// NewLeaveCmd creates the "collab leave" command.
func NewLeaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leave [workspace]",
		Short: "Leave a collaboration workspace",
		Long: `Leave a collaboration workspace. Use --force to skip confirmation.

If no workspace is specified, the active workspace is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				confirmed, err := prompt.Confirm(
					fmt.Sprintf("Leave workspace '%s'? This removes access to shared context.", wsID))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			if err := mgr.Leave(cmd.Context(), wsID, profile); err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"profile":   profile,
					"status":    "left",
				})
			}

			fmt.Printf("Left workspace '%s'\n", wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
