package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/core"
)

func NewGetTaskCmd() *cobra.Command {
	var (
		profileID     string
		historyLength int
		format        string
	)

	cmd := &cobra.Command{
		Use:   "get-task <task-id>",
		Short: "Get details of a specific A2A task",
		Long:  `Retrieve detailed information about a specific A2A task including its message history.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			taskID := a2a.TaskID(args[0])

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

			task, _, err := storage.Get(ctx, taskID)
			if err != nil {
				if err == a2a.ErrTaskNotFound {
					return fmt.Errorf("task not found: %s", taskID)
				}
				return fmt.Errorf("failed to get task: %w", err)
			}

			// Limit history if requested
			if historyLength > 0 && len(task.History) > historyLength {
				task.History = task.History[len(task.History)-historyLength:]
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(task)
			default:
				return printTaskDetails(task)
			}
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.Flags().IntVar(&historyLength, "history", 0, "Limit message history length")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	cmd.MarkFlagRequired("profile")

	return cmd
}

func printTaskDetails(task *a2a.Task) error {
	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Status: %s\n", task.Status.State)
	fmt.Printf("\nMessage History (%d messages):\n", len(task.History))
	fmt.Println("---")

	for i, msg := range task.History {
		fmt.Printf("\nMessage %d (ID: %s, Role: %s):\n", i+1, msg.ID, msg.Role)
		for j, part := range msg.Parts {
			switch p := part.(type) {
			case a2a.TextPart:
				fmt.Printf("  Part %d [text]: %s\n", j+1, p.Text)
			case a2a.FilePart:
				fmt.Printf("  Part %d [file]\n", j+1)
			case a2a.DataPart:
				fmt.Printf("  Part %d [data]\n", j+1)
			default:
				fmt.Printf("  Part %d [unknown]\n", j+1)
			}
		}
	}

	if len(task.Artifacts) > 0 {
		fmt.Printf("\nArtifacts (%d):\n", len(task.Artifacts))
		for id, artifact := range task.Artifacts {
			fmt.Printf("  - %d: %d parts\n", id, len(artifact.Parts))
		}
	}

	return nil
}
