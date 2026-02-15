package device

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"oss-aps-cli/internal/core/device"
	msgtypes "oss-aps-cli/internal/core/messenger"

	"github.com/spf13/cobra"
)

var messengerManager = msgtypes.NewManager()

func newChannelsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "channels <messenger>",
		Short: "List known channels for a messenger device",
		Long:  "Lists channels from existing mappings across all profiles for a messenger device.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChannels(args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type channelRow struct {
	ChannelID string `json:"channel_id"`
	MappedTo  string `json:"mapped_to,omitempty"`
	ProfileID string `json:"profile_id,omitempty"`
}

func runChannels(messengerName string, jsonOut bool) error {
	// Validate device exists and is a messenger
	dev, err := device.LoadDevice(messengerName)
	if err != nil {
		return err
	}
	if dev.Type != device.DeviceTypeMessenger {
		return fmt.Errorf("device '%s' is type '%s', not messenger", messengerName, dev.Type)
	}

	// Collect channels from all profile links
	links, err := messengerManager.GetMessengerLinks(messengerName)
	if err != nil {
		return err
	}

	rows := buildChannelRows(links)

	if jsonOut {
		return renderChannelsJSON(messengerName, rows)
	}

	return renderChannelsTable(messengerName, rows)
}

func buildChannelRows(links []msgtypes.ProfileMessengerLink) []channelRow {
	var rows []channelRow
	seen := make(map[string]bool)

	for _, link := range links {
		for channelID, action := range link.Mappings {
			if seen[channelID] {
				continue
			}
			seen[channelID] = true
			rows = append(rows, channelRow{
				ChannelID: channelID,
				MappedTo:  action,
				ProfileID: link.ProfileID,
			})
		}
	}

	return rows
}

func renderChannelsJSON(messengerName string, rows []channelRow) error {
	data := map[string]any{
		"messenger": messengerName,
		"channels":  rows,
		"count":     len(rows),
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func renderChannelsTable(messengerName string, rows []channelRow) error {
	platform := msgtypes.MessengerPlatform(messengerName)
	formatHint := msgtypes.ChannelIDFormat[platform]

	if len(rows) == 0 {
		fmt.Printf("No channels discovered for %s.\n", messengerName)
		fmt.Println()
		if formatHint != "" {
			fmt.Printf("  Channel ID format for %s:\n", messengerName)
			fmt.Printf("    %s\n", formatHint)
			fmt.Println()
		}
		fmt.Println("  Add a mapping:")
		fmt.Printf("    aps device link %s -p <profile> --mapping \"<channel_id>=<action>\"\n",
			messengerName)
		return nil
	}

	fmt.Printf("Channel discovery for %s\n\n", messengerName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  "+tableHeader.Render("CHANNEL ID")+"\t"+
		tableHeader.Render("MAPPED TO"))

	for _, r := range rows {
		mapped := dimStyle.Render("(unmapped)")
		if r.MappedTo != "" {
			mapped = r.MappedTo
		}
		fmt.Fprintf(w, "  %-24s\t%s\n", r.ChannelID, mapped)
	}
	w.Flush()

	if formatHint != "" {
		fmt.Println()
		fmt.Printf("  Channel ID format for %s:\n", messengerName)
		fmt.Printf("    %s\n", formatHint)
	}

	return nil
}
