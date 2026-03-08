package adapter

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

func newPresenceCmd() *cobra.Command {
	var (
		workspaceID string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "presence [workspace-id]",
		Short: "Show device presence in a workspace",
		Long: `Show the current presence status of all devices linked to a workspace.

Displays device status, last heartbeat, and sync lag information.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID := workspaceID
			if len(args) > 0 {
				wsID = args[0]
			}
			if wsID == "" {
				return fmt.Errorf("workspace ID is required (as argument or --workspace flag)")
			}
			return runPresence(wsID, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type presenceRow struct {
	DeviceID    string `json:"device_id"`
	State       string `json:"state"`
	LastSeen    string `json:"last_seen"`
	SyncLag     int    `json:"sync_lag"`
	OfflineQueue int   `json:"offline_queue"`
}

func runPresence(workspaceID string, jsonOut bool) error {
	mgr := multidevice.NewManager()

	presences, err := mgr.GetDevicePresence(workspaceID)
	if err != nil {
		return err
	}

	if len(presences) == 0 {
		fmt.Printf(dimStyle.Render("No devices linked to workspace '%s'.")+"\n",
			workspaceID)
		fmt.Println()
		fmt.Println(dimStyle.Render("  Attach a device:"))
		fmt.Printf(dimStyle.Render("    aps device attach <device-id> --workspace %s")+"\n",
			workspaceID)
		return nil
	}

	rows := make([]presenceRow, len(presences))
	for i, p := range presences {
		lastSeen := "--"
		if !p.LastHeartbeat.IsZero() {
			lastSeen = formatPresenceTime(p.LastHeartbeat)
		}
		rows[i] = presenceRow{
			DeviceID:     p.DeviceID,
			State:        string(p.State),
			LastSeen:     lastSeen,
			SyncLag:      p.SyncLag,
			OfflineQueue: p.OfflineQueue,
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
		headerStyle.Render(fmt.Sprintf("Device Presence: %s", workspaceID)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("DEVICE")+"\t"+
		tableHeader.Render("STATUS")+"\t"+
		tableHeader.Render("LAST SEEN")+"\t"+
		tableHeader.Render("SYNC LAG")+"\t"+
		tableHeader.Render("QUEUE"))

	for _, r := range rows {
		badge := styles.PresenceBadge(r.State)
		lagStr := "--"
		if r.SyncLag > 0 {
			lagStr = fmt.Sprintf("%d events", r.SyncLag)
		}
		queueStr := "--"
		if r.OfflineQueue > 0 {
			queueStr = fmt.Sprintf("%d pending", r.OfflineQueue)
		}
		fmt.Fprintf(w, "%-18s\t%s\t%s\t%s\t%s\n",
			r.DeviceID, badge, r.LastSeen, lagStr, queueStr)
	}
	w.Flush()

	// Summary
	online, away, offline := 0, 0, 0
	for _, p := range presences {
		switch p.State {
		case multidevice.PresenceOnline:
			online++
		case multidevice.PresenceAway:
			away++
		default:
			offline++
		}
	}

	summary := fmt.Sprintf("%d devices (%d online, %d away, %d offline)",
		len(presences), online, away, offline)
	fmt.Printf("\n%s\n", dimStyle.Render(summary))

	return nil
}

func formatPresenceTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return t.Format("Jan 2 15:04")
}
