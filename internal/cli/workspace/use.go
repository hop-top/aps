package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
)

// NewUseCmd creates the "collab use" command.
func NewUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <workspace>",
		Short: "Set active workspace",
		Long: `Set the active collaboration workspace.

Once set, other collab commands will use this workspace by default
when no --workspace flag is provided.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID := args[0]

			mgr, err := getManager()
			if err != nil {
				return err
			}

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			if err := mgr.SetActiveWorkspace(ctx, wsID); err != nil {
				return err
			}

			// Load workspace to show agent count
			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				// Workspace was set but we can't load details, still report success
				fmt.Printf("Active workspace: %s\n", wsID)
				return nil
			}

			online := ws.OnlineAgentCount()
			fmt.Printf("Active workspace: %s (%d agents online)\n", wsID, online)

			return nil
		},
	}

	clinote.AddFlag(cmd) // T-1291

	return cmd
}
