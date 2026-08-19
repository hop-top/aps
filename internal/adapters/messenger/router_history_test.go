package messenger

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	msgtypes "hop.top/aps/internal/core/messenger"
)

func newHistoryStore(t *testing.T) *msgtypes.SQLiteConversationStore {
	t.Helper()
	store, err := msgtypes.OpenConversationStore(filepath.Join(t.TempDir(), "conversations.db"), msgtypes.ConversationStoreOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func historySMS(id, text string) *msgtypes.NormalizedMessage {
	return &msgtypes.NormalizedMessage{
		ID:        id,
		Timestamp: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
		Platform:  string(msgtypes.PlatformSMS),
		Sender:    msgtypes.Sender{ID: "+15559990000", Name: "Caller"},
		Channel:   msgtypes.Channel{ID: "+15550001111"},
		Text:      text,
		PlatformMetadata: map[string]any{
			"service_id": "support-sms",
		},
	}
}

// actionPayloadEnvelope mirrors what a profile action reads from stdin.
type actionPayloadEnvelope struct {
	ID           string                      `json:"id"`
	Text         string                      `json:"text"`
	Platform     string                      `json:"platform"`
	Conversation msgtypes.ConversationState  `json:"conversation"`
	PriorTurns   []msgtypes.ConversationTurn `json:"prior_turns"`
}

func decodePayload(t *testing.T, raw []byte) actionPayloadEnvelope {
	t.Helper()
	var payload actionPayloadEnvelope
	require.NoError(t, json.Unmarshal(raw, &payload), "payload must stay a JSON object with the normalized message fields")
	return payload
}

func seedTurns(t *testing.T, store msgtypes.ConversationStore, texts ...string) {
	t.Helper()
	for i, text := range texts {
		msg := historySMS("seed-"+text, text)
		turn := msgtypes.NewInboundTurn(msg, "support-sms", "assistant", "reply")
		if i%2 == 1 {
			turn = msgtypes.NewOutboundTurn(msg, text, "support-sms", "assistant", "reply")
		}
		_, err := store.AppendTurn(context.Background(), turn)
		require.NoError(t, err)
	}
}

func TestMessageRouter_ExecuteAction_AttachesPriorTurnsAndThreadID(t *testing.T) {
	store := newHistoryStore(t)
	seedTurns(t, store, "hello", "hi, how can I help?")
	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor, WithConversationStore(store))

	msg := historySMS("m3", "my order is late")
	state := msg.ConversationState()

	result, err := router.ExecuteAction(context.Background(), "assistant", "reply", msg)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)

	assert.Equal(t, state.SessionID, executor.input.ThreadID, "RunInput.ThreadID carries the policy session key")

	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, "m3", payload.ID, "normalized message fields stay at the top level")
	assert.Equal(t, "my order is late", payload.Text)
	assert.Equal(t, state.ConversationID, payload.Conversation.ConversationID)
	assert.Equal(t, state.SessionID, payload.Conversation.SessionID)
	require.Len(t, payload.PriorTurns, 2)
	assert.Equal(t, "hello", payload.PriorTurns[0].Text)
	assert.Equal(t, msgtypes.TurnDirectionInbound, payload.PriorTurns[0].Direction)
	assert.Equal(t, "hi, how can I help?", payload.PriorTurns[1].Text)
	assert.Equal(t, msgtypes.TurnDirectionOutbound, payload.PriorTurns[1].Direction)
	for _, turn := range payload.PriorTurns {
		assert.NotEqual(t, "m3", turn.MessageID, "the current message is not a prior turn")
	}
}

