package workspace

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/core/multidevice"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

func newConflictsShowCmd() *cobra.Command {
	var (
		workspaceID string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "show <conflict-id>",
		Short: "Show conflict details",
		Long:  `Show full details of a conflict including both versions and their values.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConflictShow(args[0], workspaceID, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runConflictShow(conflictID, workspaceID string, jsonOut bool) error {
	mgr := multidevice.NewManager()
	conflict, err := mgr.GetConflict(workspaceID, conflictID)
	if err != nil {
		return fmt.Errorf("conflict '%s' not found in workspace '%s'",
			conflictID, workspaceID)
	}

	if jsonOut {
		data, err := json.MarshalIndent(conflict, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Conflict: %s", conflictID)))

	fmt.Printf("  Workspace:  %s\n", boldStyle.Render(workspaceID))
	fmt.Printf("  Type:       %s\n", conflict.Type)
	fmt.Printf("  Status:     %s\n",
		styles.ConflictStatusBadge(string(conflict.Status)))
	fmt.Printf("  Resource:   %s\n", conflict.Resource)
	fmt.Printf("  Detected:   %s\n",
		conflict.DetectedAt.Format("2006-01-02 15:04:05"))

	if conflict.ResolvedAt != nil {
		fmt.Printf("  Resolved:   %s\n",
			conflict.ResolvedAt.Format("2006-01-02 15:04:05"))
	}

	// Show involved events.
	fmt.Println()
	fmt.Printf("  %s\n\n", boldStyle.Render("Conflicting Events:"))

	for i, evt := range conflict.Events {
		label := fmt.Sprintf("  Version %d", i+1)
		fmt.Printf("  %s\n", boldStyle.Render(label))
		fmt.Printf("    Event ID:  %s\n", evt.ID)
		fmt.Printf("    Device:    %s\n", evt.DeviceID)
		fmt.Printf("    Type:      %s\n",
			styles.EventTypeBadge(string(evt.EventType)))
		fmt.Printf("    Time:      %s\n",
			evt.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Version:   %d\n", evt.Version)

		if evt.Payload != nil {
			payloadJSON, _ := json.MarshalIndent(evt.Payload, "    ", "  ")
			fmt.Printf("    Payload:\n    %s\n", string(payloadJSON))
		}
		fmt.Println()
	}

	// Show resolution if present.
	if conflict.Resolution != nil {
		fmt.Printf("  %s\n\n", boldStyle.Render("Resolution:"))
		fmt.Printf("    Strategy:  %s\n", conflict.Resolution.Strategy)
		if conflict.Resolution.WinnerEvent != "" {
			fmt.Printf("    Winner:    %s\n", conflict.Resolution.WinnerEvent)
		}
		if conflict.Resolution.ResolvedBy != "" {
			fmt.Printf("    Resolved by: %s\n", conflict.Resolution.ResolvedBy)
		}
	} else {
		fmt.Printf("  %s\n", dimStyle.Render("To resolve:"))
		fmt.Printf("    aps workspace conflicts resolve %s --workspace %s --strategy lww\n",
			conflictID, workspaceID)
	}

	return nil
}
