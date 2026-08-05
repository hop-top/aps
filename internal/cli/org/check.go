package org

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	coreorg "hop.top/aps/internal/core/org"
	kitcli "hop.top/kit/go/console/cli"
)

// findingKindUnloadable classifies a profile.yaml that exists on disk
// but failed to load. It complements the core org.FindingKind set: the
// core package never touches disk, so load failures are surfaced here.
const findingKindUnloadable = "unloadable_profile"

// checkRow is the table/json/yaml row shape for `aps org check`.
type checkRow struct {
	Kind     string `table:"KIND,priority=10"     json:"kind"     yaml:"kind"`
	Profiles string `table:"PROFILES,priority=9"  json:"profiles" yaml:"profiles"`
	Message  string `table:"MESSAGE,priority=8"   json:"message"  yaml:"message"`
}

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate the reporting hierarchy for consistency",
		Long: `Load every profile on disk and validate the reporting
hierarchy: cycles, dangling reports_to references, self-references,
unknown type values, and profile.yaml files that fail to load
(surfaced as unloadable_profile findings instead of being silently
skipped). Each finding contributes a row with its KIND, the affected
PROFILES, and a human-readable MESSAGE.

The command exits non-zero when any finding exists and zero on a
clean hierarchy, so it can gate CI. Output respects the global
--format flag (table|json|yaml).

Read-only: no state mutation. Idempotent.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, _ := cmd.Flags().GetString("format")
			return runCheck(format)
		},
	}
	// Read-only hierarchy validation; safely repeatable.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runCheck(format string) error {
	profiles, failures, err := loadAllProfiles()
	if err != nil {
		return err
	}

	findings := coreorg.Build(profiles).Validate()

	rows := make([]checkRow, 0, len(findings)+len(failures))
	for _, f := range findings {
		rows = append(rows, checkRow{
			Kind:     string(f.Kind),
			Profiles: strings.Join(f.ProfileIDs, ", "),
			Message:  f.Message,
		})
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].ID < failures[j].ID })
	for _, f := range failures {
		rows = append(rows, checkRow{
			Kind:     findingKindUnloadable,
			Profiles: f.ID,
			Message:  fmt.Sprintf("profile %q failed to load: %v", f.ID, f.Err),
		})
	}

	if err := listing.RenderList(os.Stdout, format, rows); err != nil {
		return err
	}
	if len(rows) > 0 {
		return fmt.Errorf("%d org finding(s)", len(rows))
	}
	return nil
}
