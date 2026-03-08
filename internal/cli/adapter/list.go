package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	coreadapter "oss-aps-cli/internal/core/adapter"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var typeFilter string
	var profileFilter string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(typeFilter, profileFilter, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by device type (messenger, protocol, desktop, mobile)")
	cmd.Flags().StringVarP(&profileFilter, "profile", "p", "", "Filter by linked profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type deviceRow struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Scope    string   `json:"scope"`
	State    string   `json:"state"`
	Health   string   `json:"health"`
	Profiles []string `json:"profiles,omitempty"`
}

func runList(typeFilter, profileFilter string, jsonOut bool) error {
	filter := &coreadapter.AdapterFilter{}
	if typeFilter != "" {
		filter.Type = coreadapter.AdapterType(typeFilter)
	}
	if profileFilter != "" {
		filter.Profile = profileFilter
	}

	devices, err := coreadapter.ListAdapters(filter)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		return renderEmptyState(typeFilter)
	}

	rows := buildDeviceRows(devices)

	if jsonOut {
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	return renderDeviceTable(rows, typeFilter)
}

func buildDeviceRows(devices []*coreadapter.Adapter) []deviceRow {
	var rows []deviceRow

	for _, d := range devices {
		state := "stopped"
		health := "unknown"

		rows = append(rows, deviceRow{
			Name:     d.Name,
			Type:     string(d.Type),
			Scope:    string(d.Scope),
			State:    state,
			Health:   health,
			Profiles: d.LinkedTo,
		})
	}

	return rows
}

func renderEmptyState(typeFilter string) error {
	fmt.Println(dimStyle.Render("No devices configured."))
	fmt.Println()

	if typeFilter == "" {
		fmt.Println(dimStyle.Render("  Create a device:"))
		fmt.Println(dimStyle.Render("    aps device create my-telegram --type=messenger"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  Available types:"))
		for _, meta := range coreadapter.AdapterTypes {
			if meta.Implemented {
				fmt.Printf("    %-12s %s\n", meta.Type, dimStyle.Render(meta.Description))
			}
		}
	}

	return nil
}

func renderDeviceTable(rows []deviceRow, typeFilter string) error {
	if typeFilter != "" {
		fmt.Printf("%s\n\n", headerStyle.Render(fmt.Sprintf("Devices (type: %s)", typeFilter)))
	} else {
		fmt.Printf("%s\n\n", headerStyle.Render("Devices"))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("NAME")+"\t"+
		tableHeader.Render("TYPE")+"\t"+
		tableHeader.Render("SCOPE")+"\t"+
		tableHeader.Render("STATE")+"\t"+
		tableHeader.Render("PROFILE"))

	for _, r := range rows {
		typ := styles.DeviceTypeBadge(r.Type)
		scope := styles.ScopeBadge(r.Scope)
		state := styles.DeviceStateDot(r.State)
		profiles := "--"
		if len(r.Profiles) > 0 {
			profiles = joinMax(r.Profiles, 2)
		}
		fmt.Fprintf(w, "%-18s\t%s\t%s\t%s\t%s\n",
			r.Name, typ, scope, state, profiles)
	}
	w.Flush()

	running := 0
	stopped := 0
	for _, r := range rows {
		if r.State == "running" {
			running++
		} else {
			stopped++
		}
	}

	var summary string
	if typeFilter != "" {
		summary = fmt.Sprintf("%d %s devices (%d running, %d stopped)",
			len(rows), typeFilter, running, stopped)
	} else {
		summary = fmt.Sprintf("%d devices (%d running, %d stopped)",
			len(rows), running, stopped)
	}
	fmt.Printf("\n%s\n", dimStyle.Render(summary))

	return nil
}

func joinMax(ss []string, max int) string {
	if len(ss) <= max {
		return join(ss)
	}
	return join(ss[:max]) + fmt.Sprintf(" (+%d)", len(ss)-max)
}

func join(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
