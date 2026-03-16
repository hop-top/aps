package voice

import (
	"github.com/charmbracelet/log"
	"hop.top/aps/internal/core/messenger"
)

// MessengerAdapter bridges APS's messenger layer into ChannelSessions.
// Voice messages (audio attachments) open a new session; text-only messages are ignored.
type MessengerAdapter struct {
	platform  string
	profileID string
	sessions  chan ChannelSession
	done      chan struct{}
}

func NewMessengerAdapter(platform, profileID string) *MessengerAdapter {
	return &MessengerAdapter{
		platform:  platform,
		profileID: profileID,
		sessions:  make(chan ChannelSession, 8),
		done:      make(chan struct{}),
	}
}

func (a *MessengerAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *MessengerAdapter) Close() error {
	close(a.done)
	return nil
}

// Deliver routes an incoming messenger message into the voice pipeline if it contains audio.
func (a *MessengerAdapter) Deliver(msg *messenger.NormalizedMessage) error {
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
	log.Info("messenger adapter: voice message received", "platform", a.platform, "caller", msg.Sender.ID)
	sess := newMessengerSession(a.platform, msg.Sender.ID, a.profileID, audioURL)
	select {
	case a.sessions <- sess:
	case <-a.done:
	}
	return nil
}

type messengerSession struct {
	platform  string
	callerID  string
	profileID string
	audioURL  string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newMessengerSession(platform, callerID, profileID, audioURL string) *messengerSession {
	return &messengerSession{
		platform:  platform,
		callerID:  callerID,
		profileID: profileID,
		audioURL:  audioURL,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
}

func (s *messengerSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *messengerSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *messengerSession) TextOut() chan<- string  { return s.textOut }
func (s *messengerSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: s.platform, CallerID: s.callerID}
}
func (s *messengerSession) Close() error {
	close(s.audioOut)
	close(s.textOut)
	return nil
}
