package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/cli/globals"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
)

func NewSendTaskCmd() *cobra.Command {
	var (
		targetProfile string
		message       string
		taskID        string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to create or continue an A2A task",
		Long: `Send a message to create a new A2A task or continue an existing task.

Example:
  aps a2a tasks send --target worker --message "Deploy application"
  aps a2a tasks send --target worker --task-id <id> --message "Continue deployment"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — A2A send is a network call to the target peer.
			if globals.IsOffline() {
				return fmt.Errorf("a2a tasks send: %w", globals.ErrOffline)
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// T-0463 — kit/console/progress on the canonical a2a send
			// per cli-conventions-with-kit.md §6.5. kit/cli wires the
			// active reporter into cmd.Context() based on --quiet /
			// --progress-format / --format.
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: "connect", Item: targetProfile})

			targetProf, err := loadProfile(targetProfile)
			if err != nil {
				return err
			}

			client, err := a2apkg.NewClient(targetProfile, targetProf)
			if err != nil {
				return fmt.Errorf("failed to create A2A client: %w", err)
			}

			msg := &a2a.Message{
				ID:   a2a.NewMessageID(),
				Role: a2a.MessageRoleUser,
				Parts: []a2a.Part{
					a2a.TextPart{Text: message},
				},
			}

			if taskID != "" {
				msg.TaskID = a2a.TaskID(taskID)
			}

			r.Emit(ctx, progress.Event{Phase: "send", Item: targetProfile})
			task, err := client.SendMessage(ctx, msg)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: "ack", Item: targetProfile, OK: &okFalse})
				return fmt.Errorf("failed to send message: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: "ack", Item: targetProfile, OK: &okTrue})

			// T-0648 — read --format from root globals; "text" is the
			// in-package fallback when no value is set.
			switch globals.Format() {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(task); err != nil {
					return fmt.Errorf("encode task: %w", err)
				}
				return nil
			default:
				fmt.Printf("Task created/updated: %s\n", task.ID)
				fmt.Printf("Status: %s\n", task.Status.State)
				if len(task.History) > 0 {
					lastMsg := task.History[len(task.History)-1]
					fmt.Printf("Last message ID: %s\n", lastMsg.ID)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message text (required)")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Existing task ID (optional, creates new if not specified)")
	if err := cmd.MarkFlagRequired("target"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("message"); err != nil {
		panic(err)
	}

	// T-0648 — kit 0.4 signature annotations. Outbound JSON-RPC call to
	// the target peer; each send mints a new message ID and (when
	// task-id is unset) a new task.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — every send mints fresh message/task IDs on the peer; a
	// preview that didn't actually hit the wire would lie about the IDs.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "send is a non-idempotent JSON-RPC call that mints fresh message and task IDs on the target peer; previewing without the wire call would invent IDs the peer never assigned."); err != nil {
		panic(err)
	}

	return cmd
}
