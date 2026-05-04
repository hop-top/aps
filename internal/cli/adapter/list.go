package adapter

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	coreadapter "hop.top/aps/internal/core/adapter"
)

// adapterSummaryRow is the kit/output row shape for `aps adapter list`
// (T-0435). Column priorities drop low-value columns first on narrow
// terminals; Name + Type stay highest.
type adapterSummaryRow struct {
	Name          string `table:"NAME,priority=10"           json:"name"           yaml:"name"`
	Type          string `table:"TYPE,priority=9"            json:"type"           yaml:"type"`
	Status        string `table:"STATUS,priority=8"          json:"status"         yaml:"status"`
	Workspace     string `table:"WORKSPACE,priority=7"       json:"workspace"      yaml:"workspace"`
	PairedDevices int    `table:"PAIRED,priority=5"          json:"paired_devices" yaml:"paired_devices"`
	LastSeenAt    string `table:"LAST SEEN,priority=4"       json:"last_seen_at"   yaml:"last_seen_at"`
}

func newListCmd() *cobra.Command {
	var typeFilter, statusFilter, workspaceFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List adapter devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(typeFilter, statusFilter, workspaceFilter)
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by adapter type (messenger, protocol, mobile, ...)")
	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by runtime status (running, stopped, failed, ...)")
	cmd.Flags().StringVar(&workspaceFilter, "workspace", "", "Filter by workspace (profile id for scope=profile, 'global' for scope=global)")

	return cmd
}

func runList(typeFilter, statusFilter, workspaceFilter string) error {
	adapters, err := coreadapter.ListAdapters(nil)
	if err != nil {
		return err
	}

	rows := buildAdapterRows(adapters)

	pred := listing.All(
		listing.MatchString(func(r adapterSummaryRow) string { return r.Type }, typeFilter),
		listing.MatchString(func(r adapterSummaryRow) string { return r.Status }, statusFilter),
		listing.MatchString(func(r adapterSummaryRow) string { return r.Workspace }, workspaceFilter),
	)
	rows = listing.Filter(rows, pred)

	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func buildAdapterRows(adapters []*coreadapter.Adapter) []adapterSummaryRow {
	rows := make([]adapterSummaryRow, 0, len(adapters))
	for _, a := range adapters {
		runtime, _ := defaultManager.GetRuntime(a.Name)

		status := string(coreadapter.StateStopped)
		var lastSeen string
		if runtime != nil {
			if runtime.State != "" {
				status = string(runtime.State)
			}
			if runtime.LastCheck != nil {
				lastSeen = runtime.LastCheck.Format(time.RFC3339)
			} else if runtime.StartedAt != nil {
				lastSeen = runtime.StartedAt.Format(time.RFC3339)
			}
		}

		rows = append(rows, adapterSummaryRow{
			Name:          a.Name,
			Type:          string(a.Type),
			Status:        status,
			Workspace:     workspaceForAdapter(a),
			PairedDevices: len(a.LinkedTo),
			LastSeenAt:    lastSeen,
		})
	}
	return rows
}

// workspaceForAdapter resolves the canonical workspace value for the
// adapter list. Profile-scoped adapters live under a profile id; global
// adapters are surfaced as "global" so the filter is set-membership
// safe (--workspace global, --workspace <profile-id>).
func workspaceForAdapter(a *coreadapter.Adapter) string {
	if a.Scope == coreadapter.ScopeProfile && a.ProfileID != "" {
		return a.ProfileID
	}
	return string(coreadapter.ScopeGlobal)
}

// join concatenates strings with ", " separator. Retained for callers
// in this package (set_permissions.go, stop.go).
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

// formatUptime is retained for callers in this package (status.go).
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
