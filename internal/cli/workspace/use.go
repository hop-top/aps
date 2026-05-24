package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	kitcli "hop.top/kit/go/console/cli"
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

	// T-0648 — sets the active workspace pointer in local state; setting
	// the same workspace twice yields the same final state.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — use overwrites a single pointer field; preview would
	// only restate the workspace argument.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "use overwrites the active-workspace pointer with the workspace argument; preview would only restate that argument."); err != nil {
		panic(err)
	}

	return cmd
}
