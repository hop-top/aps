package workspace

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	collab "hop.top/aps/internal/core/collaboration"
	kitcli "hop.top/kit/go/console/cli"
)

// NewRoleCmd creates the "collab role" command.
func NewRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role <agent> [role] [workspace]",
		Short: "Set an agent's role in the workspace",
		Long: `Change an agent's role within a collaboration workspace.

Valid roles: owner, contributor, observer

If role is not provided, an interactive selector is shown.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]

			var roleStr string
			var wsArgs []string

			if len(args) >= 2 {
				// Disambiguate: check if args[1] is a valid role
				candidate := collab.AgentRole(args[1])
				if candidate.Validate() == nil {
					roleStr = args[1]
					wsArgs = args[2:]
				} else {
					// Not a valid role — treat as workspace
					wsArgs = args[1:]
				}
			}

			if roleStr == "" {
				if err := huh.NewSelect[string]().
					Title(fmt.Sprintf("Role for %s", agent)).
					Options(
						huh.NewOption("owner", "owner"),
						huh.NewOption("contributor", "contributor"),
						huh.NewOption("observer", "observer"),
					).
					Value(&roleStr).
					Run(); err != nil {
					return err
				}
			}

			role := collab.AgentRole(roleStr)

			wsID, err := resolveWorkspace(cmd, wsArgs)
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			if err := mgr.SetRole(ctx, wsID, agent, role); err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"agent":     agent,
					"role":      string(role),
				})
			}

			fmt.Printf("Set role of '%s' to '%s' in workspace '%s'\n", agent, role, wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)
	clinote.AddFlag(cmd) // T-1291

	// T-0648 — updates the agent's role in local workspace state;
	// re-applying the same role yields the same observable state.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — role updates one field on the membership record; the
	// result is fully determined by the role argument.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "role updates a single field on the membership record; the resulting value is fully determined by the role argument the user already passed."); err != nil {
		panic(err)
	}

	return cmd
}
