package messenger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

func newHistoryServiceHandler(executor *fakeActionExecutor, store msgtypes.ConversationStore, opts ...func(*Handler)) *Handler {
	normalizer := NewNormalizer()
	resolver := &serviceRouteResolver{base: &mockResolver{
		links:   map[string]*msgtypes.ProfileMessengerLink{},
		actions: map[string]string{},
	}}
	router := NewMessageRouterWithExecutor(resolver, normalizer, executor, WithConversationStore(store))
	return NewHandler(router, normalizer, nil, opts...)
}

func recentTurnsFor(t *testing.T, store msgtypes.ConversationStore, msg *msgtypes.NormalizedMessage) []msgtypes.ConversationTurn {
	t.Helper()
	turns, err := store.RecentTurns(context.Background(), msgtypes.ConversationQuery{
		ConversationID: msg.ConversationState().ConversationID,
		Limit:          100,
	})
	require.NoError(t, err)
	return turns
}

func directionsOf(turns []msgtypes.ConversationTurn) []string {
	out := make([]string, 0, len(turns))
	for _, turn := range turns {
		out = append(out, turn.Direction)
	}
	return out
}

func TestHandler_TwilioSMSWebhook_RecordsTurnsAndBoundsPriorTurnsPerService(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSMSTestService(t, "twilio", map[string]string{"history_turns": "2"})
	store := newHistoryStore(t)
	executor := &fakeActionExecutor{output: "reply one"}
	handler := newHistoryServiceHandler(executor, store)

	rec := postSignedSMSWebhook(t, handler, "first sms")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "<Message>reply one</Message>")

	// The stored conversation identity mirrors the normalized SMS message.
	probe := &msgtypes.NormalizedMessage{
		Platform: string(msgtypes.PlatformSMS),
		Sender:   msgtypes.Sender{ID: "+15550100001"},
		Channel:  msgtypes.Channel{ID: "+15550100002"},
		PlatformMetadata: map[string]any{
			"messenger_name": "sms-alerts",
		},
	}
	turns := recentTurnsFor(t, store, probe)
	require.Equal(t, []string{msgtypes.TurnDirectionInbound, msgtypes.TurnDirectionOutbound}, directionsOf(turns))
	assert.Equal(t, "first sms", turns[0].Text)
	assert.Equal(t, "reply one", turns[1].Text)
	assert.Equal(t, "assistant", turns[1].ProfileID)
	assert.Equal(t, "reply", turns[1].ActionName)
	assert.Equal(t, "sms-alerts", turns[0].ServiceID)
	assert.NotEmpty(t, executor.input.ThreadID)
	assert.Equal(t, probe.ConversationState().SessionID, executor.input.ThreadID)

	executor.output = "reply two"
	rec = postSignedSMSWebhook(t, handler, "second sms")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"first sms", "reply one"}, priorTexts(payload))

	executor.output = "reply three"
	rec = postSignedSMSWebhook(t, handler, "third sms")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	payload = decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"second sms", "reply two"}, priorTexts(payload), "history_turns=2 bounds the attached prior turns")

	turns = recentTurnsFor(t, store, probe)
	assert.Len(t, turns, 6)
}

func TestHandler_LegacyWebhook_ReplyModeNoneSkipsOutboundTurn(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSMSTestService(t, "twilio", map[string]string{"reply": "none"})
	store := newHistoryStore(t)
	executor := &fakeActionExecutor{output: "silent reply"}
	handler := newHistoryServiceHandler(executor, store)

	rec := postSignedSMSWebhook(t, handler, "hello")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "<Message>")

	probe := &msgtypes.NormalizedMessage{
		Platform:         string(msgtypes.PlatformSMS),
		Sender:           msgtypes.Sender{ID: "+15550100001"},
		Channel:          msgtypes.Channel{ID: "+15550100002"},
		PlatformMetadata: map[string]any{"messenger_name": "sms-alerts"},
	}
	turns := recentTurnsFor(t, store, probe)
	assert.Equal(t, []string{msgtypes.TurnDirectionInbound}, directionsOf(turns), "nothing was sent, so no outbound turn")
}

