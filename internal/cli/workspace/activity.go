package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"oss-aps-cli/internal/core/multidevice"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

// NewActivityCmd creates the workspace activity command.
func NewActivityCmd() *cobra.Command {
	var (
		follow     bool
		typeFilter string
		deviceID   string
		since      string
		exclude    string
		limit      int
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "activity <workspace-id>",
		Short: "Show workspace activity log",
		Long: `Show recent activity events for a workspace.

Events are grouped by type: profile, action, device, workspace, conflict.
Use --follow to tail the log in real time.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinceTime, err := parseSinceDuration(since)
			if err != nil {
				return err
			}
			return runActivity(
				args[0], follow, typeFilter, deviceID,
				sinceTime, exclude, limit, jsonOutput,
			)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false,
		"Follow the activity log (live tail)")
	cmd.Flags().StringVarP(&typeFilter, "type", "t", "",
		"Filter by event type: profile, action, device, workspace, conflict")
	cmd.Flags().StringVar(&deviceID, "device", "",
		"Filter by device ID")
	cmd.Flags().StringVar(&since, "since", "24h",
		"Show events since duration (e.g. 1h, 30m, 7d)")
	cmd.Flags().StringVar(&exclude, "exclude", "",
		"Exclude event type category")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50,
		"Maximum number of events to display")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runActivity(
	workspaceID string, follow bool, typeFilter, deviceID string,
	since time.Time, exclude string, limit int, jsonOut bool,
) error {
	mgr := multidevice.NewManager()
	events, err := mgr.GetEvents(workspaceID, since, limit)
	if err != nil {
		return err
	}

	// Filter events.
	events = filterEvents(events, typeFilter, deviceID, exclude)

	if len(events) == 0 && !follow {
		fmt.Printf(styles.Dim.Render("No activity in workspace '%s'.")+"\n",
			workspaceID)
		fmt.Println()
		fmt.Println(styles.Dim.Render("  Activity appears when:"))
		fmt.Println(styles.Dim.Render("    - Profiles are created or updated"))
		fmt.Println(styles.Dim.Render("    - Actions are executed"))
		fmt.Println(styles.Dim.Render("    - Devices connect or disconnect"))
		return nil
	}

	if jsonOut && !follow {
		data, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if !follow {
		return renderActivityTable(workspaceID, events)
	}

	// Follow mode: print existing events then tail.
	return runActivityFollow(workspaceID, events, typeFilter, deviceID,
		exclude, jsonOut)
}

func renderActivityTable(
	workspaceID string, events []*multidevice.WorkspaceEvent,
) error {
	fmt.Printf("%s\n\n",
		styles.Title.Render(fmt.Sprintf("Activity: %s", workspaceID)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	hdr := styles.Bold
	fmt.Fprintln(w,
		hdr.Render("TIMESTAMP")+"\t"+
			hdr.Render("EVENT")+"\t"+
			hdr.Render("DEVICE")+"\t"+
			hdr.Render("DETAIL"))

	for _, evt := range events {
		ts := evt.Timestamp.Format("15:04:05")
		eventBadge := styles.EventTypeBadge(string(evt.EventType))
		dev := evt.DeviceID
		if dev == "" {
			dev = styles.Dim.Render("--")
		}
		detail := extractDetail(evt)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ts, eventBadge, dev, detail)
	}
	w.Flush()

	fmt.Printf("\n%s\n",
		styles.Dim.Render(fmt.Sprintf("%d events", len(events))))

	return nil
}

func runActivityFollow(
	workspaceID string, initial []*multidevice.WorkspaceEvent,
	typeFilter, deviceID, exclude string, jsonOut bool,
) error {
	// Print initial events.
	if !jsonOut && len(initial) > 0 {
		_ = renderActivityTable(workspaceID, initial)
		fmt.Println()
	} else if jsonOut {
		for _, evt := range initial {
			data, _ := json.Marshal(evt)
			fmt.Println(string(data))
		}
	}

	fmt.Fprintf(os.Stderr, "%s\n",
		styles.Dim.Render("(following -- Ctrl+C to stop)"))

	// Track last seen version.
	var lastVersion int64
	if len(initial) > 0 {
		lastVersion = initial[len(initial)-1].Version
	}

	// Catch interrupt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	mgr := multidevice.NewManager()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, styles.Dim.Render("\nStopped."))
			return nil
		case <-ticker.C:
			events, err := mgr.GetEvents(workspaceID, time.Time{}, 0)
			if err != nil {
				continue
			}

			var newEvents []*multidevice.WorkspaceEvent
			for _, evt := range events {
				if evt.Version > lastVersion {
					newEvents = append(newEvents, evt)
				}
			}

			newEvents = filterEvents(newEvents, typeFilter, deviceID, exclude)

			for _, evt := range newEvents {
				if jsonOut {
					data, _ := json.Marshal(evt)
					fmt.Println(string(data))
				} else {
					ts := evt.Timestamp.Format("15:04:05")
					badge := styles.EventTypeBadge(string(evt.EventType))
					dev := evt.DeviceID
					if dev == "" {
						dev = "--"
					}
					detail := extractDetail(evt)
					fmt.Printf("%s  %s  %s  %s\n", ts, badge, dev, detail)
				}
				if evt.Version > lastVersion {
					lastVersion = evt.Version
				}
			}
		}
	}
}

func filterEvents(
	events []*multidevice.WorkspaceEvent,
	typeFilter, deviceID, exclude string,
) []*multidevice.WorkspaceEvent {
	if typeFilter == "" && deviceID == "" && exclude == "" {
		return events
	}

	var filtered []*multidevice.WorkspaceEvent
	for _, evt := range events {
		if typeFilter != "" && evt.EventType.Category() != typeFilter {
			continue
		}
		if deviceID != "" && evt.DeviceID != deviceID {
			continue
		}
		if exclude != "" && evt.EventType.Category() == exclude {
			continue
		}
		filtered = append(filtered, evt)
	}

	return filtered
}

func extractDetail(evt *multidevice.WorkspaceEvent) string {
	if evt.Payload == nil {
		return styles.Dim.Render("--")
	}

	// Try common payload keys for a human-readable detail.
	for _, key := range []string{"name", "profile_id", "action_name", "resource"} {
		if v, ok := evt.Payload[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}

	return styles.Dim.Render("--")
}

func parseSinceDuration(s string) (time.Time, error) {
	if s == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}

	// Support "Nd" for days.
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour), nil
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid duration '%s': use format like 1h, 30m, 7d", s)
	}

	return time.Now().Add(-d), nil
}
