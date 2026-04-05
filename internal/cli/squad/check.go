package squad

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	coresquad "hop.top/aps/internal/core/squad"
)

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CHECK\tSTATUS\tDETAIL")
	fmt.Fprintln(w, "─────\t──────\t──────")

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
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, status, detail)
	}
	w.Flush()

	fmt.Fprintf(os.Stdout, "\n%d/%d checks passed\n", passed, len(results))

	if passed < len(results) {
		return fmt.Errorf("%d/%d checks failed", len(results)-passed, len(results))
	}
	return nil
}
