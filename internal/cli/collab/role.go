package collab

import (
	"fmt"
	"os"

	collab "oss-aps-cli/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewRoleCmd creates the "collab role" command.
func NewRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role <agent> <role> [workspace]",
		Short: "Set an agent's role in the workspace",
		Long: `Change an agent's role within a collaboration workspace.

Valid roles: owner, contributor, observer`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			role := collab.AgentRole(args[1])

			if err := role.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			wsID, err := resolveWorkspace(cmd, args[2:])
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			if err := mgr.SetRole(cmd.Context(), wsID, agent, role); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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

	return cmd
}
