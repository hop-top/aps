package squad

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/listing"
	coresquad "hop.top/aps/internal/core/squad"
	"hop.top/kit/go/console/output"
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
	return &cobra.Command{
		Use:   "check",
		Short: "Validate squad topology against the 8-item design checklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck()
		},
	}
}

func runCheck() error {
	mgr := coresquad.NewManager()

	topology := coresquad.Topology{Squads: mgr.List()}

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
	if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\n%d/%d checks passed\n", passed, len(results))

	if passed < len(results) {
		return fmt.Errorf("%d/%d checks failed", len(results)-passed, len(results))
	}
	return nil
}
