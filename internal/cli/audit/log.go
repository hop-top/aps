package audit

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

func newLogCmd() *cobra.Command {
	var (
		follow     bool
		deviceID   string
		result     string
		action     string
		since      string
		limit      int
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "log <workspace-id>",
		Short: "Show audit log",
		Long: `Show the audit log for a workspace.

Each entry records a device's access check: what action was attempted,
whether it was allowed or denied, and why.

Use --follow to tail the log in real time.
Use --json for machine-readable JSON lines output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinceTime, err := parseAuditSince(since)
			if err != nil {
				return err
			}
			return runAuditLog(
				args[0], follow, deviceID, result, action,
				sinceTime, limit, jsonOutput,
			)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false,
		"Follow the audit log (live tail)")
	cmd.Flags().StringVar(&deviceID, "device", "",
		"Filter by device ID")
	cmd.Flags().StringVar(&result, "result", "",
		"Filter by result: allow, deny")
	cmd.Flags().StringVar(&action, "action", "",
		"Filter by action (e.g. read, write, execute)")
	cmd.Flags().StringVar(&since, "since", "24h",
		"Show entries since duration (e.g. 1h, 30m, 7d)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50,
		"Maximum number of entries to display")
	cmd.Flags().BoolVar(&jsonOutput, "json", false,
		"JSON lines output")

	return cmd
}

func runAuditLog(
	workspaceID string, follow bool, deviceID, result, action string,
	since time.Time, limit int, jsonOut bool,
) error {
	logger := multidevice.NewAuditLogger(workspaceID)

	entries, err := logger.ListEntries(since, deviceID, result, action, limit)
	if err != nil {
		return err
	}

	if len(entries) == 0 && !follow {
		fmt.Printf(dimStyle.Render("No audit entries for workspace '%s'.")+"\n",
			workspaceID)
		fmt.Println()
		fmt.Println(dimStyle.Render("  Audit entries appear when devices check access."))
		fmt.Println(dimStyle.Render("  Try running: aps device attach <device> --workspace " +
			workspaceID))
		return nil
	}

	if jsonOut && !follow {
		for _, entry := range entries {
			data, _ := json.Marshal(entry)
			fmt.Println(string(data))
		}
		return nil
	}

	if !follow {
		return renderAuditTable(workspaceID, entries)
	}

	return runAuditFollow(workspaceID, entries, deviceID, result, action,
		since, jsonOut)
}

func renderAuditTable(
	workspaceID string, entries []*multidevice.AuditEntry,
) error {
	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Audit Log: %s", workspaceID)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w,
		tableHeader.Render("TIMESTAMP")+"\t"+
			tableHeader.Render("DEVICE")+"\t"+
			tableHeader.Render("ACTION")+"\t"+
			tableHeader.Render("RESOURCE")+"\t"+
			tableHeader.Render("RESULT"))

	for _, entry := range entries {
		ts := entry.Timestamp.Format("15:04:05")
		dev := entry.DeviceID
		if dev == "" {
			dev = dimStyle.Render("--")
		}
		resource := entry.Resource
		if resource == "" {
			resource = dimStyle.Render("--")
		}
		resultBadge := styles.ResultBadge(entry.Result)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			ts, dev, entry.Action, resource, resultBadge)
	}
	w.Flush()

	// Summary.
	allowed, denied := 0, 0
	for _, e := range entries {
		if e.Result == "allow" {
			allowed++
		} else {
			denied++
		}
	}

	summary := fmt.Sprintf("%d entries (%d allowed, %d denied)",
		len(entries), allowed, denied)
	fmt.Printf("\n%s\n", dimStyle.Render(summary))

	return nil
}

func runAuditFollow(
	workspaceID string, initial []*multidevice.AuditEntry,
	deviceID, result, action string, since time.Time, jsonOut bool,
) error {
	// Print initial entries.
	if !jsonOut && len(initial) > 0 {
		_ = renderAuditTable(workspaceID, initial)
		fmt.Println()
	} else if jsonOut {
		for _, entry := range initial {
			data, _ := json.Marshal(entry)
			fmt.Println(string(data))
		}
	}

	fmt.Fprintf(os.Stderr, "%s\n",
		dimStyle.Render("(following -- Ctrl+C to stop)"))

	// Track last seen timestamp.
	var lastSeen time.Time
	if len(initial) > 0 {
		lastSeen = initial[len(initial)-1].Timestamp
	} else {
		lastSeen = since
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	logger := multidevice.NewAuditLogger(workspaceID)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, dimStyle.Render("\nStopped."))
			return nil
		case <-ticker.C:
			entries, err := logger.ListEntries(
				lastSeen, deviceID, result, action, 0)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if !entry.Timestamp.After(lastSeen) {
					continue
				}

				if jsonOut {
					data, _ := json.Marshal(entry)
					fmt.Println(string(data))
				} else {
					ts := entry.Timestamp.Format("15:04:05")
					dev := entry.DeviceID
					if dev == "" {
						dev = "--"
					}
					resource := entry.Resource
					if resource == "" {
						resource = "--"
					}
					resultBadge := styles.ResultBadge(entry.Result)
					fmt.Printf("%s  %s  %s  %s  %s\n",
						ts, dev, entry.Action, resource, resultBadge)
				}

				if entry.Timestamp.After(lastSeen) {
					lastSeen = entry.Timestamp
				}
			}
		}
	}
}

func parseAuditSince(s string) (time.Time, error) {
	if s == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}

	// Support "Nd" for days.
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Now().Add(
				-time.Duration(days) * 24 * time.Hour), nil
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid duration '%s': use format like 1h, 30m, 7d", s)
	}

	return time.Now().Add(-d), nil
}
