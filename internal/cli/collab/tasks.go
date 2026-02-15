package collab

import (
	"fmt"
	"os"
	"strings"
	"time"

	collab "oss-aps-cli/internal/core/collaboration"

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
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(tasks)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks in this workspace.")
				fmt.Println()
				fmt.Println("  Send one:")
				fmt.Println("    aps collab send <agent> --action <action>")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "ID\tACTION\tFROM\tTO\tSTATUS\tAGE\n")
			for _, t := range tasks {
				age := formatAge(t.CreatedAt)
				short := shortID(t.ID)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					short,
					t.Action,
					t.SenderID,
					t.RecipientID,
					strings.ToUpper(string(t.Status)),
					age,
				)
			}
			w.Flush()

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
