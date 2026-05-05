package workspace

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/kit/go/console/output"

	"github.com/spf13/cobra"
)

// agentMatchRow is the table row shape for `aps workspace agents`.
// T-0456 — moved off hand-rolled tabwriter so styled tables activate
// on a TTY.
type agentMatchRow struct {
	Agent string `table:"AGENT,priority=10"  json:"agent" yaml:"agent"`
	Score string `table:"SCORE,priority=9"   json:"score" yaml:"score"`
	Match string `table:"MATCH,priority=8"   json:"match" yaml:"match"`
}

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
				return err
			}

			if isJSON(cmd) {
				return outputJSON(matches)
			}

			if len(matches) == 0 {
				fmt.Println("No matching agents found.")
				return nil
			}

			rows := make([]agentMatchRow, 0, len(matches))
			for _, m := range matches {
				rows = append(rows, agentMatchRow{
					Agent: m.Agent.ProfileID,
					Score: fmt.Sprintf("%.0f%%", m.Score*100),
					Match: m.Match,
				})
			}
			return listing.RenderList(os.Stdout, output.Table, rows)
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("cap", "", "Capability name to match")
	cmd.Flags().String("task", "", "Task description for fuzzy matching")
	addJSONFlag(cmd)

	return cmd
}
