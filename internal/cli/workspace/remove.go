package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/prompt"
)

// NewRemoveCmd creates the "collab remove" command.
func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <agent> [workspace]",
		Short: "Remove an agent from the workspace",
		Long: `Remove an agent from a collaboration workspace. Only the workspace owner
can perform this action. Use --force to skip confirmation.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAgent := args[0]

			wsID, err := resolveWorkspace(cmd, args[1:])
			if err != nil {
				return err
			}

			actor, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				confirmed, err := prompt.Confirm(
					fmt.Sprintf("Remove agent '%s' from workspace '%s'?", targetAgent, wsID))
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

			// T-1291 — attach --note to ctx BEFORE the manager call.
			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			if err := mgr.Remove(ctx, wsID, targetAgent, actor); err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"agent":     targetAgent,
					"actor":     actor,
					"status":    "removed",
				})
			}

			fmt.Printf("Removed '%s' from workspace '%s'\n", targetAgent, wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)
	clinote.AddFlag(cmd) // T-1291

	return cmd
}
