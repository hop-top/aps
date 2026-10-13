package umabridge

import (
	"context"
	"fmt"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

// StatusReplyDisabled is the execution status of a message whose action
// produced a reply on a service whose adapter cannot send one: the reply was
// refused, not sent.
const StatusReplyDisabled = "reply_disabled"

func errReceiveOnly(key string) error {
	return fmt.Errorf("uma bridge: %s receives only; replies are disabled", key)
}

// receiveOnlyExecutor refuses the replies of a receive-only adapter's service
// (http.mail). The action runs as on any service; a reply it produces is
// taken off the result before delivery, so nothing reaches uma, and the
// refusal is logged and reported as StatusReplyDisabled, which the host
// records as the service's last outbound event. No outbound turn is written:
// history holds a reply only once uma accepted it.
type receiveOnlyExecutor struct {
	next msgtypes.MessageExecutor
	key  string
}

func (e receiveOnlyExecutor) ExecuteMessage(ctx context.Context, handoff msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	result, err := e.next.ExecuteMessage(ctx, handoff)
	if err != nil || result == nil || result.Reply == nil {
		return result, err //nolint:wrapcheck // a decorator: the action's own outcome passes through
	}
	messageID := ""
	if handoff.Message != nil {
		messageID = handoff.Message.ID
	}
	logging.GetLogger().Warn("uma reply not sent: adapter receives only, replies are disabled",
		"service_id", handoff.ServiceID, "message_id", messageID, "uma_adapter", e.key,
		"action", handoff.ActionName, "action_status", result.Status)
	refused := *result
	refused.Reply = nil
	refused.Status = StatusReplyDisabled
	return &refused, nil
}
