package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/adapter/mobile"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
)

// pendingTableRow is the table-only row shape for `aps device
// pending`. T-0475 — moved off hand-rolled tabwriter so styled
// tables activate on a TTY via the listing wrapper. DEVICE INFO
// composes "<name>, <os> [version]" at row-build time.
type pendingTableRow struct {
	DeviceID   string `table:"DEVICE,priority=10"      json:"device_id"   yaml:"device_id"`
	Requested  string `table:"REQUESTED,priority=8"    json:"requested"   yaml:"requested"`
	DeviceInfo string `table:"DEVICE INFO,priority=7"  json:"device_info" yaml:"device_info"`
}

// pendingJSONRow is the json/yaml row shape for `aps device pending
// --json`. Keeps the structured field set stable from before the
// T-0475 migration so downstream tools see no JSON-shape change.
type pendingJSONRow struct {
	DeviceID    string `json:"device_id"    yaml:"device_id"`
	ProfileID   string `json:"profile_id"   yaml:"profile_id"`
	DeviceName  string `json:"device_name"  yaml:"device_name"`
	DeviceOS    string `json:"device_os"    yaml:"device_os"`
	RequestedAt string `json:"requested_at" yaml:"requested_at"`
}

func newPendingCmd() *cobra.Command {
	var (
		profileID  string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "pending",
		Short: "List mobile devices pending approval",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPending(profileID, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Filter by profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runPending(profileID string, jsonOut bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	pending, err := registry.ListPending(profileID)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderPendingJSON(pending)
	}

	if len(pending) == 0 {
		fmt.Println(dimStyle.Render("  No devices pending approval."))
		return nil
	}

	fmt.Printf("%s\n\n", headerStyle.Render("Pending Approvals"))

	tableRows := buildPendingTableRows(pending)
	if err := listing.RenderList(os.Stdout, output.Table, tableRows); err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", dimStyle.Render(fmt.Sprintf("  %d devices pending approval.", len(pending))))
	fmt.Println("  Approve: aps device approve <device-id> --profile=<profile>")
	fmt.Println("  Reject:  aps device reject <device-id> --profile=<profile>")

	profileHint := ""
	if profileID != "" {
		profileHint = " --profile=" + profileID
	}
	fmt.Printf("  Approve all: aps device approve --all%s\n", profileHint)

	return nil
}

// buildPendingTableRows is the human table projection of the
// pending-device list. Pre-renders the REQUESTED dim badge and
// composes the DEVICE INFO column so the styled renderer just
// lays cells out.
func buildPendingTableRows(devices []*mobile.MobileAdapter) []pendingTableRow {
	rows := make([]pendingTableRow, 0, len(devices))
	for _, d := range devices {
		info := fmt.Sprintf("%s, %s", d.AdapterName, d.AdapterOS)
		if d.AdapterVersion != "" {
			info += " " + d.AdapterVersion
		}
		rows = append(rows, pendingTableRow{
			DeviceID:   d.AdapterID,
			Requested:  dimStyle.Render(formatTimeAgo(d.RegisteredAt)),
			DeviceInfo: info,
		})
	}
	return rows
}

func renderPendingJSON(devices []*mobile.MobileAdapter) error {
	out := make([]pendingJSONRow, 0, len(devices))
	for _, d := range devices {
		out = append(out, pendingJSONRow{
			DeviceID:    d.AdapterID,
			ProfileID:   d.ProfileID,
			DeviceName:  d.AdapterName,
			DeviceOS:    d.AdapterOS,
			RequestedAt: d.RegisteredAt.Format(time.RFC3339),
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// MobileStatusDot returns a status dot for mobile device pairing state
func MobileStatusDot(status mobile.PairingState) string {
	switch status {
	case mobile.PairingStateActive:
		return styles.StatusDot(true)
	case mobile.PairingStatePending:
		return styles.DeviceStateDot("starting")
	case mobile.PairingStateRevoked:
		return styles.DeviceStateDot("failed")
	case mobile.PairingStateExpired:
		return styles.DeviceStateDot("stopped")
	default:
		return styles.DeviceStateDot("unknown")
	}
}
