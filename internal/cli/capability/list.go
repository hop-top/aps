package capability

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
)

// capabilitySummaryRow is the row shape for `aps capability list`.
type capabilitySummaryRow struct {
	Name        string   `table:"NAME,priority=9" json:"name" yaml:"name"`
	Source      string   `table:"SOURCE,priority=8" json:"source" yaml:"source"`
	Type        string   `table:"TYPE,priority=6" json:"type,omitempty" yaml:"type,omitempty"`
	Path        string   `table:"PATH,priority=4" json:"path,omitempty" yaml:"path,omitempty"`
	Tags        []string `table:"TAGS,priority=3" json:"tags,omitempty" yaml:"tags,omitempty"`
	Description string   `table:"DESCRIPTION,priority=2" json:"description,omitempty" yaml:"description,omitempty"`
	// Profiles holds the IDs that link this capability — preserved on
	// the row for --enabled-on filtering (not rendered in the table).
	Profiles []string `table:"-" json:"profiles,omitempty" yaml:"profiles,omitempty"`
}

func newListCmd() *cobra.Command {
	var (
		tag          string
		builtinOnly  bool
		externalOnly bool
		enabledOn    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all capabilities (builtin + external)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if builtinOnly && externalOnly {
				return fmt.Errorf("--builtin and --external are mutually exclusive")
			}
			format, _ := cmd.Flags().GetString("format")
			return runList(format, tag, builtinOnly, externalOnly, enabledOn)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "",
		"Filter to capabilities carrying this tag")
	cmd.Flags().BoolVar(&builtinOnly, "builtin", false,
		"Show only builtin capabilities")
	cmd.Flags().BoolVar(&externalOnly, "external", false,
		"Show only external (installed) capabilities")
	cmd.Flags().StringVar(&enabledOn, "enabled-on", "",
		"Show only capabilities linked to the given profile id")

	return cmd
}

func runList(format, tag string, builtinOnly, externalOnly bool, enabledOn string) error {
	rows := buildCapabilityRows()

	pred := listing.All(
		listing.MatchSlice(func(r capabilitySummaryRow) []string { return r.Tags }, tag),
		listing.MatchSlice(func(r capabilitySummaryRow) []string { return r.Profiles }, enabledOn),
		capabilitySourcePred(builtinOnly, externalOnly),
	)
	rows = listing.Filter(rows, pred)
	return listing.RenderList(os.Stdout, format, rows)
}

// buildCapabilityRows merges builtins + external installs into rows.
// Externals reuse the builtin slot when names collide (builtin wins).
func buildCapabilityRows() []capabilitySummaryRow {
	var rows []capabilitySummaryRow
	seen := make(map[string]bool)

	for _, b := range capability.ListBuiltins() {
		profs, _ := core.ProfilesUsingCapability(b.Name)
		rows = append(rows, capabilitySummaryRow{
			Name:        b.Name,
			Source:      "builtin",
			Description: b.Description,
			Tags:        b.Tags,
			Profiles:    profs,
		})
		seen[b.Name] = true
	}

	caps, _ := capability.List()
	for _, c := range caps {
		if seen[c.Name] {
			continue
		}
		profs, _ := core.ProfilesUsingCapability(c.Name)
		rows = append(rows, capabilitySummaryRow{
			Name:        c.Name,
			Source:      "external",
			Type:        string(c.Type),
			Path:        c.Path,
			Description: c.Description,
			Tags:        c.Tags,
			Profiles:    profs,
		})
		seen[c.Name] = true
	}

	return rows
}

// capabilitySourcePred gates on --builtin / --external. Caller validates
// mutual exclusion before constructing.
func capabilitySourcePred(builtinOnly, externalOnly bool) listing.Predicate[capabilitySummaryRow] {
	switch {
	case builtinOnly:
		return func(r capabilitySummaryRow) bool { return r.Source == "builtin" }
	case externalOnly:
		return func(r capabilitySummaryRow) bool { return r.Source == "external" }
	default:
		return nil
	}
}

// join is retained for sibling commands (show.go) that pre-date the
// listing.RenderList migration and still print plain joined slices.
func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
