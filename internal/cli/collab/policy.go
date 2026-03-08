package collab

import (
	"fmt"
	"os"

	collab "hop.top/aps/internal/core/collaboration"

	"github.com/spf13/cobra"
)

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
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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

	fmt.Printf("Default: %s\n", ws.Policy.Default)

	if len(ws.Policy.Overrides) > 0 {
		fmt.Println()
		fmt.Println("Overrides:")

		w := newTabWriter()
		fmt.Fprintf(w, "  RESOURCE\tSTRATEGY\n")
		for k, v := range ws.Policy.Overrides {
			fmt.Fprintf(w, "  %s\t%s\n", k, v)
		}
		w.Flush()
	}

	return nil
}
