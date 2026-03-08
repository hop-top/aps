package capability

import (
	"fmt"
	"os"
	"text/tabwriter"

	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

func newPatternsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "patterns",
		Short: "Show smart patterns + builtin capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPatterns()
		},
	}
}

func runPatterns() error {
	fmt.Println(headerStyle.Render("Builtin Capabilities"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("NAME")+"\t"+
		tableHeader.Render("DESCRIPTION"))
	for _, b := range capability.ListBuiltins() {
		fmt.Fprintf(w, "%s\t%s\n",
			styles.KindBadge(b.Name), b.Description)
	}
	w.Flush()

	fmt.Println()
	fmt.Println(headerStyle.Render("Smart Patterns"))
	fmt.Println()

	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("TOOL")+"\t"+
		tableHeader.Render("DEFAULT PATH")+"\t"+
		tableHeader.Render("DESCRIPTION"))
	for _, p := range capability.ListSmartPatterns() {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			boldStyle.Render(p.ToolName),
			dimStyle.Render(p.DefaultPath),
			p.Description)
	}
	w.Flush()

	fmt.Printf("\n%s\n", dimStyle.Render(fmt.Sprintf(
		"%d patterns available. Use 'aps cap link <name> <tool>' for smart linking.",
		len(capability.ListSmartPatterns()))))

	return nil
}
