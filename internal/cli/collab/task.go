package collab

import (
	"fmt"
	"strings"

	collab "hop.top/aps/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewTaskCmd creates the "collab task" command.
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task <task-id>",
		Short: "Show task details",
		Long:  `Display detailed information about a specific inter-agent task.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			router := collab.NewMessageRouter(store, nil)

			task, err := router.Get(cmd.Context(), wsID, taskID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(task)
			}

			fmt.Printf("ID:         %s\n", task.ID)
			fmt.Printf("Action:     %s\n", task.Action)
			fmt.Printf("From:       %s\n", task.SenderID)
			fmt.Printf("To:         %s\n", task.RecipientID)
			fmt.Printf("Status:     %s\n", strings.ToUpper(string(task.Status)))
			fmt.Printf("Created:    %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated:    %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))

			if task.CompletedAt != nil {
				fmt.Printf("Completed:  %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
			}

			if task.Timeout > 0 {
				fmt.Printf("Timeout:    %s\n", task.Timeout)
			}

			if len(task.Dependencies) > 0 {
				fmt.Printf("Depends on: %s\n", strings.Join(task.Dependencies, ", "))
			}

			if len(task.Input) > 0 {
				fmt.Printf("Input:      %s\n", string(task.Input))
			}

			if len(task.Output) > 0 {
				fmt.Printf("Output:     %s\n", string(task.Output))
			}

			if task.Error != "" {
				fmt.Printf("Error:      %s\n", task.Error)
			}

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
