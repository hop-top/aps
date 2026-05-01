package workspace

import (
	"fmt"

	collab "hop.top/aps/internal/core/collaboration"

	"github.com/spf13/cobra"
)

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

			w := newTabWriter()
			fmt.Fprintf(w, "TIME\tACTOR\tEVENT\tRESOURCE\n")
			for _, e := range events {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					e.Timestamp.Format("15:04:05"),
					e.Actor,
					e.Event,
					e.Resource,
				)
			}
			w.Flush()

			return nil
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
