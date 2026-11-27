package squad

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/globals"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show squad details",
		Long: `Print the full record for a single squad — ID, name,
team-topology type, domain boundary, description, the complete
member ID list (no truncation, unlike aps squad list), and the
created/updated timestamps. The squad is looked up by ID; unknown
IDs return an error from the underlying squad manager.

The output respects the global --format flag (table|json|yaml); json
and yaml emit the squad record alone.

Read-only: no state mutation. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			if format == "" {
				format = globals.Format()
			}
			return runShow(cmd.OutOrStdout(), args[0], format)
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

	switch f := strings.ToLower(format); f {
	case output.JSON, output.YAML:
		if err := output.Render(w, f, s); err != nil {
			return fmt.Errorf("render squad: %w", err)
		}
		return nil
	}

	fmt.Fprintf(w, "Squad: %s\n", s.Name)
	fmt.Fprintf(w, "  ID:          %s\n", s.ID)
	fmt.Fprintf(w, "  Type:        %s\n", s.Type)
	fmt.Fprintf(w, "  Domain:      %s\n", s.Domain)
	if s.Description != "" {
		fmt.Fprintf(w, "  Description: %s\n", s.Description)
	}
	fmt.Fprintf(w, "  Members:     %s\n", formatMembers(s.Members))
	fmt.Fprintf(w, "  Created:     %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "  Updated:     %s\n", s.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func formatMembers(members []string) string {
	if len(members) == 0 {
		return "(none)"
	}
	return strings.Join(members, ", ")
}
