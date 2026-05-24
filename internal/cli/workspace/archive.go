package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/prompt"
	kitcli "hop.top/kit/go/console/cli"
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

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			if err := mgr.Archive(ctx, wsID); err != nil {
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
	clinote.AddFlag(cmd) // T-1291

	// T-0648 — transitions the workspace to archived/read-only in local
	// state; archiving an archived workspace is a no-op.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — archive flips a single state bit on the workspace
	// record; preview would only restate the workspace ID.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "archive flips the workspace state bit to read-only; the operation is a one-field write whose result is fully determined by the workspace ID."); err != nil {
		panic(err)
	}

	return cmd
}
