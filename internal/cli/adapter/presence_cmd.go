package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/multidevice"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
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

// presenceRow is the json/yaml row shape for `aps device presence
// --json`. Numeric fields (sync_lag, offline_queue) stay ints so
// downstream tools can consume the structured output without parsing
// suffixed strings ("5 events", "3 pending"). T-0474 — added yaml
// tags for parity with json since structured callers may opt into
// either format.
type presenceRow struct {
	DeviceID     string `json:"device_id"     yaml:"device_id"`
	State        string `json:"state"         yaml:"state"`
	LastSeen     string `json:"last_seen"     yaml:"last_seen"`
	SyncLag      int    `json:"sync_lag"      yaml:"sync_lag"`
	OfflineQueue int    `json:"offline_queue" yaml:"offline_queue"`
}

// presenceTableRow is the human-readable row for the table path.
// State, SyncLag, and Queue are pre-rendered with badges + suffixes
// ("5 events", "3 pending"). T-0474 — separate from `presenceRow`
// so structured (--json/--yaml) consumers keep ints for sync_lag
// + offline_queue while the table path shows units.
type presenceTableRow struct {
	DeviceID string `table:"DEVICE,priority=10"   json:"device_id"  yaml:"device_id"`
	State    string `table:"STATUS,priority=9"    json:"state"      yaml:"state"`
	LastSeen string `table:"LAST SEEN,priority=8" json:"last_seen"  yaml:"last_seen"`
	SyncLag  string `table:"SYNC LAG,priority=7"  json:"sync_lag"   yaml:"sync_lag"`
	Queue    string `table:"QUEUE,priority=6"     json:"queue"      yaml:"queue"`
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

	tableRows := make([]presenceTableRow, len(rows))
	for i, r := range rows {
		lagStr := "--"
		if r.SyncLag > 0 {
			lagStr = fmt.Sprintf("%d events", r.SyncLag)
		}
		queueStr := "--"
		if r.OfflineQueue > 0 {
			queueStr = fmt.Sprintf("%d pending", r.OfflineQueue)
		}
		tableRows[i] = presenceTableRow{
			DeviceID: r.DeviceID,
			State:    styles.PresenceBadge(r.State),
			LastSeen: r.LastSeen,
			SyncLag:  lagStr,
			Queue:    queueStr,
		}
	}
	if err := listing.RenderList(os.Stdout, output.Table, tableRows); err != nil {
		return err
	}

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
