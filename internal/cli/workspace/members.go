package workspace

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"

	"github.com/spf13/cobra"
)

// memberRow is the table/json/yaml row shape for `aps workspace members`.
// T-0456 — moved off hand-rolled tabwriter so the kit-themed styled
// renderer activates on a TTY. Higher-priority columns survive narrow
// terminals.
type memberRow struct {
	Agent    string `table:"AGENT,priority=10"     json:"agent"      yaml:"agent"`
	Role     string `table:"ROLE,priority=9"       json:"role"       yaml:"role"`
	Status   string `table:"STATUS,priority=8"     json:"status"     yaml:"status"`
	LastSeen string `table:"LAST SEEN,priority=7"  json:"last_seen"  yaml:"last_seen"`
}

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

			rows := make([]memberRow, 0, len(members))
			for _, m := range members {
				rows = append(rows, memberRow{
					Agent:    m.ProfileID,
					Role:     styles.RoleBadge(string(m.Role)),
					Status:   m.Status,
					LastSeen: m.LastSeen.Format("2006-01-02 15:04:05"),
				})
			}
			if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
				return err
			}

			fmt.Printf("\n%s\n", styles.Dim.Render(
				fmt.Sprintf("%d members", len(members))))

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
