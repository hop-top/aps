package bundle

import (
	"fmt"
	"os"
	"sort"

	"hop.top/aps/internal/cli/listing"
	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
)

// bundleSummaryRow is the table/json/yaml row for `aps bundle list`.
type bundleSummaryRow struct {
	Name         string   `table:"NAME,priority=9" json:"name" yaml:"name"`
	Source       string   `table:"SOURCE,priority=8" json:"source" yaml:"source"`
	Capabilities int      `table:"CAPS,priority=6" json:"capabilities" yaml:"capabilities"`
	Tags         []string `table:"TAGS,priority=4" json:"tags,omitempty" yaml:"tags,omitempty"`
	Description  string   `table:"DESCRIPTION,priority=3" json:"description,omitempty" yaml:"description,omitempty"`
}

func newListCmd() *cobra.Command {
	var (
		tag         string
		builtinOnly bool
		userOnly    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List built-in and user bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if builtinOnly && userOnly {
				return fmt.Errorf("--builtin and --user are mutually exclusive")
			}
			format, _ := cmd.Flags().GetString("format")
			return runList(format, tag, builtinOnly, userOnly)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "",
		"Filter to bundles carrying this tag")
	cmd.Flags().BoolVar(&builtinOnly, "builtin", false,
		"Show only built-in bundles")
	cmd.Flags().BoolVar(&userOnly, "user", false,
		"Show only user bundles (incl. overrides)")

	return cmd
}

func runList(format, tag string, builtinOnly, userOnly bool) error {
	rows, err := loadBundleRows()
	if err != nil {
		return err
	}

	pred := listing.All(
		listing.MatchSlice(func(r bundleSummaryRow) []string { return r.Tags }, tag),
		sourcePredicate(builtinOnly, userOnly),
	)
	rows = listing.Filter(rows, pred)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	return listing.RenderList(os.Stdout, format, rows)
}

// loadBundleRows merges builtins and user overrides into summary rows.
// User-named bundles override builtins (rendered with "user (overrides …)").
func loadBundleRows() ([]bundleSummaryRow, error) {
	builtins, err := corebundle.LoadBuiltins()
	if err != nil {
		return nil, fmt.Errorf("failed to load built-in bundles: %w", err)
	}
	userBundles, err := corebundle.LoadUserOverrides()
	if err != nil {
		return nil, fmt.Errorf("failed to load user bundles: %w", err)
	}

	builtinNames := make(map[string]bool, len(builtins))
	for _, b := range builtins {
		builtinNames[b.Name] = true
	}
	userNames := make(map[string]bool, len(userBundles))
	for _, b := range userBundles {
		userNames[b.Name] = true
	}

	var rows []bundleSummaryRow
	for _, b := range builtins {
		if userNames[b.Name] {
			continue // user copy will be emitted with override marker
		}
		rows = append(rows, toRow(b, "built-in"))
	}
	for _, b := range userBundles {
		source := "user"
		if builtinNames[b.Name] {
			source = "user (overrides built-in)"
		}
		rows = append(rows, toRow(b, source))
	}
	return rows, nil
}

func toRow(b corebundle.Bundle, source string) bundleSummaryRow {
	return bundleSummaryRow{
		Name:         b.Name,
		Source:       source,
		Capabilities: len(b.Capabilities),
		Tags:         b.Tags,
		Description:  b.Description,
	}
}

// sourcePredicate gates rows on --builtin / --user. Caller validates
// mutual exclusion before constructing.
func sourcePredicate(builtinOnly, userOnly bool) listing.Predicate[bundleSummaryRow] {
	switch {
	case builtinOnly:
		return func(r bundleSummaryRow) bool { return r.Source == "built-in" }
	case userOnly:
		return func(r bundleSummaryRow) bool { return r.Source != "built-in" }
	default:
		return nil
	}
}
