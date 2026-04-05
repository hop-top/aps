package collab

import (
	"fmt"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

// NewListCmd creates the "collab list" command.
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List collaboration workspaces",
		Long:  `List all collaboration workspaces with summary information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := getManager()
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")

			opts := collab.ListOptions{
				Limit: limit,
			}

			workspaces, err := mgr.List(cmd.Context(), opts)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(workspaces)
			}

			if len(workspaces) == 0 {
				fmt.Println(styles.Dim.Render("No collaboration workspaces yet."))
				fmt.Println()
				fmt.Println(styles.Dim.Render("  Create your first:"))
				fmt.Println(styles.Dim.Render("    aps collab new my-team --profile <profile>"))
				return nil
			}

			fmt.Printf("%s\n\n", styles.Title.Render("Workspaces"))

			w := newTabWriter()
			fmt.Fprintln(w, collabTableHeader.Render("NAME")+"\t"+
				collabTableHeader.Render("STATE")+"\t"+
				collabTableHeader.Render("AGENTS")+"\t"+
				collabTableHeader.Render("OWNER")+"\t"+
				collabTableHeader.Render("CREATED"))
			for _, ws := range workspaces {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
					ws.Config.Name,
					ws.State,
					len(ws.Agents),
					ws.Config.OwnerProfileID,
					styles.Dim.Render(ws.CreatedAt.Format("2006-01-02 15:04")),
				)
			}
			w.Flush()

			fmt.Printf("\n%s\n", styles.Dim.Render(
				fmt.Sprintf("%d workspaces", len(workspaces))))

			return nil
		},
	}

	addJSONFlag(cmd)
	addLimitFlag(cmd)

	return cmd
}
