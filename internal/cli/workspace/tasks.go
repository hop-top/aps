package workspace

import (
	"fmt"
	"os"
	"strings"
	"time"

	"hop.top/aps/internal/cli/listing"
	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"

	"github.com/spf13/cobra"
)

// taskRow is the table row shape for `aps workspace tasks`. T-0456 —
// moved off hand-rolled tabwriter so styled tables activate on a TTY.
type taskRow struct {
	ID     string `table:"ID,priority=10"      json:"id"           yaml:"id"`
	Action string `table:"ACTION,priority=9"   json:"action"       yaml:"action"`
	From   string `table:"FROM,priority=8"     json:"sender_id"    yaml:"sender_id"`
	To     string `table:"TO,priority=7"       json:"recipient_id" yaml:"recipient_id"`
	Status string `table:"STATUS,priority=6"   json:"status"       yaml:"status"`
	Age    string `table:"AGE,priority=5"      json:"age"          yaml:"age"`
}

// NewTasksCmd creates the "collab tasks" command.
func NewTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks [workspace]",
		Short: "List tasks in a workspace",
		Long:  `List inter-agent tasks in a collaboration workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			status, _ := cmd.Flags().GetString("status")
			agent, _ := cmd.Flags().GetString("agent")

			filters := make(map[string]string)
			if status != "" {
				filters["status"] = status
			}
			if agent != "" {
				filters["agent"] = agent
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			router := collab.NewMessageRouter(store, nil)

			opts := collab.ListOptions{
				Limit:   limit,
				Filters: filters,
			}

			tasks, err := router.List(cmd.Context(), wsID, opts)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(tasks)
			}

			if len(tasks) == 0 {
				fmt.Println(styles.Dim.Render("No tasks in this workspace."))
				fmt.Println()
				fmt.Println(styles.Dim.Render("  Send one:"))
				fmt.Println(styles.Dim.Render("    aps workspace send <agent> --action <action>"))
				return nil
			}

			fmt.Printf("%s\n\n", styles.Title.Render(
				fmt.Sprintf("Tasks (%s)", wsID)))

			rows := make([]taskRow, 0, len(tasks))
			for _, t := range tasks {
				rows = append(rows, taskRow{
					ID:     shortID(t.ID),
					Action: t.Action,
					From:   t.SenderID,
					To:     t.RecipientID,
					Status: strings.ToUpper(string(t.Status)),
					Age:    formatAge(t.CreatedAt),
				})
			}
			if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
				return err
			}

			fmt.Printf("\n%s\n", styles.Dim.Render(
				fmt.Sprintf("%d tasks", len(tasks))))

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)
	addLimitFlag(cmd)
	cmd.Flags().String("status", "", "Filter by status (submitted, working, completed, failed, cancelled)")
	cmd.Flags().String("agent", "", "Filter by agent (sender or recipient)")

	return cmd
}

// shortID returns the first 8 characters of an ID for display.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// formatAge returns a human-readable duration since the given time.
func formatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
