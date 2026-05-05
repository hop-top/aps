package adapter

import (
	"encoding/json"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	coreadapter "hop.top/aps/internal/core/adapter"
	msgtypes "hop.top/aps/internal/core/messenger"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
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

// channelRow is the table/json/yaml row shape for `aps device
// channels`. T-0476 — added `table:"COL,priority=N"` + yaml tags
// so styled tables activate on a TTY via the listing wrapper. The
// JSON envelope (renderChannelsJSON) wraps these rows under
// `{messenger, channels, count}` and is unchanged.
type channelRow struct {
	ChannelID string `table:"CHANNEL ID,priority=10" json:"channel_id"           yaml:"channel_id"`
	MappedTo  string `table:"MAPPED TO,priority=9"   json:"mapped_to,omitempty"  yaml:"mapped_to,omitempty"`
	ProfileID string `json:"profile_id,omitempty"    yaml:"profile_id,omitempty"`
}

func runChannels(messengerName string, jsonOut bool) error {
	// Validate device exists and is a messenger
	dev, err := coreadapter.LoadAdapter(messengerName)
	if err != nil {
		return err
	}
	if dev.Type != coreadapter.AdapterTypeMessenger {
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

	tableRows := make([]channelRow, len(rows))
	for i, r := range rows {
		mapped := r.MappedTo
		if mapped == "" {
			mapped = dimStyle.Render("(unmapped)")
		}
		tableRows[i] = channelRow{
			ChannelID: r.ChannelID,
			MappedTo:  mapped,
		}
	}
	if err := listing.RenderList(os.Stdout, output.Table, tableRows); err != nil {
		return err
	}

	if formatHint != "" {
		fmt.Println()
		fmt.Printf("  Channel ID format for %s:\n", messengerName)
		fmt.Printf("    %s\n", formatHint)
	}

	return nil
}
