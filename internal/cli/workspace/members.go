package workspace

import (
	"fmt"

	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

// NewMembersCmd creates the "collab members" command.
func NewMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members [workspace]",
		Short: "List workspace members",
		Long:  `List all agents that are members of a collaboration workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			members, err := mgr.Members(cmd.Context(), wsID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(members)
			}

			if len(members) == 0 {
				fmt.Println(styles.Dim.Render("No members in workspace."))
				return nil
			}

			fmt.Printf("%s\n\n", styles.Title.Render(
				fmt.Sprintf("Members (%s)", wsID)))

			w := newTabWriter()
			fmt.Fprintln(w, collabTableHeader.Render("AGENT")+"\t"+
				collabTableHeader.Render("ROLE")+"\t"+
				collabTableHeader.Render("STATUS")+"\t"+
				collabTableHeader.Render("LAST SEEN"))
			for _, m := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					m.ProfileID,
					styles.RoleBadge(string(m.Role)),
					m.Status,
					styles.Dim.Render(m.LastSeen.Format("2006-01-02 15:04:05")),
				)
			}
			w.Flush()

			fmt.Printf("\n%s\n", styles.Dim.Render(
				fmt.Sprintf("%d members", len(members))))

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
