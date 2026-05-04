package a2a

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
)

// a2aTaskSummaryRow is the table row shape for `aps a2a tasks list`.
//
// Recipient is best-effort: a2a.Task has no first-class recipient
// field, so it is read from task.Metadata["recipient"] when senders
// chose to record it. Empty cell when absent — JSON/YAML formats keep
// the field for downstream consumers.
//
// CreatedAt is omitted: a2a.Task carries Status.Timestamp (the latest
// update) but no creation time; we surface UpdatedAt only and leave
// CreatedAt to a future schema bump in the upstream a2a-go module.
type a2aTaskSummaryRow struct {
	ID        string `table:"ID,priority=10"        json:"id"        yaml:"id"`
	Status    string `table:"STATUS,priority=9"     json:"status"    yaml:"status"`
	Profile   string `table:"PROFILE,priority=8"    json:"profile"   yaml:"profile"`
	Recipient string `table:"RECIPIENT,priority=6" json:"recipient" yaml:"recipient"`
	Messages  int    `table:"MESSAGES,priority=5"   json:"messages"  yaml:"messages"`
	UpdatedAt string `table:"UPDATED,priority=4"    json:"updated_at" yaml:"updated_at"`
}

func NewListTasksCmd() *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List A2A tasks for a profile",
		Long: `List A2A tasks for the active profile (--profile global) with
optional filtering by status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			profileID := globals.Profile()
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}

			profile, err := loadProfile(profileID)
			if err != nil {
				return err
			}

			agentsDir, err := core.GetAgentsDir()
			if err != nil {
				return fmt.Errorf("failed to get agents directory: %w", err)
			}

			storage, err := a2apkg.NewStorage(&a2apkg.StorageConfig{
				BasePath: filepath.Join(agentsDir, "a2a", profile.ID),
			})
			if err != nil {
				return fmt.Errorf("failed to create storage: %w", err)
			}

			resp, err := storage.List(ctx, &a2a.ListTasksRequest{})
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			rows := make([]a2aTaskSummaryRow, 0, len(resp.Tasks))
			for _, t := range resp.Tasks {
				rows = append(rows, taskToSummaryRow(t, profile.ID))
			}

			pred := listing.All(
				listing.MatchString(func(r a2aTaskSummaryRow) string { return r.Status }, status),
			)
			rows = listing.Filter(rows, pred)

			return listing.RenderList(os.Stdout, globals.Format(), rows)
		},
	}

	cmd.Flags().StringVar(&status, "status", "",
		"Filter by status (submitted, working, completed, failed, cancelled)")

	return cmd
}

func taskToSummaryRow(t *a2a.Task, profileID string) a2aTaskSummaryRow {
	updated := ""
	if t.Status.Timestamp != nil {
		updated = t.Status.Timestamp.Format("2006-01-02 15:04:05")
	}

	recipient := ""
	if t.Metadata != nil {
		if v, ok := t.Metadata["recipient"].(string); ok {
			recipient = v
		}
	}

	return a2aTaskSummaryRow{
		ID:        string(t.ID),
		Status:    string(t.Status.State),
		Profile:   profileID,
		Recipient: recipient,
		Messages:  len(t.History),
		UpdatedAt: updated,
	}
}
