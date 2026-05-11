package messenger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
)

func TestTelegramProviderNormalizeIngressHandlesNativeUpdate(t *testing.T) {
	provider := NewTelegramProvider(TelegramProviderConfig{})
	body := []byte(`{
		"update_id": 555,
		"message": {
			"message_id": 77,
			"message_thread_id": 12,
			"date": 1773230400,
			"from": {"id": 42, "first_name": "Ada", "last_name": "Lovelace", "username": "ada"},
			"chat": {"id": -1001234567890, "title": "Ops", "type": "supergroup"},
			"text": "hello"
		}
	}`)

	msg, err := provider.NormalizeIngress(context.Background(), coremessenger.NativeIngress{
		ServiceID:  "support-bot",
		Provider:   "telegram",
		Mode:       coremessenger.IngressModeWebhook,
		ReceivedAt: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
		Body:       body,
	})
	if err != nil {
		t.Fatalf("NormalizeIngress: %v", err)
	}

	if msg.ID != "telegram:update:555" {
		t.Fatalf("ID = %q, want telegram:update:555", msg.ID)
	}
	if msg.Timestamp != time.Unix(1773230400, 0).UTC() {
		t.Fatalf("Timestamp = %s", msg.Timestamp)
	}
	if msg.Platform != "telegram" || msg.Channel.ID != "-1001234567890" || msg.Channel.Type != coremessenger.ChannelTypeGroup {
		t.Fatalf("message routing fields = %#v", msg)
	}
	if msg.Sender.ID != "42" || msg.Sender.Name != "Ada Lovelace" || msg.Sender.PlatformHandle != "ada" {
		t.Fatalf("sender = %#v", msg.Sender)
	}
	if msg.Thread == nil || msg.Thread.ID != "12" || msg.Thread.Type != coremessenger.ThreadTypeTopic {
		t.Fatalf("thread = %#v, want topic 12", msg.Thread)
	}
	if msg.PlatformMetadata["messenger_name"] != "support-bot" || msg.PlatformMetadata["telegram_message_id"] != "77" {
		t.Fatalf("metadata = %#v", msg.PlatformMetadata)
	}
}

func TestTelegramProviderDeliverMessageUsesSendMessageReplyTarget(t *testing.T) {
	transport := &captureTelegramTransport{response: &TelegramAPIResponse{
		OK:     true,
		Result: &TelegramMessageSummary{MessageID: 9001},
	}}
	provider := NewTelegramProvider(TelegramProviderConfig{
		BotToken:  "bot-token",
		Transport: transport,
		Now:       func() time.Time { return time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC) },
	})

	receipt, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "telegram",
		ServiceID: "support-bot",
		ChannelID: "-1001234567890",
		Text:      "ack",
		Metadata: map[string]any{
			"reply_to_message_id": "77",
			"message_thread_id":   "12",
		},
	})
	if err != nil {
		t.Fatalf("DeliverMessage: %v", err)
	}

	if transport.token != "bot-token" {
		t.Fatalf("token = %q", transport.token)
	}
	if transport.request.ChatID != int64(-1001234567890) || transport.request.Text != "ack" {
		t.Fatalf("sendMessage request = %#v", transport.request)
	}
	if transport.request.MessageThreadID != 12 {
		t.Fatalf("message_thread_id = %d, want 12", transport.request.MessageThreadID)
	}
	if transport.request.ReplyParameters == nil || transport.request.ReplyParameters.MessageID != 77 {
		t.Fatalf("reply_parameters = %#v", transport.request.ReplyParameters)
	}
	if receipt.DeliveryID != "9001" || receipt.Status != "success" {
		t.Fatalf("receipt = %#v", receipt)
	}
}