func saveTelegramHistoryService(t *testing.T, extra map[string]string) {
	t.Helper()
	options := map[string]string{
		"default_action":       "reply",
		"allowed_chats":        "-1001234567890",
		"webhook_secret_token": "telegram-secret",
	}
	for k, v := range extra {
		options[k] = v
	}
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env:     map[string]string{"TELEGRAM_BOT_TOKEN": "secret:TELEGRAM_BOT_TOKEN"},
		Options: options,
	}))
}

func postTelegramWebhook(t *testing.T, handler *Handler, text string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"update_id":555,"message":{"message_id":77,"date":1773230400,"from":{"id":42,"first_name":"Ada"},"chat":{"id":-1001234567890,"type":"group"},"text":"` + text + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	req.Header.Set(msgtypes.TelegramSecretTokenHeader, "telegram-secret")
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "support-bot", "telegram")
	return rec
}

func telegramProbe() *msgtypes.NormalizedMessage {
	return &msgtypes.NormalizedMessage{
		Platform:         string(msgtypes.PlatformTelegram),
		Sender:           msgtypes.Sender{ID: "42"},
		Channel:          msgtypes.Channel{ID: "-1001234567890", Type: msgtypes.ChannelTypeGroup},
		PlatformMetadata: map[string]any{"service_id": "support-bot"},
	}
}

func TestHandler_TelegramServiceWebhook_RecordsDeliveredReply(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
	saveTelegramHistoryService(t, nil)
	store := newHistoryStore(t)
	executor := &fakeActionExecutor{output: "reply from action"}
	transport := &captureTelegramTransport{response: &TelegramAPIResponse{OK: true, Result: &TelegramMessageSummary{MessageID: 9001}}}
	handler := newHistoryServiceHandler(executor, store, WithTelegramTransport(transport))

	rec := postTelegramWebhook(t, handler, "hello")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	turns := recentTurnsFor(t, store, telegramProbe())
	require.Equal(t, []string{msgtypes.TurnDirectionInbound, msgtypes.TurnDirectionOutbound}, directionsOf(turns))
	assert.Equal(t, "hello", turns[0].Text)
	assert.Equal(t, "reply from action", turns[1].Text)
	assert.Equal(t, "support-bot", turns[1].ServiceID)
	assert.Equal(t, "assistant", turns[1].ProfileID)

	rec = postTelegramWebhook(t, handler, "again")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	payload := decodePayload(t, executor.input.Payload)
	assert.Equal(t, []string{"hello", "reply from action"}, priorTexts(payload))
}

func TestHandler_TelegramServiceWebhookChatMode_RecordsTurns(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
	saveTelegramHistoryService(t, map[string]string{"default_action": "chat", "execution": "chat"})
	store := newHistoryStore(t)
	executor := &fakeActionExecutor{output: "unused"}
	chat := &captureChatTurnRunner{reply: "assistant reply from chat"}
	transport := &captureTelegramTransport{response: &TelegramAPIResponse{OK: true, Result: &TelegramMessageSummary{MessageID: 9001}}}
	handler := newHistoryServiceHandler(executor, store, WithTelegramTransport(transport), WithChatTurnRunner(chat))

	rec := postTelegramWebhook(t, handler, "hello from telegram")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, executor.input.ProfileID, "chat mode never runs the action executor")

	turns := recentTurnsFor(t, store, telegramProbe())
	require.Equal(t, []string{msgtypes.TurnDirectionInbound, msgtypes.TurnDirectionOutbound}, directionsOf(turns))
	assert.Equal(t, "hello from telegram", turns[0].Text)
	assert.Equal(t, "assistant reply from chat", turns[1].Text)
	assert.Equal(t, "chat", turns[0].ActionName)
}

func TestServiceHistoryTurns(t *testing.T) {
	assert.Equal(t, 0, serviceHistoryTurns(nil))
	assert.Equal(t, 0, serviceHistoryTurns(&core.ServiceConfig{}))
	assert.Equal(t, 0, serviceHistoryTurns(&core.ServiceConfig{Options: map[string]string{"history_turns": "abc"}}))
	assert.Equal(t, 0, serviceHistoryTurns(&core.ServiceConfig{Options: map[string]string{"history_turns": "-3"}}))
	assert.Equal(t, 7, serviceHistoryTurns(&core.ServiceConfig{Options: map[string]string{"history_turns": " 7 "}}))
}
