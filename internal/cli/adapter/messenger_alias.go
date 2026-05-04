package adapter

import (
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	coreadapter "hop.top/aps/internal/core/adapter"
)

// NewMessengerCmd creates the "aps messenger" alias command.
// It delegates to device subcommands that apply to messenger devices,
// providing a convenient shorthand for messenger-focused workflows.
func NewMessengerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "messenger",
		Aliases: []string{"messengers"},
		Short:   "Messenger device commands (alias for 'aps device' with messenger context)",
		Long: `Messenger commands provide shortcuts for common messenger device operations.

These commands are equivalent to their 'aps device' counterparts but
pre-filtered for messenger-type devices.`,
	}

	// Core messenger operations
	cmd.AddCommand(newMessengerListCmd())
	// T-0398 — link parent (add/list/delete) replaces flat link/links/unlink.
	cmd.AddCommand(newLinkParentCmd())
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newTestMessengerCmd())

	// Lifecycle commands
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

// messengerSummaryRow is the row shape for `aps adapter messenger list`
// (T-0436). Profile inherits from the global --profile (T-0376); the
// command exposes --platform / --status only.
type messengerSummaryRow struct {
	Name         string `table:"NAME,priority=10"      json:"name"          yaml:"name"`
	Platform     string `table:"PLATFORM,priority=9"   json:"platform"      yaml:"platform"`
	Status       string `table:"STATUS,priority=8"     json:"status"        yaml:"status"`
	Profile      string `table:"PROFILE,priority=6"    json:"profile"       yaml:"profile"`
	ChannelCount int    `table:"CHANNELS,priority=5"   json:"channel_count" yaml:"channel_count"`
}

// newMessengerListCmd wraps the list command for messenger-type adapters.
// Output format is controlled by kit's persistent --format flag (T-0345);
// the per-command --json flag was removed in T-0363.
func newMessengerListCmd() *cobra.Command {
	var platformFilter, statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messenger adapters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMessengerList(platformFilter, statusFilter, globals.Profile())
		},
	}

	cmd.Flags().StringVar(&platformFilter, "platform", "", "Filter by messenger platform (telegram, slack, discord, ...)")
	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by runtime status (running, stopped, failed, ...)")

	return cmd
}

func runMessengerList(platformFilter, statusFilter, profileFilter string) error {
	adapters, err := coreadapter.ListAdapters(&coreadapter.AdapterFilter{
		Type: coreadapter.AdapterTypeMessenger,
	})
	if err != nil {
		return err
	}

	rows := buildMessengerRows(adapters)

	pred := listing.All(
		listing.MatchString(func(r messengerSummaryRow) string { return r.Platform }, platformFilter),
		listing.MatchString(func(r messengerSummaryRow) string { return r.Status }, statusFilter),
		listing.MatchString(func(r messengerSummaryRow) string { return r.Profile }, profileFilter),
	)
	rows = listing.Filter(rows, pred)

	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func buildMessengerRows(adapters []*coreadapter.Adapter) []messengerSummaryRow {
	rows := make([]messengerSummaryRow, 0, len(adapters))
	for _, a := range adapters {
		runtime, _ := defaultManager.GetRuntime(a.Name)
		status := string(coreadapter.StateStopped)
		if runtime != nil && runtime.State != "" {
			status = string(runtime.State)
		}

		profile := ""
		if a.Scope == coreadapter.ScopeProfile {
			profile = a.ProfileID
		}

		rows = append(rows, messengerSummaryRow{
			Name:         a.Name,
			Platform:     platformForMessenger(a),
			Status:       status,
			Profile:      profile,
			ChannelCount: messengerChannelCount(a.Name),
		})
	}
	return rows
}

// platformForMessenger derives the messenger platform from the adapter
// config. Manifest config["platform"] takes precedence; we fall back to
// the adapter name (channels.go uses the same convention — adapters are
// commonly named after their platform: "telegram", "slack", ...).
func platformForMessenger(a *coreadapter.Adapter) string {
	if a.Config != nil {
		if v, ok := a.Config["platform"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return a.Name
}

// messengerChannelCount sums the channel mappings across every linked
// profile for the named messenger. Errors are swallowed and reported
// as zero so a single unhealthy profile manifest can't break the list.
func messengerChannelCount(name string) int {
	links, err := messengerManager.GetMessengerLinks(name)
	if err != nil {
		return 0
	}
	total := 0
	for _, l := range links {
		total += len(l.Mappings)
	}
	return total
}

// T-0398 removed newMessengerLinkCmd / newMessengerUnlinkCmd. The
// link parent (newLinkParentCmd) covers add/list/delete.
