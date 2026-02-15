package device

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/device"
	msgtypes "oss-aps-cli/internal/core/messenger"

	"github.com/spf13/cobra"
)

func newLinksCmd() *cobra.Command {
	var profileID string
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "links [messenger]",
		Short: "List messenger-profile links",
		Long:  "Lists all messenger-profile links, optionally filtered by profile or messenger.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var messengerFilter string
			if len(args) > 0 {
				messengerFilter = args[0]
			}
			return runLinks(profileID, messengerFilter, jsonOutput, verbose)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Filter by profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")

	return cmd
}

type linkRow struct {
	Messenger     string `json:"messenger"`
	ProfileID     string `json:"profile_id"`
	ChannelID     string `json:"channel_id"`
	Action        string `json:"action"`
	Status        string `json:"status"`
	DefaultAction string `json:"default_action,omitempty"`
}

func runLinks(profileID, messengerFilter string, jsonOut, verbose bool) error {
	rows, err := collectLinkRows(profileID, messengerFilter)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderLinksJSON(rows)
	}

	if len(rows) == 0 {
		return renderLinksEmpty()
	}

	return renderLinksTable(rows, verbose)
}

func collectLinkRows(profileID, messengerFilter string) ([]linkRow, error) {
	var allLinks []msgtypes.ProfileMessengerLink

	if profileID != "" {
		// Filter by specific profile
		links, err := messengerManager.GetProfileLinks(profileID)
		if err != nil {
			return nil, err
		}
		allLinks = links
	} else {
		// Scan all profiles
		profileIDs, err := core.ListProfiles()
		if err != nil {
			return nil, err
		}

		for _, pid := range profileIDs {
			links, err := messengerManager.GetProfileLinks(pid)
			if err != nil {
				continue
			}
			allLinks = append(allLinks, links...)
		}
	}

	var rows []linkRow
	for _, link := range allLinks {
		if messengerFilter != "" && link.MessengerName != messengerFilter {
			continue
		}

		status := "active"
		if !link.Enabled {
			status = "disabled"
		}

		// Add rows for each channel mapping
		for channelID, action := range link.Mappings {
			rows = append(rows, linkRow{
				Messenger: link.MessengerName,
				ProfileID: link.ProfileID,
				ChannelID: channelID,
				Action:    action,
				Status:    status,
			})
		}

		// Add default action row if present
		if link.DefaultAction != "" {
			rows = append(rows, linkRow{
				Messenger:     link.MessengerName,
				ProfileID:     link.ProfileID,
				ChannelID:     "*  (default)",
				Action:        link.DefaultAction,
				Status:        status,
				DefaultAction: link.DefaultAction,
			})
		}
	}

	return rows, nil
}

func renderLinksJSON(rows []linkRow) error {
	data := map[string]any{
		"links": rows,
		"count": len(rows),
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func renderLinksEmpty() error {
	fmt.Println("No messenger links found.")
	fmt.Println()
	fmt.Println("  Link a messenger:")
	fmt.Println("    aps device link telegram -p research-agent \\")
	fmt.Println("      --mapping \"<channel_id>=<action>\"")
	fmt.Println()
	fmt.Println("  Available messengers:")
	fmt.Println("    aps device list --type=messenger")
	return nil
}

func renderLinksTable(rows []linkRow, verbose bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  "+tableHeader.Render("MESSENGER")+"\t"+
		tableHeader.Render("PROFILE")+"\t"+
		tableHeader.Render("CHANNEL")+"\t"+
		tableHeader.Render("ACTION")+"\t"+
		tableHeader.Render("STATUS"))

	for _, r := range rows {
		statusStr := successStyle.Render(r.Status)
		if r.Status == "disabled" {
			statusStr = dimStyle.Render(r.Status)
		}

		fmt.Fprintf(w, "  %-12s\t%-18s\t%-20s\t%-18s\t%s\n",
			r.Messenger, r.ProfileID, r.ChannelID, r.Action, statusStr)
	}
	w.Flush()

	// Count unique messengers
	messengers := make(map[string]bool)
	for _, r := range rows {
		messengers[r.Messenger] = true
	}

	fmt.Printf("\n  %d mappings across %d messengers\n", len(rows), len(messengers))

	return nil
}

// getAllLinkedMessengerNames returns the names of all messenger devices
// that have at least one profile link. Used by the messenger alias command.
func getAllLinkedMessengerNames() ([]string, error) {
	devices, err := device.ListDevices(&device.DeviceFilter{
		Type: device.DeviceTypeMessenger,
	})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names, nil
}
