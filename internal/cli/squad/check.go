package squad

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/listing"
	coresquad "hop.top/aps/internal/core/squad"
	kitcli "hop.top/kit/go/console/cli"
)

// checkRow is the table/json/yaml row shape for `aps squad check`.
// T-0477 — moved off hand-rolled tabwriter so styled tables activate
// on a TTY via the listing wrapper. The hard-coded ─── separator row
// from the prior implementation is dropped — kit/output draws its
// own border on TTY, and the plain tabwriter renderer aligns columns
// without a manual divider.
type checkRow struct {
	Check  string `table:"CHECK,priority=10"  json:"check"  yaml:"check"`
	Status string `table:"STATUS,priority=9"  json:"status" yaml:"status"`
	Detail string `table:"DETAIL,priority=8"  json:"detail" yaml:"detail"`
}

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate squad topology against the 8-item design checklist",
		Long: `Validate the current squad topology against the design
checklist defined in the core squad package — invariants drawn
from team-topology theory (e.g. stream-aligned squad density,
platform-squad coverage, enabling-squad scope). Each check
contributes a row to the output with CHECK name, PASS/FAIL
status, and a DETAIL string. The command exits non-zero when any
check fails so it can gate CI.

Squads are read from the local squad store ($APS_DATA_PATH/squads);
a manifest that cannot be read fails the command, naming the file,
rather than being validated around. Read-only: no state mutation.
Idempotent. The output respects the global --format flag: the table
renderer works on a TTY (styled table) and non-TTY (plain tabwriter)
alike, and json/yaml emit the rows alone (the pass/fail summary then
goes to stderr).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd.OutOrStdout(), cmd.ErrOrStderr(), formatOf(cmd))
		},
	}
	// T-0648 — read-only topology validation; safely repeatable.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runCheck(w, errw io.Writer, format string) error {
	squads, err := defaultManager.ListChecked()
	if err != nil {
		return fmt.Errorf("list squads: %w", err)
	}

	topology := coresquad.Topology{Squads: squads}

	var contracts []coresquad.Contract
	var exitConditions []coresquad.ExitCondition
	var contextLoads []coresquad.ContextLoad

	results := coresquad.ValidateTopology(topology, contracts, exitConditions, contextLoads)

	rows := make([]checkRow, 0, len(results))
	passed := 0
	for _, r := range results {
		status := "FAIL"
		if r.Passed {
			status = "PASS"
			passed++
		}
		detail := r.Detail
		if detail == "" {
			detail = "-"
		}
		rows = append(rows, checkRow{
			Check:  r.Name,
			Status: status,
			Detail: detail,
		})
	}
	if err := listing.RenderList(w, format, rows); err != nil {
		return err
	}

	// The summary line is text: keep it off stdout when stdout carries a
	// json/yaml document.
	summary := w
	if structured(format) {
		summary = errw
	}
	_, _ = fmt.Fprintf(summary, "\n%d/%d checks passed\n", passed, len(results))

	if passed < len(results) {
		return fmt.Errorf("%d/%d checks failed", len(results)-passed, len(results))
	}
	return nil
}