func TestMessageRouter_ExecuteAction_RecordsInboundTurn(t *testing.T) {
	store := newHistoryStore(t)
	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor, WithConversationStore(store))

	msg := historySMS("m1", "first contact")
	_, err := router.ExecuteAction(context.Background(), "assistant", "reply", msg)
	require.NoError(t, err)

	turns, err := store.RecentTurns(context.Background(), msgtypes.ConversationQuery{ConversationID: msg.ConversationState().ConversationID})
	require.NoError(t, err)
	require.Len(t, turns, 1)
	assert.Equal(t, msgtypes.TurnDirectionInbound, turns[0].Direction)
	assert.Equal(t, "m1", turns[0].MessageID)
	assert.Equal(t, "first contact", turns[0].Text)
	assert.Equal(t, "assistant", turns[0].ProfileID)
	assert.Equal(t, "reply", turns[0].ActionName)
	assert.Equal(t, "support-sms", turns[0].ServiceID)

	// The second message sees the first as a prior turn, but not itself.
	second := historySMS("m2", "still waiting")
	_, err = router.ExecuteAction(context.Background(), "assistant", "reply", second)
	require.NoError(t, err)
	payload := decodePayload(t, executor.input.Payload)
	require.Len(t, payload.PriorTurns, 1)
	assert.Equal(t, "m1", payload.PriorTurns[0].MessageID)
}

func TestMessageRouter_ExecuteAction_PriorTurnLimit(t *testing.T) {
	store := newHistoryStore(t)
	seedTurns(t, store, "a", "b", "c", "d", "e")
	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor,
		WithConversationStore(store), WithPriorTurnLimit(2))

	_, err := router.ExecuteAction(context.Background(), "assistant", "reply", historySMS("m9", "next"))
	require.NoError(t, err)
	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"d", "e"}, priorTexts(payload), "router default keeps the newest N")

	_, err = router.ExecuteAction(context.Background(), "assistant", "reply", historySMS("m10", "next"), PriorTurnLimit(3))
	require.NoError(t, err)
	payload = decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"d", "e", "next"}, priorTexts(payload), "per-call limit overrides the router default")
}

func TestMessageRouter_ExecuteAction_SessionScopedPriorTurns(t *testing.T) {
	store := newHistoryStore(t)
	root := &msgtypes.NormalizedMessage{
		ID: "root", Platform: "slack", Sender: msgtypes.Sender{ID: "U1"},
		Channel: msgtypes.Channel{ID: "C1", Type: msgtypes.ChannelTypeGroup}, Text: "root message",
	}
	_, err := store.AppendTurn(context.Background(), msgtypes.NewInboundTurn(root, "slack-svc", "p", "a"))
	require.NoError(t, err)
	inThread := &msgtypes.NormalizedMessage{
		ID: "t1", Platform: "slack", Sender: msgtypes.Sender{ID: "U2"},
		Channel: msgtypes.Channel{ID: "C1", Type: msgtypes.ChannelTypeGroup}, Text: "thread reply",
		Thread: &msgtypes.Thread{ID: "1700.1", Type: msgtypes.ThreadTypeReply},
	}
	_, err = store.AppendTurn(context.Background(), msgtypes.NewInboundTurn(inThread, "slack-svc", "p", "a"))
	require.NoError(t, err)

	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor, WithConversationStore(store))
	next := &msgtypes.NormalizedMessage{
		ID: "t2", Platform: "slack", Sender: msgtypes.Sender{ID: "U1"},
		Channel: msgtypes.Channel{ID: "C1", Type: msgtypes.ChannelTypeGroup}, Text: "another thread reply",
		Thread: &msgtypes.Thread{ID: "1700.1", Type: msgtypes.ThreadTypeReply},
	}
	_, err = router.ExecuteAction(context.Background(), "p", "a", next)
	require.NoError(t, err)

	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"thread reply"}, priorTexts(payload), "prior turns follow the session (thread) key, not the whole channel")
	assert.Equal(t, next.ConversationState().SessionID, executor.input.ThreadID)
}

func TestMessageRouter_ExecuteAction_WithoutStoreStillAttachesIdentity(t *testing.T) {
	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor)

	msg := historySMS("m1", "hello")
	_, err := router.ExecuteAction(context.Background(), "assistant", "reply", msg)
	require.NoError(t, err)

	assert.Equal(t, msg.ConversationState().SessionID, executor.input.ThreadID)
	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, msg.ConversationState().ConversationID, payload.Conversation.ConversationID)
	assert.NotNil(t, payload.PriorTurns, "prior_turns is always an array")
	assert.Empty(t, payload.PriorTurns)
}

