package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"hop.top/aps/internal/core/multidevice"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

// NewSyncCmd creates the workspace sync command.
func NewSyncCmd() *cobra.Command {
	var (
		deviceID   string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "sync <workspace-id>",
		Short: "Sync workspace state",
		Long: `Sync workspace state for the current device.

Normally sync happens automatically when devices connect.
Use this command when:
  - A device was offline and you want to force a sync
  - You suspect sync is behind and want to catch up
  - You want to verify sync status`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(args[0], deviceID, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&deviceID, "device", "",
		"Device ID to sync (defaults to current device)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runSync(workspaceID, deviceID string, jsonOut bool) error {
	if deviceID == "" {
		deviceID = detectCurrentDevice()
	}

	mgr := multidevice.NewManager()

	if !jsonOut {
		fmt.Printf("Syncing %s... ", styles.Bold.Render(workspaceID))
	}

	start := time.Now()
	result, err := mgr.SyncDevice(deviceID, workspaceID, 0)
	if err != nil {
		if !jsonOut {
			fmt.Println(styles.Error.Render("failed"))
		}
		return err
	}

	if jsonOut {
		data, err := json.MarshalIndent(map[string]interface{}{
			"workspace_id":       workspaceID,
			"device_id":          deviceID,
			"events_processed":   result.EventsProcessed,
			"duration_ms":        result.Duration.Milliseconds(),
			"conflicts_detected": result.ConflictsDetected,
			"auto_resolved":      result.AutoResolved,
			"manual_pending":     result.ManualPending,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	elapsed := time.Since(start)
	fmt.Printf("%s (%d events, %s)\n",
		styles.Success.Render("done"),
		result.EventsProcessed,
		formatSyncDuration(elapsed))

	if result.ConflictsDetected > 0 {
		fmt.Printf("\n%s %d conflicts detected",
			styles.Warn.Render("Warning:"), result.ConflictsDetected)
		if result.AutoResolved > 0 {
			fmt.Printf(" (%d auto-resolved)", result.AutoResolved)
		}
		if result.ManualPending > 0 {
			fmt.Printf(" (%d need manual resolution)", result.ManualPending)
		}
		fmt.Printf("\n  aps conflict list %s\n", workspaceID)
	}

	return nil
}

func detectCurrentDevice() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "local"
	}
	return hostname
}

func formatSyncDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
