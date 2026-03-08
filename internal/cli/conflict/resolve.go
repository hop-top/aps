package conflict

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"hop.top/aps/internal/core/multidevice"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

func newResolveCmd() *cobra.Command {
	var (
		workspaceID string
		strategy    string
		choice      string
		dryRun      bool
		force       bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "resolve <conflict-id>",
		Short: "Resolve a conflict",
		Long: `Resolve a workspace conflict using the specified strategy.

Strategies:
  lww     Last-write-wins: the most recent event wins
  manual  Choose a specific event as the winner

Use --dry-run to see what would happen without applying changes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConflictResolve(
				args[0], workspaceID, strategy, choice,
				dryRun, force, jsonOutput,
			)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&strategy, "strategy", "lww",
		"Resolution strategy: lww, manual")
	cmd.Flags().StringVar(&choice, "choice", "",
		"Event ID to choose as winner (for manual strategy)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would happen without applying")
	cmd.Flags().BoolVar(&force, "force", false,
		"Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runConflictResolve(
	conflictID, workspaceID, strategy, choice string,
	dryRun, force, jsonOut bool,
) error {
	mgr := multidevice.NewManager()
	conflict, err := mgr.GetConflict(workspaceID, conflictID)
	if err != nil {
		return fmt.Errorf("conflict '%s' not found in workspace '%s'",
			conflictID, workspaceID)
	}

	if conflict.Status == multidevice.ConflictResolved {
		fmt.Printf(dimStyle.Render(
			"Conflict '%s' is already resolved.")+"\n", conflictID)
		return nil
	}

	// Validate strategy.
	switch strategy {
	case "lww", "manual":
		// ok
	default:
		return fmt.Errorf("invalid strategy '%s': must be lww or manual",
			strategy)
	}

	if strategy == "manual" && choice == "" {
		// List events for the user to choose from.
		fmt.Println(boldStyle.Render("Choose a winning event:"))
		fmt.Println()
		for i, evt := range conflict.Events {
			fmt.Printf("  %d. %s (device: %s, time: %s)\n",
				i+1, evt.ID, evt.DeviceID,
				evt.Timestamp.Format("15:04:05"))
		}
		return fmt.Errorf("use --choice <event-id> to select a winner")
	}

	// Show what will happen.
	fmt.Printf("  Conflict:  %s\n", boldStyle.Render(conflictID))
	fmt.Printf("  Resource:  %s\n", conflict.Resource)
	fmt.Printf("  Strategy:  %s\n", strategy)
	fmt.Printf("  Events:    %d conflicting versions\n", len(conflict.Events))
	fmt.Println()

	if strategy == "lww" {
		// Preview LWW winner.
		var latest *multidevice.WorkspaceEvent
		for _, evt := range conflict.Events {
			if latest == nil || evt.Timestamp.After(latest.Timestamp) {
				latest = evt
			}
		}
		if latest != nil {
			fmt.Printf("  Winner:    %s (device: %s, latest write)\n",
				latest.ID, latest.DeviceID)
		}
	} else if choice != "" {
		fmt.Printf("  Winner:    %s (manual choice)\n", choice)
	}

	if dryRun {
		fmt.Println()
		fmt.Println(dimStyle.Render("  No changes made. Remove --dry-run to apply."))
		return nil
	}

	if !force {
		fmt.Printf("\n  Apply this resolution? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	if err := mgr.ResolveConflict(
		workspaceID, conflictID, strategy, choice,
	); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"conflict_id":  conflictID,
			"workspace_id": workspaceID,
			"strategy":     strategy,
			"status":       "resolved",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("\n%s Conflict '%s' resolved using %s strategy.\n",
		styles.Success.Render("OK"), conflictID, strategy)

	return nil
}
