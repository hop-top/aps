package adapter

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	coreadapter "hop.top/aps/internal/core/adapter"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// linkSummaryRow is the row shape for `aps adapter link list` (T-0437).
// Inherits messenger type-scoping per messenger_alias.go: the listing
// surfaces device-profile links across messenger adapters only.
type linkSummaryRow struct {
	Profile     string `table:"PROFILE,priority=10"     json:"profile"     yaml:"profile"`
	Device      string `table:"DEVICE,priority=9"       json:"device"      yaml:"device"`
	Permissions string `table:"PERMISSIONS,priority=7"  json:"permissions" yaml:"permissions"`
	LinkedAt    string `table:"LINKED AT,priority=5"    json:"linked_at"   yaml:"linked_at"`
}

// newLinkListCmd creates the `aps adapter link list` subcommand. T-0398
// renamed from `links` (plural-as-list noun) to `list` (CRUD verb
// under the `link` noun parent). `ls` kept as conventional shorthand.
//
// T-0437 — moved off tabwriter to listing.RenderList; filters consolidated
// to --profile and --messenger.
func newLinkListCmd() *cobra.Command {
	var profileFilter, messengerFilter string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List messenger-profile links",
		Long:    "Lists all messenger-profile links, optionally filtered by profile or messenger.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLinks(profileFilter, messengerFilter)
		},
	}

	cmd.Flags().StringVarP(&profileFilter, "profile", "p", "", "Filter by profile id")
	cmd.Flags().StringVar(&messengerFilter, "messenger", "", "Filter by messenger device name")

	return cmd
}

func runLinks(profileFilter, messengerFilter string) error {
	rows, err := collectLinkRows(profileFilter)
	if err != nil {
		return err
	}

	pred := listing.All(
		listing.MatchString(func(r linkSummaryRow) string { return r.Device }, messengerFilter),
	)
	rows = listing.Filter(rows, pred)

	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func collectLinkRows(profileFilter string) ([]linkSummaryRow, error) {
	var allLinks []msgtypes.ProfileMessengerLink

	if profileFilter != "" {
		links, err := messengerManager.GetProfileLinks(profileFilter)
		if err != nil {
			return nil, err
		}
		allLinks = links
	} else {
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

	rows := make([]linkSummaryRow, 0, len(allLinks))
	for _, l := range allLinks {
		rows = append(rows, linkSummaryRow{
			Profile:     l.ProfileID,
			Device:      l.MessengerName,
			Permissions: permissionsSummary(l),
			LinkedAt:    formatLinkedAt(l.CreatedAt),
		})
	}
	return rows, nil
}

// permissionsSummary renders a compact "what this link can do" string:
// state (enabled/disabled), routes count, and default-action flag. The
// table column stays narrow while still answering the question agents
// ask first ("does this link route messages and to where?").
func permissionsSummary(l msgtypes.ProfileMessengerLink) string {
	state := "enabled"
	if !l.Enabled {
		state = "disabled"
	}
	def := "-"
	if l.DefaultAction != "" {
		def = l.DefaultAction
	}
	return fmt.Sprintf("%s,routes=%d,default=%s", state, len(l.Mappings), def)
}

func formatLinkedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// getAllLinkedMessengerNames returns the names of all messenger devices
// that have at least one profile link. Used by the messenger alias command.
func getAllLinkedMessengerNames() ([]string, error) {
	devices, err := coreadapter.ListAdapters(&coreadapter.AdapterFilter{
		Type: coreadapter.AdapterTypeMessenger,
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
