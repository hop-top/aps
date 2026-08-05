package org

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	coreorg "hop.top/aps/internal/core/org"
	kitcli "hop.top/kit/go/console/cli"
)

// Relation values rendered by `aps org show`, in row order: the
// profile itself, its management chain, then reports.
const (
	relationSelf       = "self"
	relationManager    = "manager"
	relationDirect     = "direct-report"
	relationTransitive = "transitive-report"
)

// showRow is the table/json/yaml row shape for `aps org show`.
type showRow struct {
	Relation    string `table:"RELATION,priority=10" json:"relation"     yaml:"relation"`
	ID          string `table:"ID,priority=9"        json:"id"           yaml:"id"`
	DisplayName string `table:"NAME,priority=8"      json:"display_name" yaml:"display_name"`
	Type        string `table:"TYPE,priority=7"      json:"type"         yaml:"type"`
	Channels    string `table:"CHANNELS,priority=6"  json:"channels"     yaml:"channels"`
}

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <profile-id>",
		Short: "Show a profile's management chain and reports",
		Long: `Show where a profile sits in the reporting hierarchy: its
management chain up to the root (RELATION self, then manager rows in
walk order), its direct reports, and its transitive reports. Each
row carries the profile id, display name, effective type, and the
communication CHANNELS derivable from its configuration (a2a, acp,
email, webhooks).

--depth N limits how many levels of transitive reports are included;
the default 0 means unlimited. Direct reports always render.

Cyclic reports_to data never hangs: the partial chain is printed and
the command exits non-zero with a clear cycle error. Unknown profile
ids are an error. Output respects the global --format flag
(table|json|yaml).

Read-only: no state mutation. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			depth, _ := cmd.Flags().GetInt("depth")
			format, _ := cmd.Flags().GetString("format")
			return runShow(args[0], depth, format)
		},
	}

	cmd.Flags().Int("depth", 0, "Levels of transitive reports to include (0 = unlimited)")

	// Read-only hierarchy lookup; safely repeatable.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runShow(id string, depth int, format string) error {
	profiles, _, err := loadAllProfiles()
	if err != nil {
		return err
	}
	g := coreorg.Build(profiles)

	chain, chainErr := g.Chain(id)
	if len(chain) == 0 && chainErr == nil {
		return fmt.Errorf("unknown profile %q", id)
	}

	rows := make([]showRow, 0, len(chain))
	for i, p := range chain {
		relation := relationManager
		if i == 0 {
			relation = relationSelf
		}
		rows = append(rows, profileToShowRow(relation, p))
	}

	direct := g.DirectReports(id)
	directIDs := make(map[string]bool, len(direct))
	for _, p := range direct {
		directIDs[p.ID] = true
		rows = append(rows, profileToShowRow(relationDirect, p))
	}
	for _, p := range g.TransitiveReports(id, depth) {
		if directIDs[p.ID] {
			continue
		}
		rows = append(rows, profileToShowRow(relationTransitive, p))
	}

	if err := listing.RenderList(os.Stdout, format, rows); err != nil {
		return err
	}
	if chainErr != nil {
		return fmt.Errorf("management chain for %q: %w", id, chainErr)
	}
	return nil
}

// profileToShowRow projects a profile into the row shape rendered by
// `aps org show`. Channels are comma-joined for table readability.
func profileToShowRow(relation string, p core.Profile) showRow {
	return showRow{
		Relation:    relation,
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Type:        p.EffectiveType(),
		Channels:    strings.Join(coreorg.Channels(p), ", "),
	}
}
