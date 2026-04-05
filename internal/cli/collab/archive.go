package collab

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/prompt"
)

// NewArchiveCmd creates the "collab archive" command.
func NewArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive [workspace]",
		Short: "Archive a collaboration workspace",
		Long: `Archive a collaboration workspace. Archived workspaces are read-only
and cannot accept new agents or tasks. Use --force to skip confirmation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				confirmed, err := prompt.Confirm(
					fmt.Sprintf("Archive workspace '%s'? It becomes read-only.", wsID))
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

			if err := mgr.Archive(cmd.Context(), wsID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"status":    "archived",
				})
			}

			fmt.Printf("Archived workspace '%s'\n", wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
