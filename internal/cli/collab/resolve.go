package collab

import (
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	collab "hop.top/aps/internal/core/collaboration"
)

// NewResolveCmd creates the "collab resolve" command.
func NewResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <conflict-id>",
		Short: "Resolve a conflict",
		Long: `Resolve a detected conflict using the specified strategy.

Strategies:
  keep-first   Keep the earliest write
  keep-last    Keep the most recent write
  priority     Resolve by agent role priority
  rollback     Revert to pre-conflict value`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conflictID := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			strategy, _ := cmd.Flags().GetString("strategy")

			// Interactive strategy selection when not provided
			if !cmd.Flags().Changed("strategy") {
				if err := huh.NewSelect[string]().
					Title("Resolution strategy").
					Options(
						huh.NewOption("priority — resolve by agent role", "priority"),
						huh.NewOption("keep-first — keep earliest write", "keep-first"),
						huh.NewOption("keep-last — keep most recent write", "keep-last"),
						huh.NewOption("rollback — revert to pre-conflict", "rollback"),
					).
					Value(&strategy).
					Run(); err != nil {
					return err
				}
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			conflicts, err := store.LoadConflicts(wsID)
			if err != nil {
				return fmt.Errorf("loading conflicts: %w", err)
			}

			// Find the target conflict
			var targetIdx int = -1
			for i := range conflicts {
				if conflicts[i].ID == conflictID {
					targetIdx = i
					break
				}
			}

			if targetIdx < 0 {
				return fmt.Errorf("conflict %q not found", conflictID)
			}

			target := &conflicts[targetIdx]

			if target.IsResolved() {
				return fmt.Errorf("conflict %q is already resolved", conflictID)
			}

			// Create and apply the policy
			policy, err := collab.NewPolicy(
				collab.ResolutionStrategy(strategy),
			)
			if err != nil {
				return err
			}

			// Load workspace for policy resolution context
			mgr := collab.NewManager(store)
			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				return fmt.Errorf("loading workspace: %w", err)
			}

			// Ensure workspace context is available
			if ws.Context == nil {
				variables, _ := store.LoadContext(wsID)
				ws.Context = collab.NewWorkspaceContextFromState(
					variables, nil,
				)
			}

			resolution, err := policy.Resolve(
				cmd.Context(), *target, ws,
			)
			if err != nil {
				return err
			}

			// Mark resolved
			now := time.Now()
			target.Resolution = resolution
			target.ResolvedAt = &now

			if err := store.SaveConflicts(wsID, conflicts); err != nil {
				return fmt.Errorf("saving conflicts: %w", err)
			}

			if isJSON(cmd) {
				return outputJSON(target)
			}

			fmt.Printf("Resolved conflict %s\n", shortID(conflictID))
			fmt.Printf("  Strategy: %s\n", resolution.Strategy)
			fmt.Printf("  Details:  %s\n", resolution.Details)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("strategy", "priority",
		"Resolution strategy (keep-first, keep-last, priority, rollback)")
	addJSONFlag(cmd)

	return cmd
}
