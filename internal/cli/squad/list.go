package squad

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	coresquad "hop.top/aps/internal/core/squad"
	kitcli "hop.top/kit/go/console/cli"
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
		Long: `List every squad in the local squad store. Each row reports
the squad ID, name, team-topology type (stream-aligned, enabling,
complicated-subsystem, platform), domain boundary, member count,
and a truncated member-ID list (first three IDs plus "+N" when the
squad has more).

The output respects the global --format flag (table|json|yaml): json
and yaml emit an array of rows (id, name, type, domain, member_count,
members), [] when nothing matches. Squads whose manifest cannot be
read are skipped and named on stderr. The local filter flags are: --member selects squads containing a given
profile ID, and --role selects by team-topology type. Read-only:
no state mutation. Idempotent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			memberFilter, _ := cmd.Flags().GetString("member")
			// --role filters by team-topology squad type (stream-aligned,
			// enabling, complicated-subsystem, platform). The flag name
			// stays --role per the task spec (T-0431) and matches how
			// squads are referred to organisationally; the underlying
			// field is Squad.Type.
			roleFilter, _ := cmd.Flags().GetString("role")

			// A squad whose manifest cannot be read is left out of the
			// listing, as profile list does for a damaged profile, but is
			// named on stderr so it is not silently lost. Only an
			// unreadable squads directory fails the command.
			squads, err := defaultManager.ListChecked()
			var skipped *coresquad.SkippedError
			switch {
			case errors.As(err, &skipped):
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipped unreadable squads:\n%v\n", err)
			case err != nil:
				return fmt.Errorf("list squads: %w", err)
			}

			pred := listing.All(
				listing.MatchSlice(func(s coresquad.Squad) []string { return s.Members }, memberFilter),
				listing.MatchString(func(s coresquad.Squad) string { return string(s.Type) }, roleFilter),
			)
			filtered := listing.Filter(squads, pred)

			rows := make([]squadSummaryRow, 0, len(filtered))
			for _, s := range filtered {
				rows = append(rows, squadToSummaryRow(s))
			}

			return listing.RenderList(cmd.OutOrStdout(), formatOf(cmd), rows)
		},
	}

	cmd.Flags().String("member", "", "Filter to squads containing this profile ID")
	cmd.Flags().String("role", "", "Filter by squad type (stream-aligned, enabling, complicated-subsystem, platform)")

	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

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
