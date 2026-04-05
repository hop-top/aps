package collab

import (
	"fmt"
	"strings"
	"time"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

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
				fmt.Println(styles.Dim.Render("    aps collab send <agent> --action <action>"))
				return nil
			}

			fmt.Printf("%s\n\n", styles.Title.Render(
				fmt.Sprintf("Tasks (%s)", wsID)))

			w := newTabWriter()
			fmt.Fprintln(w, collabTableHeader.Render("ID")+"\t"+
				collabTableHeader.Render("ACTION")+"\t"+
				collabTableHeader.Render("FROM")+"\t"+
				collabTableHeader.Render("TO")+"\t"+
				collabTableHeader.Render("STATUS")+"\t"+
				collabTableHeader.Render("AGE"))
			for _, t := range tasks {
				age := formatAge(t.CreatedAt)
				short := shortID(t.ID)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					short,
					t.Action,
					t.SenderID,
					t.RecipientID,
					strings.ToUpper(string(t.Status)),
					styles.Dim.Render(age),
				)
			}
			w.Flush()

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
