package workspace

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/kit/go/console/output"

	"github.com/spf13/cobra"
)

// auditRow is the table row shape for `aps workspace audit`. T-0456 —
// moved off hand-rolled tabwriter so styled tables activate on a TTY.
type auditRow struct {
	Time     string `table:"TIME,priority=10"     json:"time"     yaml:"time"`
	Actor    string `table:"ACTOR,priority=9"     json:"actor"    yaml:"actor"`
	Event    string `table:"EVENT,priority=8"     json:"event"    yaml:"event"`
	Resource string `table:"RESOURCE,priority=7"  json:"resource" yaml:"resource"`
}

// NewAuditCmd creates the "collab audit" command.
func NewAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit [workspace]",
		Short: "Show audit trail",
		Long: `Display the audit trail for a collaboration workspace.

All state-changing operations are recorded: agent joins, task
creation, conflict resolution, context mutations, and more.

Filter with --since, --actor, --event, and --limit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			since, _ := cmd.Flags().GetString("since")
			actor, _ := cmd.Flags().GetString("actor")
			event, _ := cmd.Flags().GetString("event")
			limit, _ := cmd.Flags().GetInt("limit")

			store, err := getStorage()
			if err != nil {
				return err
			}

			auditLog := collab.NewWorkspaceAuditLog(store)

			opts := collab.AuditQueryOptions{
				Actor: actor,
				Event: event,
				Since: since,
				Limit: limit,
			}

			events, err := auditLog.Query(cmd.Context(), wsID, opts)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(events)
			}

			if len(events) == 0 {
				fmt.Println("No audit events found.")
				return nil
			}

			rows := make([]auditRow, 0, len(events))
			for _, e := range events {
				rows = append(rows, auditRow{
					Time:     e.Timestamp.Format("15:04:05"),
					Actor:    e.Actor,
					Event:    e.Event,
					Resource: e.Resource,
				})
			}
			return listing.RenderList(os.Stdout, output.Table, rows)
		},
	}

	addWorkspaceFlag(cmd)
	cmd.Flags().String("since", "", "Show events since duration (e.g. 1h, 24h)")
	cmd.Flags().String("actor", "", "Filter by actor (agent ID)")
	cmd.Flags().String("event", "", "Filter by event type (supports glob, e.g. task.*)")
	addLimitFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
