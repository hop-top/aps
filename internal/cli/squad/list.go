package squad

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all squads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
}

func runList() error {
	squads := defaultManager.List()

	if len(squads) == 0 {
		fmt.Fprintln(os.Stdout, "No squads configured.")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "  Create one:")
		fmt.Fprintln(os.Stdout,
			"    aps squad create my-squad --type=stream-aligned --domain=core")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tDOMAIN\tMEMBERS")

	for _, s := range squads {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			s.ID, s.Name, s.Type, s.Domain, len(s.Members))
	}

	return w.Flush()
}
