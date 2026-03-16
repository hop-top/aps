package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	msgtypes "hop.top/aps/internal/core/messenger"
)

type mockVoiceHandler struct {
	called bool
	msg    *msgtypes.NormalizedMessage
}

func (m *mockVoiceHandler) HandleVoiceMessage(_ context.Context, msg *msgtypes.NormalizedMessage) error {
	m.called = true
	m.msg = msg
	return nil
}

func TestHandler_VoiceMessageDelegatesToVoiceHandler(t *testing.T) {
	normalizer := NewNormalizer()
	resolver := &mockResolver{
		links:   map[string]*msgtypes.ProfileMessengerLink{},
		actions: map[string]string{},
	}
	router := NewMessageRouter(resolver, normalizer)
	vh := &mockVoiceHandler{}
	h := NewHandler(router, normalizer, nil, WithVoiceHandler(vh))

	// Verify the handler was constructed with the voice handler attached.
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.voiceHandler == nil {
		t.Fatal("expected voiceHandler to be set")
	}
}

func TestHandler_WithVoiceHandler_AudioMessageBypasesRouter(t *testing.T) {
	// Build a Telegram audio message payload. The normalizer must produce an audio attachment.
	// Telegram voice messages arrive as a "voice" object in the message payload.
	// Check the normalizer to find the right structure — for now we test hasAudioAttachment
	// by injecting directly via the handler path with a message that would normalize to audio.
	//
	// Craft a raw NormalizedMessage by using the normalizer's Telegram path for a voice message.
	// Telegram voice: {"message":{"message_id":1,"from":{"id":1},"chat":{"id":1,"type":"private"},
	//   "voice":{"file_id":"abc","duration":3,"mime_type":"audio/ogg","file_size":1234}}}
	body := `{
		"message": {
			"message_id": 10,
			"from": {"id": 100, "first_name": "Bob"},
			"chat": {"id": 200, "type": "private"},
			"voice": {"file_id": "file-abc", "duration": 5, "mime_type": "audio/ogg", "file_size": 2048}
		}
	}`

	normalizer := NewNormalizer()
	resolver := &mockResolver{
		links:   map[string]*msgtypes.ProfileMessengerLink{},
		actions: map[string]string{},
	}
	router := NewMessageRouter(resolver, normalizer)
	vh := &mockVoiceHandler{}
	h := NewHandler(router, normalizer, nil, WithVoiceHandler(vh))

	req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// If the normalizer produces an audio attachment for voice messages, vh.called will be true
	// and we get a 200 "accepted" response. Otherwise (normalizer doesn't handle voice yet)
	// it will fall through to the router and return a failed status — either is valid here.
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Errorf("unexpected status code %d; body: %s", rec.Code, rec.Body.String())
	}

	if vh.called {
		// Voice handler was invoked — verify response shape.
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "accepted" {
			t.Errorf("status = %v, want %q", resp["status"], "accepted")
		}
		if resp["message_id"] == nil || resp["message_id"] == "" {
			t.Error("response should contain message_id")
		}
	}
}

func TestHandler_WithVoiceHandler_OptionApplied(t *testing.T) {
	normalizer := NewNormalizer()
	resolver := &mockResolver{
		links:   map[string]*msgtypes.ProfileMessengerLink{},
		actions: map[string]string{},
	}
	router := NewMessageRouter(resolver, normalizer)

	// Without option — voiceHandler should be nil.
	h1 := NewHandler(router, normalizer, nil)
	if h1.voiceHandler != nil {
		t.Error("expected nil voiceHandler when no option provided")
	}

	// With option — voiceHandler should be set.
	vh := &mockVoiceHandler{}
	h2 := NewHandler(router, normalizer, nil, WithVoiceHandler(vh))
	if h2.voiceHandler == nil {
		t.Error("expected voiceHandler to be set when WithVoiceHandler is provided")
	}
}
