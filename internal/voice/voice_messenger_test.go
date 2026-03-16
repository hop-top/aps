package voice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/voice"
)

func TestMessengerVoiceHandler_AudioMessage(t *testing.T) {
	h := voice.NewMessengerVoiceHandler()
	defer h.Close()

	msg := &msgtypes.NormalizedMessage{
		ID:        "msg-1",
		Platform:  "telegram",
		ProfileID: "profile-1",
		Sender:    msgtypes.Sender{ID: "user-1"},
		Channel:   msgtypes.Channel{ID: "chan-1"},
		Attachments: []msgtypes.Attachment{
			{Type: "audio", URL: "https://example.com/voice.ogg"},
		},
	}

	err := h.HandleVoiceMessage(context.Background(), msg)
	assert.NoError(t, err)

	sess := <-h.Sessions()
	assert.Equal(t, "telegram", sess.Meta().ChannelType)
	assert.Equal(t, "user-1", sess.Meta().CallerID)
	sess.Close()
}

func TestMessengerVoiceHandler_TextOnlyIgnored(t *testing.T) {
	h := voice.NewMessengerVoiceHandler()
	defer h.Close()

	msg := &msgtypes.NormalizedMessage{
		ID:       "msg-2",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "u1"},
		Channel:  msgtypes.Channel{ID: "c1"},
		Text:     "just text",
	}

	err := h.HandleVoiceMessage(context.Background(), msg)
	assert.NoError(t, err)

	select {
	case <-h.Sessions():
		t.Fatal("expected no session for text-only message")
	default:
	}
}
