package conflict

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"oss-aps-cli/internal/core/multidevice"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		includeResolved bool
		jsonOutput      bool
	)

	cmd := &cobra.Command{
		Use:     "list <workspace-id>",
		Aliases: []string{"ls"},
		Short:   "List workspace conflicts",
		Long: `List conflicts detected in a workspace.

By default, only unresolved conflicts are shown.
Use --include-resolved to include auto-resolved and manually resolved conflicts.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConflictList(args[0], includeResolved, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&includeResolved, "include-resolved", false,
		"Include resolved conflicts")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type conflictRow struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Resource string `json:"resource"`
	Devices  int    `json:"devices"`
	Age      string `json:"age"`
}

func runConflictList(workspaceID string, includeResolved, jsonOut bool) error {
	mgr := multidevice.NewManager()
	conflicts, err := mgr.ListConflicts(workspaceID, includeResolved)
	if err != nil {
		return err
	}

	if len(conflicts) == 0 {
		fmt.Printf(dimStyle.Render("No conflicts in workspace '%s'.")+"\n",
			workspaceID)
		fmt.Println()
		fmt.Println(dimStyle.Render("  Conflicts occur when multiple devices modify"))
		fmt.Println(dimStyle.Render("  the same resource concurrently during sync."))
		if !includeResolved {
			fmt.Println()
			fmt.Println(dimStyle.Render("  Use --include-resolved to see past conflicts."))
		}
		return nil
	}

	rows := make([]conflictRow, len(conflicts))
	for i, c := range conflicts {
		// Count unique devices involved.
		deviceSet := make(map[string]bool)
		for _, evt := range c.Events {
			deviceSet[evt.DeviceID] = true
		}

		rows[i] = conflictRow{
			ID:       c.ID,
			Type:     string(c.Type),
			Status:   string(c.Status),
			Resource: c.Resource,
			Devices:  len(deviceSet),
			Age:      formatAge(c.DetectedAt),
		}
	}

	if jsonOut {
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Conflicts: %s", workspaceID)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w,
		tableHeader.Render("ID")+"\t"+
			tableHeader.Render("TYPE")+"\t"+
			tableHeader.Render("STATUS")+"\t"+
			tableHeader.Render("RESOURCE")+"\t"+
			tableHeader.Render("DEVICES")+"\t"+
			tableHeader.Render("AGE"))

	for _, r := range rows {
		statusBadge := styles.ConflictStatusBadge(r.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			r.ID, r.Type, statusBadge, r.Resource, r.Devices, r.Age)
	}
	w.Flush()

	// Summary counts.
	pending, manual, autoResolved := 0, 0, 0
	for _, c := range conflicts {
		switch c.Status {
		case multidevice.ConflictPending:
			pending++
		case multidevice.ConflictManual:
			manual++
		case multidevice.ConflictAutoResolved:
			autoResolved++
		}
	}

	summary := fmt.Sprintf("%d conflicts", len(conflicts))
	if autoResolved > 0 {
		summary += fmt.Sprintf(" (%d auto-resolved pending review, %d manual)",
			autoResolved, manual)
	} else if manual > 0 {
		summary += fmt.Sprintf(" (%d manual)", manual)
	}
	fmt.Printf("\n%s\n", dimStyle.Render(summary))

	return nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%dd", days)
}
