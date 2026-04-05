package voice

import (
	"net/http"

	"charm.land/log/v2"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebAdapter serves WebSocket voice sessions at /ws.
type WebAdapter struct {
	profileID string
	sessions  chan ChannelSession
	done      chan struct{}
}

func NewWebAdapter(profileID string) *WebAdapter {
	return &WebAdapter{
		profileID: profileID,
		sessions:  make(chan ChannelSession, 8),
		done:      make(chan struct{}),
	}
}

func (a *WebAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

// ServeHTTP handles WebSocket upgrade at /ws; all other paths return 404.
func (a *WebAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ws" {
		http.NotFound(w, r)
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("web adapter: websocket upgrade failed", "err", err)
		return
	}
	sess := newWebSession(conn, a.profileID)
	select {
	case a.sessions <- sess:
	case <-a.done:
		conn.Close()
	}
}

func (a *WebAdapter) Close() error {
	close(a.done)
	return nil
}

// webSession wraps a WebSocket connection as a ChannelSession.
type webSession struct {
	conn      *websocket.Conn
	profileID string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newWebSession(conn *websocket.Conn, profileID string) *webSession {
	s := &webSession{
		conn:      conn,
		profileID: profileID,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

func (s *webSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *webSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *webSession) TextOut() chan<- string  { return s.textOut }
func (s *webSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "web"}
}
func (s *webSession) Close() error { return s.conn.Close() }

func (s *webSession) readLoop() {
	defer close(s.audioIn)
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		s.audioIn <- msg
	}
}

func (s *webSession) writeLoop() {
	for {
		select {
		case frame, ok := <-s.audioOut:
			if !ok {
				return
			}
			if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
		case text, ok := <-s.textOut:
			if !ok {
				return
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
				return
			}
		}
	}
}
