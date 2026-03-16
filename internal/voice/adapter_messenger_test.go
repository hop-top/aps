package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/voice"
)

func TestMessengerAdapter_RouteVoiceMessage(t *testing.T) {
	adapter := voice.NewMessengerAdapter("telegram", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	msg := &messenger.NormalizedMessage{
		ID:       "msg-1",
		Platform: "telegram",
		Sender:   messenger.Sender{ID: "user-1", Name: "Alice"},
		Channel:  messenger.Channel{ID: "chan-1"},
		Attachments: []messenger.Attachment{
			{Type: "audio", URL: "https://example.com/voice.ogg", MimeType: "audio/ogg"},
		},
	}

	err = adapter.Deliver(msg)
	assert.NoError(t, err)

	sess := <-sessions
	assert.Equal(t, "telegram", sess.Meta().ChannelType)
	assert.Equal(t, "user-1", sess.Meta().CallerID)
	sess.Close()
}

func TestMessengerAdapter_IgnoresNonAudioMessages(t *testing.T) {
	adapter := voice.NewMessengerAdapter("telegram", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	msg := &messenger.NormalizedMessage{
		ID:       "msg-2",
		Platform: "telegram",
		Sender:   messenger.Sender{ID: "u1"},
		Channel:  messenger.Channel{ID: "c1"},
		Text:     "just text, no audio",
	}
	err = adapter.Deliver(msg)
	assert.NoError(t, err)

	// no session should be emitted
	select {
	case <-sessions:
		t.Fatal("expected no session for text-only message")
	default:
		// correct
	}
}
