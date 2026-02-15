package collab

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewConflictsCmd creates the "collab conflicts" command.
func NewConflictsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflicts [workspace]",
		Short: "List conflicts in a workspace",
		Long:  `List active conflicts detected in a collaboration workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			conflicts, err := store.LoadConflicts(wsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			// Filter to unresolved conflicts
			var active []any
			type conflictRow struct {
				id       string
				typ      string
				agents   string
				resource string
				age      string
			}
			var rows []conflictRow

			for _, c := range conflicts {
				if c.IsResolved() {
					continue
				}
				active = append(active, c)
				rows = append(rows, conflictRow{
					id:       shortID(c.ID),
					typ:      string(c.Type),
					agents:   strings.Join(c.AgentIDs, ", "),
					resource: c.Resource,
					age:      formatAge(c.DetectedAt),
				})
			}

			if isJSON(cmd) {
				return outputJSON(active)
			}

			if len(rows) == 0 {
				fmt.Println("No conflicts. All clear.")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "ID\tTYPE\tAGENTS\tRESOURCE\tAGE\n")
			for _, r := range rows {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					r.id,
					r.typ,
					r.agents,
					r.resource,
					r.age,
				)
			}
			w.Flush()

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
