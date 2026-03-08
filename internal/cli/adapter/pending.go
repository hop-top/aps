package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"hop.top/aps/internal/core/adapter/mobile"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("DEVICE")+"\t"+
		tableHeader.Render("REQUESTED")+"\t"+
		tableHeader.Render("DEVICE INFO"))

	for _, d := range pending {
		ago := formatTimeAgo(d.RegisteredAt)
		info := fmt.Sprintf("%s, %s", d.AdapterName, d.AdapterOS)
		if d.AdapterVersion != "" {
			info += " " + d.AdapterVersion
		}
		fmt.Fprintf(w, "%-24s\t%s\t%s\n",
			d.AdapterID, dimStyle.Render(ago), info)
	}
	w.Flush()

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

func renderPendingJSON(devices []*mobile.MobileAdapter) error {
	type pendingDevice struct {
		DeviceID    string `json:"device_id"`
		ProfileID   string `json:"profile_id"`
		DeviceName  string `json:"device_name"`
		DeviceOS    string `json:"device_os"`
		RequestedAt string `json:"requested_at"`
	}

	var out []pendingDevice
	for _, d := range devices {
		out = append(out, pendingDevice{
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
