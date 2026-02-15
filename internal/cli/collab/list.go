package collab

import (
	"fmt"
	"os"

	collab "oss-aps-cli/internal/core/collaboration"

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
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(workspaces)
			}

			if len(workspaces) == 0 {
				fmt.Println("No collaboration workspaces yet.")
				fmt.Println()
				fmt.Println("  Create your first:")
				fmt.Println("    aps collab new my-team --profile <profile>")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "NAME\tSTATE\tAGENTS\tOWNER\tCREATED\n")
			for _, ws := range workspaces {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
					ws.Config.Name,
					ws.State,
					len(ws.Agents),
					ws.Config.OwnerProfileID,
					ws.CreatedAt.Format("2006-01-02 15:04"),
				)
			}
			w.Flush()

			return nil
		},
	}

	addJSONFlag(cmd)
	addLimitFlag(cmd)

	return cmd
}
