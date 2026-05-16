package messenger

import (
	"context"
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

const defaultChatFailureReply = "I could not process that chat message right now."

// ChatTurnRunner is the narrow bridge expected from the native APS chat
// runtime used by `aps chat`. It deliberately keeps provider delivery outside
// the chat layer; callers deliver ChatTurnResult.ReplyText through the shared
// message ProviderDelivery path.
type ChatTurnRunner interface {
	RunChatTurn(ctx context.Context, turn ChatTurn) (*ChatTurnResult, error)
}

// ChatTurn is the provider-neutral chat handoff passed from message services
// into the native profile-backed chat runtime.
type ChatTurn struct {
	ServiceID      string
	Provider       string
	ProfileID      string
	SessionID      string
	ConversationID string
	MessageID      string
	ChannelID      string
	SenderID       string
	Text           string
	Message        *msgtypes.NormalizedMessage
	Handoff        msgtypes.ExecutionHandoff
}

// ChatTurnResult is the native chat reply returned to the message runtime.
type ChatTurnResult struct {
	SessionID string
	ReplyText string
	Metadata  map[string]string
}

// ChatMessageExecutor routes message handoffs into the native chat runtime.
type ChatMessageExecutor struct {
	runner  ChatTurnRunner
	service *core.ServiceConfig
}

func NewChatMessageExecutor(runner ChatTurnRunner, service *core.ServiceConfig) *ChatMessageExecutor {
	return &ChatMessageExecutor{
		runner:  runner,
		service: service,
	}
}

func (e *ChatMessageExecutor) ExecuteMessage(ctx context.Context, handoff msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if handoff.Message == nil {
		return nil, fmt.Errorf("chat message handoff has nil message")
	}

	state := handoff.Message.ConversationState()
	turn := ChatTurn{
		ServiceID:      handoff.ServiceID,
		Provider:       handoff.Provider,
		ProfileID:      handoff.ProfileID,
		SessionID:      state.SessionID,
		ConversationID: state.ConversationID,
		MessageID:      handoff.Message.ID,
		ChannelID:      handoff.Message.Channel.ID,
		SenderID:       handoff.Message.Sender.ID,
		Text:           handoff.Message.Text,
		Message:        handoff.Message,
		Handoff:        handoff,
	}

	if e.runner == nil {
		err := fmt.Errorf("core chat runtime is not configured")
		logChatHandoffError(err, turn)
		return e.failureResult(handoff.Message, turn), nil
	}

	reply, err := e.runner.RunChatTurn(ctx, turn)
	if err != nil {
		logChatHandoffError(err, turn)
		return e.failureResult(handoff.Message, turn), nil
	}
	if reply == nil {
		err := fmt.Errorf("core chat runtime returned nil reply")
		logChatHandoffError(err, turn)
		return e.failureResult(handoff.Message, turn), nil
	}

	text := strings.TrimSpace(reply.ReplyText)
	result := &msgtypes.ExecutionResult{
		Status:   "success",
		Output:   text,
		Metadata: chatExecutionMetadata(state, reply),
	}
	if text == "" || replyMode(e.service) == "none" {
		return result, nil
	}
	result.Reply = &msgtypes.DeliveryRequest{
		Text:     text,
		Metadata: replyMetadata(handoff.Message, e.service),
	}
	return result, nil
}

func (e *ChatMessageExecutor) failureResult(msg *msgtypes.NormalizedMessage, turn ChatTurn) *msgtypes.ExecutionResult {
	result := &msgtypes.ExecutionResult{
		Status: "failed",
		Output: defaultChatFailureReply,
		Metadata: map[string]string{
			"session_id":      turn.SessionID,
			"conversation_id": turn.ConversationID,
		},
	}
	if replyMode(e.service) != "none" {
		result.Reply = &msgtypes.DeliveryRequest{
			Text:     defaultChatFailureReply,
			Metadata: replyMetadata(msg, e.service),
		}
	}
	return result
}

func chatExecutionMetadata(state msgtypes.ConversationState, reply *ChatTurnResult) map[string]string {
	metadata := map[string]string{
		"session_id":      state.SessionID,
		"conversation_id": state.ConversationID,
	}
	if reply != nil {
		if reply.SessionID != "" {
			metadata["chat_session_id"] = reply.SessionID
		}
		for key, value := range reply.Metadata {
			if strings.TrimSpace(key) != "" {
				metadata[key] = value
			}
		}
	}
	return metadata
}

func logChatHandoffError(err error, turn ChatTurn) {
	logging.GetLogger().Error("messenger chat handoff failed", err,
		"service_id", turn.ServiceID,
		"provider", turn.Provider,
		"profile_id", turn.ProfileID,
		"session_id", turn.SessionID,
		"message_id", turn.MessageID,
	)
}
