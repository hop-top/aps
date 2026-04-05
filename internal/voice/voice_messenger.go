package voice

import (
	"context"

	"charm.land/log/v2"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// MessengerVoiceHandler implements internal/adapters/messenger.VoiceHandler.
// It receives audio messages from the messenger webhook pipeline and emits ChannelSessions.
type MessengerVoiceHandler struct {
	sessions chan ChannelSession
	done     chan struct{}
}

func NewMessengerVoiceHandler() *MessengerVoiceHandler {
	return &MessengerVoiceHandler{
		sessions: make(chan ChannelSession, 8),
		done:     make(chan struct{}),
	}
}

// Sessions returns the channel of incoming voice sessions.
func (h *MessengerVoiceHandler) Sessions() <-chan ChannelSession {
	return h.sessions
}

// Close shuts down the handler.
func (h *MessengerVoiceHandler) Close() {
	close(h.done)
}

// HandleVoiceMessage implements messenger.VoiceHandler.
// Called by the messenger webhook handler when an audio attachment is detected.
func (h *MessengerVoiceHandler) HandleVoiceMessage(_ context.Context, msg *msgtypes.NormalizedMessage) error {
	audioURL := ""
	for _, att := range msg.Attachments {
		if att.Type == "audio" {
			audioURL = att.URL
			break
		}
	}
	if audioURL == "" {
		return nil
	}
	log.Info("voice: messenger audio message received", "platform", msg.Platform, "caller", msg.Sender.ID)
	sess := &messengerVoiceSession{
		platform:  msg.Platform,
		callerID:  msg.Sender.ID,
		profileID: msg.ProfileID,
		audioURL:  audioURL,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
	select {
	case h.sessions <- sess:
	case <-h.done:
	}
	return nil
}

type messengerVoiceSession struct {
	platform  string
	callerID  string
	profileID string
	audioURL  string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func (s *messengerVoiceSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *messengerVoiceSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *messengerVoiceSession) TextOut() chan<- string  { return s.textOut }
func (s *messengerVoiceSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: s.platform, CallerID: s.callerID}
}
func (s *messengerVoiceSession) Close() error {
	close(s.audioOut)
	close(s.textOut)
	return nil
}
