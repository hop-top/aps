package a2a

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	eventqueue "github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/a2aproject/a2a-go/log"

	"oss-aps-cli/internal/core"
)

var _ a2asrv.AgentExecutor = (*Executor)(nil)

// Executor handles task execution for A2A agents
type Executor struct {
	profile *core.Profile
	storage *Storage
}

// NewExecutor creates a new Executor instance
func NewExecutor(profile *core.Profile, storage *Storage) *Executor {
	return &Executor{
		profile: profile,
		storage: storage,
	}
}

// GetProfile returns associated profile
func (e *Executor) GetProfile() *core.Profile {
	return e.profile
}

// Execute implements a2asrv.AgentExecutor interface
// Processes incoming A2A messages and translates them to APS profile commands
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	message := reqCtx.Message

	log.Info(ctx, "executing task", "task_id", reqCtx.TaskID, "profile_id", e.profile.ID)

	if message == nil {
		return fmt.Errorf("no message to execute")
	}

	if reqCtx.StoredTask == nil {
		event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)
		if err := queue.Write(ctx, event); err != nil {
			return fmt.Errorf("failed to write submitted status: %w", err)
		}
	}

	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	if err := queue.Write(ctx, event); err != nil {
		return fmt.Errorf("failed to write working status: %w", err)
	}

	for _, part := range message.Parts {
		if textPart, ok := part.(a2a.TextPart); ok {
			responseText := fmt.Sprintf("Processed: %s", textPart.Text)
			response := a2a.NewMessageForTask(a2a.MessageRoleAgent, reqCtx, a2a.TextPart{Text: responseText})
			if err := queue.Write(ctx, response); err != nil {
				return fmt.Errorf("failed to write response message: %w", err)
			}
			break
		}
	}

	event = a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, nil)
	event.Final = true
	if err := queue.Write(ctx, event); err != nil {
		return fmt.Errorf("failed to write completed status: %w", err)
	}

	return nil
}

// Cancel implements a2asrv.AgentExecutor interface
// Handles task cancellation requests
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	log.Info(ctx, "canceling task", "task_id", reqCtx.TaskID)

	errorMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: "Task canceled by request"})
	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, errorMsg)
	event.Final = true
	if err := queue.Write(ctx, event); err != nil {
		return fmt.Errorf("failed to write canceled status: %w", err)
	}

	return nil
}
