package collab

import (
	"fmt"

	collab "hop.top/aps/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewCapsCmd creates the "collab caps" command.
func NewCapsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "caps",
		Aliases: []string{"capabilities"},
		Short:   "List capabilities in a workspace",
		Long: `List all registered capabilities in a collaboration workspace.

Optionally filter by a specific agent using --agent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			agentFilter, _ := cmd.Flags().GetString("agent")

			store, err := getStorage()
			if err != nil {
				return err
			}

			mgr := collab.NewManager(store)

			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				return err
			}

			registry := collab.NewCapabilityRegistry()

			for _, agent := range ws.Agents {
				if len(agent.Capabilities) > 0 {
					_ = registry.Register(
						cmd.Context(), wsID,
						agent.ProfileID, agent.Capabilities,
					)
				}
			}

			caps, err := registry.ListCapabilities(
				cmd.Context(), wsID, agentFilter,
			)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(caps)
			}

			if len(caps) == 0 {
				fmt.Println("No capabilities registered.")
				return nil
			}

			for _, c := range caps {
				fmt.Printf("  %s\n", c)
			}

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("agent", "", "Filter by agent profile ID")
	addJSONFlag(cmd)

	return cmd
}
