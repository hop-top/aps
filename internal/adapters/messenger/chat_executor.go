package messenger

import (
	"context"
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
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

	// ReplyDestination selects where the reply lands. The zero value
	// (ReplyDestinationChannel) sends the reply in the originating
	// conversation — the default for group bot replies. SideChat opens a
	// private DM with the original sender for AskUserQuestion-style turns
	// (clarifying questions, sensitive prompts).
	ReplyDestination ReplyDestination `json:"reply_destination,omitempty"`
	// SideChatLifecycle is honored only when ReplyDestination is
	// ReplyDestinationSideChat. Zero value (SideChatLifecycleKeep) continues
	// an existing side-chat or opens one if absent; Open forces a fresh
	// private channel; Close tears it down after delivering this turn.
	// Per-provider DM-opening implementations land separately.
	SideChatLifecycle SideChatLifecycle `json:"side_chat_lifecycle,omitempty"`
}

// ReplyDestination selects how a chat reply is routed back to the requester.
// Zero value is ReplyDestinationChannel (default in-channel reply); SideChat
// signals the runtime should deliver via a private DM rather than the
// originating conversation.
type ReplyDestination string

const (
	// ReplyDestinationChannel sends the reply in the originating conversation.
	// Zero-value default for group bot replies.
	ReplyDestinationChannel ReplyDestination = ""
	// ReplyDestinationSideChat opens a private DM with the original sender.
	// Used for clarifying questions, sensitive prompts, or AskUserQuestion-
	// style turns that should not surface in the group thread.
	ReplyDestinationSideChat ReplyDestination = "side_chat"
)

// SideChatLifecycle hints whether the runtime should open / keep / close
// the private side-chat for this turn. Ignored unless ReplyDestination is
// ReplyDestinationSideChat.
type SideChatLifecycle string

const (
	// SideChatLifecycleKeep (zero value) continues an existing side-chat
	// for the requester or opens one if none exists.
	SideChatLifecycleKeep SideChatLifecycle = ""
	// SideChatLifecycleOpen forces a fresh private channel for this turn,
	// even if a prior side-chat exists.
	SideChatLifecycleOpen SideChatLifecycle = "open"
	// SideChatLifecycleClose tears down the private channel after the
	// reply lands. Used for one-shot AskUserQuestion exchanges that
	// should not persist as an ongoing DM.
	SideChatLifecycleClose SideChatLifecycle = "close"
)

// ChatMessageExecutor routes message handoffs into the native chat runtime.
type ChatMessageExecutor struct {
	runner      ChatTurnRunner
	service     *core.ServiceConfig
	failureText string
}

func NewChatMessageExecutor(runner ChatTurnRunner, service *core.ServiceConfig) *ChatMessageExecutor {
	if runner == nil {
		runner = nativeChatRunner{}
	}
	return &ChatMessageExecutor{
		runner:      runner,
		service:     service,
		failureText: defaultChatFailureReply,
	}
}

type nativeChatRunner struct{}

func (nativeChatRunner) RunChatTurn(ctx context.Context, turn ChatTurn) (*ChatTurnResult, error) {
	service, err := corechat.NewService(ctx, turn.ProfileID, corechat.ServiceOptions{})
	if err != nil {
		return nil, err
	}
	reply, err := service.Send(ctx, turn.SessionID, turn.Text)
	if err != nil {
		return nil, err
	}
	return &ChatTurnResult{
		SessionID: turn.SessionID,
		ReplyText: reply.Content,
		Metadata: map[string]string{
			"profile_id": turn.ProfileID,
		},
	}, nil
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
		Status:   "completed",
		Output:   text,
		Metadata: chatExecutionMetadata(state, reply),
	}
	if text == "" || replyMode(e.service) == replyModeNone {
		return result, nil
	}
	result.Reply = &msgtypes.DeliveryRequest{
		Text:     text,
		Metadata: deliveryReplyMetadata(handoff.Message, e.service, reply),
	}
	return result, nil
}

// deliveryReplyMetadata layers delivery-mode hints onto the
// provider-specific reply metadata. Channel mode (the default) returns
// the unmodified provider metadata; SideChat mode adds reply_destination +
// side_chat_lifecycle keys so per-provider DeliverMessage paths can
// route to a private DM and honor the open/close lifecycle. Per-provider
// DM-opening implementations land separately; this only carries the
// signal across the bridge.
func deliveryReplyMetadata(msg *msgtypes.NormalizedMessage, service *core.ServiceConfig, reply *ChatTurnResult) map[string]any {
	metadata := replyMetadata(msg, service)
	if reply == nil || reply.ReplyDestination != ReplyDestinationSideChat {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["reply_destination"] = string(ReplyDestinationSideChat)
	if reply.SideChatLifecycle != SideChatLifecycleKeep {
		metadata["side_chat_lifecycle"] = string(reply.SideChatLifecycle)
	}
	return metadata
}

func (e *ChatMessageExecutor) failureResult(msg *msgtypes.NormalizedMessage, turn ChatTurn) *msgtypes.ExecutionResult {
	text := strings.TrimSpace(e.failureText)
	if text == "" {
		text = defaultChatFailureReply
	}
	result := &msgtypes.ExecutionResult{
		Status: "failed",
		Output: text,
		Metadata: map[string]string{
			"session_id":      turn.SessionID,
			"conversation_id": turn.ConversationID,
		},
	}
	if replyMode(e.service) != replyModeNone {
		result.Reply = &msgtypes.DeliveryRequest{
			Text:     text,
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
		// Surface delivery hints in the execution metadata so
		// consumers reading ExecutionResult.Metadata (audit logs, hub
		// forwarders, runtime observers) see the routing decision even
		// when the per-leg reply has been routed elsewhere.
		if reply.ReplyDestination == ReplyDestinationSideChat {
			metadata["reply_destination"] = string(ReplyDestinationSideChat)
			if reply.SideChatLifecycle != SideChatLifecycleKeep {
				metadata["side_chat_lifecycle"] = string(reply.SideChatLifecycle)
			}
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
