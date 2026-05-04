package workspace

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
)

// policySummaryRow is the per-rule row rendered by
// `aps workspace policy [workspace]`. Each Workspace.Policy is
// expanded to N+1 rows: one for the default strategy plus one row
// per override key. Resource carries the resource glob the rule
// targets ("*" for the default fall-through).
type policySummaryRow struct {
	Key       string `table:"KEY,priority=10" json:"key" yaml:"key"`
	Strategy  string `table:"STRATEGY,priority=9" json:"strategy" yaml:"strategy"`
	Scope     string `table:"SCOPE,priority=6" json:"scope" yaml:"scope"`
	Resource  string `table:"RESOURCE,priority=7" json:"resource" yaml:"resource"`
	UpdatedAt string `table:"UPDATED,priority=4" json:"updated_at" yaml:"updated_at"`
}

// NewPolicyCmd creates the "collab policy" command.
func NewPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy [workspace]",
		Short: "Manage conflict resolution policy",
		Long: `View or set the conflict resolution policy for a workspace.

Use --set to change the default policy. Use --key with --set to
configure an override for a specific resource key.

Valid strategies:
  priority     Resolve by agent role priority (default)
  keep-first   Keep the earliest write
  keep-last    Keep the most recent write
  rollback     Revert to pre-conflict value
  consensus    Require agent agreement
  voting       Majority vote
  merge        Attempt to merge changes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			setPolicy, _ := cmd.Flags().GetString("set")
			key, _ := cmd.Flags().GetString("key")

			mgr, err := getManager()
			if err != nil {
				return err
			}

			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				return err
			}

			if setPolicy != "" {
				return setPolicyAction(cmd, ws, setPolicy, key)
			}

			return showPolicy(cmd, ws)
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("set", "", "Set default resolution strategy")
	cmd.Flags().String("key", "", "Resource key for override (use with --set)")
	addJSONFlag(cmd)

	return cmd
}

func setPolicyAction(
	cmd *cobra.Command,
	ws *collab.Workspace,
	strategy, key string,
) error {
	store, err := getStorage()
	if err != nil {
		return err
	}

	if key != "" {
		// Set override for a specific resource
		if ws.Policy.Overrides == nil {
			ws.Policy.Overrides = make(
				map[string]collab.ResolutionStrategy,
			)
		}
		ws.Policy.Overrides[key] = collab.ResolutionStrategy(strategy)
	} else {
		ws.Policy.Default = collab.ResolutionStrategy(strategy)
	}

	if err := store.SaveWorkspace(ws); err != nil {
		return fmt.Errorf("saving workspace: %w", err)
	}

	if isJSON(cmd) {
		return outputJSON(ws.Policy)
	}

	if key != "" {
		fmt.Printf("Set policy for '%s': %s\n", key, strategy)
	} else {
		fmt.Printf("Set default policy: %s\n", strategy)
	}

	return nil
}

func showPolicy(cmd *cobra.Command, ws *collab.Workspace) error {
	if isJSON(cmd) {
		return outputJSON(ws.Policy)
	}

	rows := buildPolicyRows(ws)
	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func buildPolicyRows(ws *collab.Workspace) []policySummaryRow {
	updated := ws.UpdatedAt.Format("2006-01-02 15:04:05")
	rows := make([]policySummaryRow, 0, 1+len(ws.Policy.Overrides))
	rows = append(rows, policySummaryRow{
		Key:       "default",
		Strategy:  string(ws.Policy.Default),
		Scope:     "workspace",
		Resource:  "*",
		UpdatedAt: updated,
	})
	for resource, strategy := range ws.Policy.Overrides {
		rows = append(rows, policySummaryRow{
			Key:       resource,
			Strategy:  string(strategy),
			Scope:     "override",
			Resource:  resource,
			UpdatedAt: updated,
		})
	}
	return rows
}
