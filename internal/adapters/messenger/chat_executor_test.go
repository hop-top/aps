package messenger

import (
	"context"
	"testing"
	"time"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// stubChatTurnRunner returns a fixed ChatTurnResult and captures the turn
// the executor passed in. The shape mirrors captureChatTurnRunner in
// telegram_provider_test.go but keeps this test self-contained.
type stubChatTurnRunner struct {
	turn   ChatTurn
	result *ChatTurnResult
	err    error
}

func (s *stubChatTurnRunner) RunChatTurn(_ context.Context, turn ChatTurn) (*ChatTurnResult, error) {
	s.turn = turn
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

// newChatHandoff builds the minimal ExecutionHandoff the executor needs to
// run end-to-end. The provider, service ID, channel, and sender values are
// only used to populate logging / metadata, not to drive behavior.
func newChatHandoff(text string) msgtypes.ExecutionHandoff {
	msg := &msgtypes.NormalizedMessage{
		ID:        "msg-1",
		Platform:  string(msgtypes.PlatformTelegram),
		Channel:   msgtypes.Channel{ID: "channel-1"},
		Sender:    msgtypes.Sender{ID: "sender-1"},
		Text:      text,
		Timestamp: time.Now(),
	}
	return msgtypes.ExecutionHandoff{
		ServiceID: "support-bot",
		Provider:  "telegram",
		ProfileID: "assistant",
		Message:   msg,
	}
}

func TestChatExecutor_DefaultReplyTargetsOriginatingChannel(t *testing.T) {
	runner := &stubChatTurnRunner{
		result: &ChatTurnResult{
			SessionID: "sess-1",
			ReplyText: "hello back",
		},
	}
	executor := NewChatMessageExecutor(runner, &core.ServiceConfig{})

	result, err := executor.ExecuteMessage(context.Background(), newChatHandoff("hi"))
	if err != nil {
		t.Fatalf("ExecuteMessage: %v", err)
	}
	if result == nil || result.Reply == nil {
		t.Fatalf("expected reply, got %#v", result)
	}
	if _, ok := result.Reply.Metadata["reply_destination"]; ok {
		t.Errorf("default mode must not stamp reply_destination, got %#v", result.Reply.Metadata)
	}
	if _, ok := result.Reply.Metadata["side_chat_lifecycle"]; ok {
		t.Errorf("default mode must not stamp side_chat_lifecycle, got %#v", result.Reply.Metadata)
	}
	if _, ok := result.Metadata["reply_destination"]; ok {
		t.Errorf("execution metadata must not stamp reply_destination in default mode, got %#v", result.Metadata)
	}
}

func TestChatExecutor_SideChatModeStampsReplyMetadata(t *testing.T) {
	runner := &stubChatTurnRunner{
		result: &ChatTurnResult{
			SessionID:         "sess-1",
			ReplyText:         "what's your full name?",
			ReplyDestination:  ReplyDestinationSideChat,
			SideChatLifecycle: SideChatLifecycleOpen,
		},
	}
	executor := NewChatMessageExecutor(runner, &core.ServiceConfig{})

	result, err := executor.ExecuteMessage(context.Background(), newChatHandoff("trigger"))
	if err != nil {
		t.Fatalf("ExecuteMessage: %v", err)
	}
	if result == nil || result.Reply == nil {
		t.Fatalf("expected reply, got %#v", result)
	}
	if got, want := result.Reply.Metadata["reply_destination"], "side_chat"; got != want {
		t.Errorf("reply.Metadata[reply_destination] = %v, want %q", got, want)
	}
	if got, want := result.Reply.Metadata["side_chat_lifecycle"], "open"; got != want {
		t.Errorf("reply.Metadata[side_chat_lifecycle] = %v, want %q", got, want)
	}
	if got, want := result.Metadata["reply_destination"], "side_chat"; got != want {
		t.Errorf("execution.Metadata[reply_destination] = %q, want %q", got, want)
	}
	if got, want := result.Metadata["side_chat_lifecycle"], "open"; got != want {
		t.Errorf("execution.Metadata[side_chat_lifecycle] = %q, want %q", got, want)
	}
}

func TestChatExecutor_SideChatKeepLifecycleOmitsKey(t *testing.T) {
	// SideChatLifecycleKeep is the zero value and the documented "continue
	// existing side-chat or open one if absent" default. Don't stamp a key
	// for the zero value — consumers should treat absence as keep.
	runner := &stubChatTurnRunner{
		result: &ChatTurnResult{
			ReplyText:        "follow-up",
			ReplyDestination: ReplyDestinationSideChat,
		},
	}
	executor := NewChatMessageExecutor(runner, &core.ServiceConfig{})

	result, err := executor.ExecuteMessage(context.Background(), newChatHandoff("trigger"))
	if err != nil {
		t.Fatalf("ExecuteMessage: %v", err)
	}
	if got := result.Reply.Metadata["reply_destination"]; got != "side_chat" {
		t.Errorf("reply.Metadata[reply_destination] = %v, want %q", got, "side_chat")
	}
	if _, ok := result.Reply.Metadata["side_chat_lifecycle"]; ok {
		t.Errorf("zero-value lifecycle must not stamp side_chat_lifecycle, got %#v", result.Reply.Metadata)
	}
}

func TestChatExecutor_SideChatCloseLifecycleStamped(t *testing.T) {
	runner := &stubChatTurnRunner{
		result: &ChatTurnResult{
			ReplyText:         "thanks — closing private channel",
			ReplyDestination:  ReplyDestinationSideChat,
			SideChatLifecycle: SideChatLifecycleClose,
		},
	}
	executor := NewChatMessageExecutor(runner, &core.ServiceConfig{})

	result, err := executor.ExecuteMessage(context.Background(), newChatHandoff("trigger"))
	if err != nil {
		t.Fatalf("ExecuteMessage: %v", err)
	}
	if got, want := result.Reply.Metadata["side_chat_lifecycle"], "close"; got != want {
		t.Errorf("reply.Metadata[side_chat_lifecycle] = %v, want %q", got, want)
	}
}

func TestChatExecutor_RunnerMetadataWinsOverDeliveryHints(t *testing.T) {
	// If the runner explicitly stamps reply_destination in Metadata, it
	// should override the ReplyDestination field's auto-stamp — the runner
	// has more context (e.g. a forced override for an A/B path).
	runner := &stubChatTurnRunner{
		result: &ChatTurnResult{
			ReplyText:        "override case",
			ReplyDestination: ReplyDestinationSideChat,
			Metadata: map[string]string{
				"reply_destination": "channel",
			},
		},
	}
	executor := NewChatMessageExecutor(runner, &core.ServiceConfig{})

	result, err := executor.ExecuteMessage(context.Background(), newChatHandoff("trigger"))
	if err != nil {
		t.Fatalf("ExecuteMessage: %v", err)
	}
	if got, want := result.Metadata["reply_destination"], "channel"; got != want {
		t.Errorf("runner-supplied Metadata[reply_destination] must win, got %q want %q", got, want)
	}
}