type failingStore struct{}

func (failingStore) AppendTurn(context.Context, msgtypes.ConversationTurn) (msgtypes.ConversationTurn, error) {
	return msgtypes.ConversationTurn{}, errors.New("disk on fire")
}

func (failingStore) RecentTurns(context.Context, msgtypes.ConversationQuery) ([]msgtypes.ConversationTurn, error) {
	return nil, errors.New("disk on fire")
}

func (failingStore) ListConversations(context.Context, msgtypes.ConversationFilter) ([]msgtypes.ConversationSummary, error) {
	return nil, errors.New("disk on fire")
}

func (failingStore) Close() error { return nil }

func TestMessageRouter_ExecuteAction_StoreFailureDoesNotBlockAction(t *testing.T) {
	executor := &fakeActionExecutor{output: "still ran"}
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), executor, WithConversationStore(failingStore{}))

	result, err := router.ExecuteAction(context.Background(), "assistant", "reply", historySMS("m1", "hello"))
	require.NoError(t, err)
	assert.Equal(t, "still ran", result.Output)
	payload := decodePayload(t, executor.input.Payload)
	assert.Empty(t, payload.PriorTurns)
}

func TestMessageRouter_RecordOutboundTurn(t *testing.T) {
	store := newHistoryStore(t)
	router := NewMessageRouterWithExecutor(&mockResolver{}, NewNormalizer(), &fakeActionExecutor{}, WithConversationStore(store))

	msg := historySMS("m1", "hello")
	_, err := router.ExecuteAction(context.Background(), "assistant", "reply", msg)
	require.NoError(t, err)
	router.recordOutboundTurn(context.Background(), msg, "Hi! How can I help?", "assistant", "reply")

	turns, err := store.RecentTurns(context.Background(), msgtypes.ConversationQuery{ConversationID: msg.ConversationState().ConversationID})
	require.NoError(t, err)
	require.Len(t, turns, 2)
	assert.Equal(t, msgtypes.TurnDirectionOutbound, turns[1].Direction)
	assert.Equal(t, "Hi! How can I help?", turns[1].Text)
	assert.Equal(t, "assistant", turns[1].SenderID)
	assert.Equal(t, "m1", turns[1].MessageID)

	// Empty replies are not turns.
	router.recordOutboundTurn(context.Background(), msg, "   ", "assistant", "reply")
	turns, err = store.RecentTurns(context.Background(), msgtypes.ConversationQuery{ConversationID: msg.ConversationState().ConversationID})
	require.NoError(t, err)
	assert.Len(t, turns, 2)
}

func TestMessageRouter_HandleMessage_ThreadsPriorTurnLimit(t *testing.T) {
	store := newHistoryStore(t)
	seedTurns(t, store, "a", "b", "c")
	executor := &fakeActionExecutor{}
	links := map[string]*msgtypes.ProfileMessengerLink{
		"support-sms:+15550001111": {ProfileID: "assistant", MessengerName: "support-sms", Enabled: true},
	}
	actions := map[string]string{"support-sms:+15550001111": "assistant=reply"}
	router := NewMessageRouterWithExecutor(&mockResolver{links: links, actions: actions}, NewNormalizer(), executor, WithConversationStore(store))

	msg := historySMS("m4", "next")
	msg.PlatformMetadata["messenger_name"] = "support-sms"
	result, err := router.HandleMessage(context.Background(), msg, PriorTurnLimit(1))
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"c"}, priorTexts(payload))
}

func priorTexts(payload actionPayloadEnvelope) []string {
	out := make([]string, 0, len(payload.PriorTurns))
	for _, turn := range payload.PriorTurns {
		out = append(out, turn.Text)
	}
	return out
}
