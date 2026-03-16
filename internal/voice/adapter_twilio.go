package voice

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

var twilioUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TwilioAdapter accepts Twilio Media Streams WebSocket connections at /twilio/media-stream.
type TwilioAdapter struct {
	phoneNumber string
	profileID   string
	sessions    chan ChannelSession
	done        chan struct{}
}

func NewTwilioAdapter(phoneNumber, profileID string) *TwilioAdapter {
	return &TwilioAdapter{
		phoneNumber: phoneNumber,
		profileID:   profileID,
		sessions:    make(chan ChannelSession, 8),
		done:        make(chan struct{}),
	}
}

func (a *TwilioAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *TwilioAdapter) Close() error {
	close(a.done)
	return nil
}

func (a *TwilioAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/twilio/media-stream" {
		http.NotFound(w, r)
		return
	}
	conn, err := twilioUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("twilio adapter: websocket upgrade failed", "err", err)
		return
	}
	log.Info("twilio adapter: call connected", "phone", a.phoneNumber)
	sess := newTwilioSession(conn, a.phoneNumber, a.profileID)
	select {
	case a.sessions <- sess:
	case <-a.done:
		conn.Close()
	}
}

type twilioSession struct {
	conn        *websocket.Conn
	phoneNumber string
	profileID   string
	audioIn     chan []byte
	audioOut    chan []byte
	textOut     chan string
}

func newTwilioSession(conn *websocket.Conn, phoneNumber, profileID string) *twilioSession {
	s := &twilioSession{
		conn:        conn,
		phoneNumber: phoneNumber,
		profileID:   profileID,
		audioIn:     make(chan []byte, 32),
		audioOut:    make(chan []byte, 32),
		textOut:     make(chan string, 8),
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

func (s *twilioSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *twilioSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *twilioSession) TextOut() chan<- string  { return s.textOut }
func (s *twilioSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "twilio", CallerID: s.phoneNumber}
}
func (s *twilioSession) Close() error { return s.conn.Close() }

func (s *twilioSession) readLoop() {
	defer close(s.audioIn)
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		s.audioIn <- msg
	}
}

func (s *twilioSession) writeLoop() {
	for frame := range s.audioOut {
		if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			return
		}
	}
}
