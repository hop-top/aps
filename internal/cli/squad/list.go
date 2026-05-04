package squad

import (
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	coresquad "hop.top/aps/internal/core/squad"
	"hop.top/kit/go/console/output"
)

// squadSummaryRow is the table/json/yaml row shape for `aps squad
// list`. Higher-priority columns survive narrow terminals.
type squadSummaryRow struct {
	ID          string `table:"ID,priority=10"           json:"id"            yaml:"id"`
	Name        string `table:"NAME,priority=9"          json:"name"          yaml:"name"`
	Type        string `table:"TYPE,priority=8"          json:"type"          yaml:"type"`
	Domain      string `table:"DOMAIN,priority=7"        json:"domain"        yaml:"domain"`
	MemberCount int    `table:"MEMBERS,priority=6"       json:"member_count"  yaml:"member_count"`
	Members     string `table:"MEMBER LIST,priority=5"   json:"members"       yaml:"members"`
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all squads",
		RunE: func(cmd *cobra.Command, args []string) error {
			memberFilter, _ := cmd.Flags().GetString("member")
			// --role filters by team-topology squad type (stream-aligned,
			// enabling, complicated-subsystem, platform). The flag name
			// stays --role per the task spec (T-0431) and matches how
			// squads are referred to organisationally; the underlying
			// field is Squad.Type.
			roleFilter, _ := cmd.Flags().GetString("role")

			squads := defaultManager.List()

			pred := listing.All(
				listing.MatchSlice(func(s coresquad.Squad) []string { return s.Members }, memberFilter),
				listing.MatchString(func(s coresquad.Squad) string { return string(s.Type) }, roleFilter),
			)
			filtered := listing.Filter(squads, pred)

			rows := make([]squadSummaryRow, 0, len(filtered))
			for _, s := range filtered {
				rows = append(rows, squadToSummaryRow(s))
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "" {
				format = output.Table
			}
			return listing.RenderList(os.Stdout, format, rows)
		},
	}

	cmd.Flags().String("member", "", "Filter to squads containing this profile ID")
	cmd.Flags().String("role", "", "Filter by squad type (stream-aligned, enabling, complicated-subsystem, platform)")

	return cmd
}

// squadToSummaryRow projects a Squad into the row shape rendered by
// `aps squad list`. Member list is comma-joined and truncated to 3
// IDs (+N suffix when more) for table readability.
func squadToSummaryRow(s coresquad.Squad) squadSummaryRow {
	return squadSummaryRow{
		ID:          s.ID,
		Name:        s.Name,
		Type:        string(s.Type),
		Domain:      s.Domain,
		MemberCount: len(s.Members),
		Members:     truncateSquadMembers(s.Members, 3),
	}
}

// truncateSquadMembers joins up to max IDs with ", ", appending "+N"
// when the slice is longer.
func truncateSquadMembers(ids []string, max int) string {
	if len(ids) <= max {
		return strings.Join(ids, ", ")
	}
	return strings.Join(ids[:max], ", ") + ", +" + strconv.Itoa(len(ids)-max)
}
