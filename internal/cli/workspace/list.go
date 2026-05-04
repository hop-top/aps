package workspace

import (
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/kit/go/console/output"
)

// workspaceSummaryRow is the table/json/yaml row shape for `aps
// workspace list`. Higher-priority columns survive narrow terminals.
type workspaceSummaryRow struct {
	ID          string `table:"ID,priority=10"           json:"id"            yaml:"id"`
	Name        string `table:"NAME,priority=9"          json:"name"          yaml:"name"`
	Status      string `table:"STATUS,priority=8"        json:"status"        yaml:"status"`
	Owner       string `table:"OWNER,priority=7"         json:"owner"         yaml:"owner"`
	MemberCount int    `table:"MEMBERS,priority=6"       json:"member_count"  yaml:"member_count"`
	Members     string `table:"MEMBER LIST,priority=5"   json:"members"       yaml:"members"`
	CreatedAt   string `table:"CREATED,priority=4"       json:"created_at"    yaml:"created_at"`
}

// NewListCmd creates the "workspace list" command.
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

			workspaces, err := mgr.List(cmd.Context(), collab.ListOptions{Limit: limit})
			if err != nil {
				return err
			}

			memberFilter, _ := cmd.Flags().GetString("member")
			ownerFilter, _ := cmd.Flags().GetString("owner")
			archived, _ := cmd.Flags().GetBool("archived")

			pred := listing.All(
				listing.MatchSlice(workspaceMemberIDs, memberFilter),
				listing.MatchString(func(w *collab.Workspace) string { return w.Config.OwnerProfileID }, ownerFilter),
				listing.BoolFlag(cmd.Flags().Changed("archived"),
					func(w *collab.Workspace) bool { return w.State == collab.StateArchived }, archived),
			)
			filtered := listing.Filter(workspaces, pred)

			rows := make([]workspaceSummaryRow, 0, len(filtered))
			for _, ws := range filtered {
				rows = append(rows, workspaceToSummaryRow(ws))
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "" {
				format = output.Table
			}
			return listing.RenderList(os.Stdout, format, rows)
		},
	}

	cmd.Flags().String("member", "", "Filter to workspaces containing this profile ID")
	cmd.Flags().String("owner", "", "Filter by workspace owner profile ID")
	cmd.Flags().Bool("archived", false, "Filter to archived (true) or non-archived (false) workspaces")

	addLimitFlag(cmd)
	return cmd
}

// workspaceMemberIDs flattens a workspace's agent list into the
// profile-id slice the listing.MatchSlice predicate expects.
func workspaceMemberIDs(w *collab.Workspace) []string {
	out := make([]string, 0, len(w.Agents))
	for _, a := range w.Agents {
		out = append(out, a.ProfileID)
	}
	return out
}

// workspaceToSummaryRow projects a Workspace into the row shape
// rendered by `aps workspace list`. Member list is comma-joined and
// truncated for table readability.
func workspaceToSummaryRow(w *collab.Workspace) workspaceSummaryRow {
	ids := workspaceMemberIDs(w)
	return workspaceSummaryRow{
		ID:          w.ID,
		Name:        w.Config.Name,
		Status:      string(w.State),
		Owner:       w.Config.OwnerProfileID,
		MemberCount: len(ids),
		Members:     truncateMembers(ids, 3),
		CreatedAt:   w.CreatedAt.Format("2006-01-02 15:04"),
	}
}

// truncateMembers joins up to max IDs with ", ", appending "+N" when
// the slice is longer.
func truncateMembers(ids []string, max int) string {
	if len(ids) <= max {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[:max], ", ") + ", +" + strconv.Itoa(len(ids)-max)
}
