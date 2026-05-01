package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/core"
)

func NewListTasksCmd() *cobra.Command {
	var (
		profileID string
		status    string
		format    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List A2A tasks for a profile",
		Long:  `List A2A tasks for a profile with optional filtering by status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			profile, err := loadProfile(profileID)
			if err != nil {
				return err
			}

			agentsDir, err := core.GetAgentsDir()
			if err != nil {
				return fmt.Errorf("failed to get agents directory: %w", err)
			}

			config := &a2apkg.StorageConfig{
				BasePath: filepath.Join(agentsDir, "a2a", profile.ID),
			}

			storage, err := a2apkg.NewStorage(config)
			if err != nil {
				return fmt.Errorf("failed to create storage: %w", err)
			}

			req := &a2a.ListTasksRequest{}
			resp, err := storage.List(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			// Filter by status if specified
			tasks := resp.Tasks
			if status != "" {
				filtered := make([]*a2a.Task, 0)
				for _, task := range tasks {
					if string(task.Status.State) == status {
						filtered = append(filtered, task)
					}
				}
				tasks = filtered
			}

			// Apply limit
			if limit > 0 && len(tasks) > limit {
				tasks = tasks[:limit]
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(tasks)
			default:
				return printTasksTable(tasks)
			}
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (submitted, working, completed, failed, cancelled)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of tasks returned")
	cmd.MarkFlagRequired("profile")

	return cmd
}

func printTasksTable(tasks []*a2a.Task) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTATUS\tCREATED\tMESSAGES")

	for _, task := range tasks {
		created := "N/A"
		if task.Status.Timestamp != nil {
			created = task.Status.Timestamp.Format("2006-01-02 15:04:05")
		}

		messageCount := len(task.History)

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
			task.ID,
			task.Status.State,
			created,
			messageCount,
		)
	}

	return w.Flush()
}
