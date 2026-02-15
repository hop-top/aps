package collab

import (
	"fmt"
	"os"

	collab "oss-aps-cli/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewAgentsCmd creates the "collab agents" command.
func NewAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Find agents by capability",
		Long: `Find agents in a workspace that match a capability or task description.

Use --cap for exact capability matching, or --task for fuzzy matching
against a task description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			cap, _ := cmd.Flags().GetString("cap")
			task, _ := cmd.Flags().GetString("task")

			if cap == "" && task == "" {
				return fmt.Errorf("specify --cap or --task to search for agents")
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			mgr := collab.NewManager(store)

			// Load workspace to rebuild capability index from agents
			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			registry := collab.NewCapabilityRegistry()

			// Register each agent's capabilities
			for _, agent := range ws.Agents {
				if len(agent.Capabilities) > 0 {
					_ = registry.Register(
						cmd.Context(), wsID,
						agent.ProfileID, agent.Capabilities,
					)
				}
			}

			query := collab.CapabilityQuery{
				Capability: cap,
				Task:       task,
			}

			matches, err := registry.FindAgents(cmd.Context(), wsID, query)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(matches)
			}

			if len(matches) == 0 {
				fmt.Println("No matching agents found.")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "AGENT\tSCORE\tMATCH\n")
			for _, m := range matches {
				fmt.Fprintf(w, "%s\t%.0f%%\t%s\n",
					m.Agent.ProfileID,
					m.Score*100,
					m.Match,
				)
			}
			w.Flush()

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("cap", "", "Capability name to match")
	cmd.Flags().String("task", "", "Task description for fuzzy matching")
	addJSONFlag(cmd)

	return cmd
}