func TestHandler_TelegramServiceWebhookValidatesSecretAndDelivers(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env: map[string]string{
			"TELEGRAM_BOT_TOKEN": "secret:TELEGRAM_BOT_TOKEN",
		},
		Options: map[string]string{
			"default_action":       "reply",
			"allowed_chats":        "-1001234567890",
			"webhook_secret_token": "telegram-secret",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{output: "reply from action"}
	transport := &captureTelegramTransport{response: &TelegramAPIResponse{
		OK:     true,
		Result: &TelegramMessageSummary{MessageID: 9001},
	}}
	handler := newServiceTestHandler(executor, WithTelegramTransport(transport))
	body := `{"update_id":555,"message":{"message_id":77,"date":1773230400,"from":{"id":42,"first_name":"Ada"},"chat":{"id":-1001234567890,"type":"group"},"text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	req.Header.Set(coremessenger.TelegramSecretTokenHeader, "telegram-secret")
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "telegram")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if executor.input.ProfileID != "assistant" || executor.input.ActionID != "reply" {
		t.Fatalf("executor input = %#v, want assistant/reply", executor.input)
	}
	if transport.token != "bot-token" || transport.request.Text != "reply from action" {
		t.Fatalf("telegram sendMessage = token %q request %#v", transport.token, transport.request)
	}
	if transport.request.ReplyParameters == nil || transport.request.ReplyParameters.MessageID != 77 {
		t.Fatalf("reply target = %#v, want message 77", transport.request.ReplyParameters)
	}
	if !strings.Contains(rec.Body.String(), `"delivery"`) {
		t.Fatalf("response missing delivery: %s", rec.Body.String())
	}
}

func TestHandler_TelegramServiceWebhookChatModeDeliversNativeChatReply(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env: map[string]string{
			"TELEGRAM_BOT_TOKEN": "secret:TELEGRAM_BOT_TOKEN",
		},
		Options: map[string]string{
			"default_action":       "chat",
			"execution":            "chat",
			"allowed_chats":        "-1001234567890",
			"webhook_secret_token": "telegram-secret",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{output: "action output should not be used"}
	chat := &captureChatTurnRunner{reply: "assistant reply from chat"}
	transport := &captureTelegramTransport{response: &TelegramAPIResponse{
		OK:     true,
		Result: &TelegramMessageSummary{MessageID: 9001},
	}}
	handler := newServiceTestHandler(executor, WithTelegramTransport(transport), WithChatTurnRunner(chat))
	body := `{"update_id":555,"message":{"message_id":77,"date":1773230400,"from":{"id":42,"first_name":"Ada"},"chat":{"id":-1001234567890,"type":"group"},"text":"hello from telegram"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	req.Header.Set(coremessenger.TelegramSecretTokenHeader, "telegram-secret")
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "telegram")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if executor.input.ProfileID != "" || executor.input.ActionID != "" {
		t.Fatalf("action executor input = %#v, want no action execution", executor.input)
	}
	if chat.turn.ProfileID != "assistant" || chat.turn.Text != "hello from telegram" {
		t.Fatalf("chat turn = %#v", chat.turn)
	}
	if chat.turn.SessionID == "" || chat.turn.SessionID != chat.turn.Message.ConversationState().SessionID {
		t.Fatalf("chat session = %q, want ConversationState session", chat.turn.SessionID)
	}
	if transport.token != "bot-token" || transport.request.Text != "assistant reply from chat" {
		t.Fatalf("telegram sendMessage = token %q request %#v", transport.token, transport.request)
	}
	if transport.request.ReplyParameters == nil || transport.request.ReplyParameters.MessageID != 77 {
		t.Fatalf("reply target = %#v, want message 77", transport.request.ReplyParameters)
	}
	if !strings.Contains(rec.Body.String(), `"delivery"`) {
		t.Fatalf("response missing delivery: %s", rec.Body.String())
	}
}

func TestHandler_TelegramServiceWebhookRejectsMissingSecret(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env: map[string]string{
			"TELEGRAM_BOT_TOKEN": "token",
		},
		Options: map[string]string{
			"default_action":       "reply",
			"webhook_secret_token": "telegram-secret",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	transport := &captureTelegramTransport{}
	handler := newServiceTestHandler(executor, WithTelegramTransport(transport))
	body := `{"update_id":555,"message":{"message_id":77,"from":{"id":42},"chat":{"id":-1001234567890,"type":"group"},"text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "telegram")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor input = %#v, want no execution", executor.input)
	}
	if transport.request.Text != "" {
		t.Fatalf("telegram transport was called: %#v", transport.request)
	}
}

func TestHandler_TelegramServiceWebhookRejectsDisallowedChat(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env: map[string]string{
			"TELEGRAM_BOT_TOKEN": "token",
		},
		Options: map[string]string{
			"default_action":       "reply",
			"allowed_chats":        "-1001234567890",
			"webhook_secret_token": "telegram-secret",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	transport := &captureTelegramTransport{}
	handler := newServiceTestHandler(executor, WithTelegramTransport(transport))
	body := `{"update_id":555,"message":{"message_id":77,"from":{"id":42},"chat":{"id":-1009999999999,"type":"group"},"text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	req.Header.Set(coremessenger.TelegramSecretTokenHeader, "telegram-secret")
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "telegram")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor input = %#v, want no execution", executor.input)
	}
	if transport.request.Text != "" {
		t.Fatalf("telegram transport was called: %#v", transport.request)
	}
}

type captureTelegramTransport struct {
	token    string
	request  TelegramSendMessageRequest
	response *TelegramAPIResponse
	err      error
}

func (t *captureTelegramTransport) SendMessage(_ context.Context, botToken string, req TelegramSendMessageRequest) (*TelegramAPIResponse, error) {
	t.token = botToken
	t.request = req
	if t.err != nil {
		return nil, t.err
	}
	if t.response != nil {
		return t.response, nil
	}
	return &TelegramAPIResponse{OK: true, Result: &TelegramMessageSummary{MessageID: 1}}, nil
}

type captureChatTurnRunner struct {
	turn  ChatTurn
	reply string
	err   error
}

func (r *captureChatTurnRunner) RunChatTurn(_ context.Context, turn ChatTurn) (*ChatTurnResult, error) {
	r.turn = turn
	if r.err != nil {
		return nil, r.err
	}
	return &ChatTurnResult{
		SessionID: turn.SessionID,
		ReplyText: r.reply,
	}, nil
}
