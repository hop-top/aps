package squad

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show squad details",
		Long: `Print the full record for a single squad — ID, name,
team-topology type, domain boundary, description, the complete
member ID list (no truncation, unlike aps squad list), and the
created/updated timestamps. The squad is looked up by ID in
$APS_DATA_PATH/squads/<id>/manifest.yaml; an unknown ID is a
"squad not found" error, and a manifest that cannot be read is an
error naming the file.

The output respects the global --format flag: json and yaml emit
the squad record alone, as one object (id, name, type, domain,
description, members, golden_path_defined, created_at, updated_at;
members is a list of profile IDs, [] when empty); any other format
prints the text view.

Read-only: no state mutation. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd.OutOrStdout(), args[0], formatOf(cmd))
		},
	}
	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runShow(w io.Writer, id, format string) error {
	s, err := defaultManager.Get(id)
	if err != nil {
		return err
	}

	if structured(format) {
		return renderStructured(w, format, newSquadRecord(s))
	}

	_, _ = fmt.Fprintf(w, "Squad: %s\n", s.Name)
	_, _ = fmt.Fprintf(w, "  ID:          %s\n", s.ID)
	_, _ = fmt.Fprintf(w, "  Type:        %s\n", s.Type)
	_, _ = fmt.Fprintf(w, "  Domain:      %s\n", s.Domain)
	if s.Description != "" {
		_, _ = fmt.Fprintf(w, "  Description: %s\n", s.Description)
	}
	_, _ = fmt.Fprintf(w, "  Members:     %s\n", formatMembers(s.Members))
	_, _ = fmt.Fprintf(w, "  Golden path: %s\n", yesNo(s.GoldenPathDefined))
	_, _ = fmt.Fprintf(w, "  Created:     %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(w, "  Updated:     %s\n", s.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func formatMembers(members []string) string {
	if len(members) == 0 {
		return "(none)"
	}
	return strings.Join(members, ", ")
}

func yesNo(b bool) string {
	if b {
		return "defined"
	}
	return "not defined"
}
