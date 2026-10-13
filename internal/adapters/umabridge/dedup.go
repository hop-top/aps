package umabridge

import (
	"context"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

// StatusDuplicate is the execution status of a message the bridge did not
// run because its reply is already recorded.
const StatusDuplicate = "duplicate"

// dedupExecutor keeps a provider event that was already answered from being
// run and answered again. uma's own de-duplication of provider events lives
// in the engine's memory, so it ends with the process: the provider
// redelivers an event it never saw acknowledged (the restart itself, a bridge
// rebuilt on a configuration change), and the engine records it as new.
//
// aps history outlives the engine. A reply is recorded as an outbound turn
// carrying the provider id of the message it answered, so that turn is the
// proof the event was handled. Only that turn counts: an event whose action
// failed or sent no reply has none, and runs again.
//
// The check runs after normalization and the service's allowlists, so a
// refused sender is refused as before, and before the action. A history that
// cannot be read lets the message through, as the router does: routing is
// never blocked by history.
type dedupExecutor struct {
	next    msgtypes.MessageExecutor
	history msgtypes.ConversationStore
}

func (d dedupExecutor) ExecuteMessage(ctx context.Context, handoff msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	if d.answered(ctx, handoff) {
		logging.GetLogger().Info("uma message already answered; not run again",
			"service_id", handoff.ServiceID, "message_id", handoff.Message.ID)
		return &msgtypes.ExecutionResult{Status: StatusDuplicate}, nil
	}
	return d.next.ExecuteMessage(ctx, handoff)
}

// answered reports whether history holds the reply to handoff's message. The
// conversation key embeds the service id, so the lookup never crosses
// services.
func (d dedupExecutor) answered(ctx context.Context, handoff msgtypes.ExecutionHandoff) bool {
	msg := handoff.Message
	if msg == nil || msg.ID == "" {
		return false
	}
	state := msg.ConversationState()
	turns, err := d.history.RecentTurns(ctx, msgtypes.ConversationQuery{
		ConversationID: state.ConversationID,
		SessionID:      state.SessionID,
		Limit:          msgtypes.DefaultConversationRetention,
	})
	if err != nil {
		logging.GetLogger().Warn("uma bridge: history unreadable; not checking for a duplicate",
			"service_id", handoff.ServiceID, "message_id", msg.ID, "error", err)
		return false
	}
	for _, turn := range turns {
		if turn.Direction == msgtypes.TurnDirectionOutbound && turn.MessageID == msg.ID {
			return true
		}
	}
	return false
}
