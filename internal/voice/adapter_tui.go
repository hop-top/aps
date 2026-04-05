package voice

import (
	"fmt"
	"net"
	"os"

	"charm.land/log/v2"
)

// TUIAdapter listens on a Unix domain socket for Hex TUI connections.
type TUIAdapter struct {
	socketPath string
	profileID  string
	listener   net.Listener
	sessions   chan ChannelSession
}

func NewTUIAdapter(socketPath, profileID string) (*TUIAdapter, error) {
	_ = os.Remove(socketPath) // clean up stale socket
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("tui adapter listen %s: %w", socketPath, err)
	}
	a := &TUIAdapter{
		socketPath: socketPath,
		profileID:  profileID,
		listener:   l,
		sessions:   make(chan ChannelSession, 8),
	}
	go a.acceptLoop()
	return a, nil
}

func (a *TUIAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *TUIAdapter) Close() error {
	return a.listener.Close()
}

func (a *TUIAdapter) acceptLoop() {
	for {
		conn, err := a.listener.Accept()
		if err != nil {
			return // listener closed
		}
		log.Info("tui adapter: new connection")
		a.sessions <- newTUISession(conn, a.profileID)
	}
}

// tuiSession wraps a Unix socket connection as a ChannelSession.
type tuiSession struct {
	conn      net.Conn
	profileID string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newTUISession(conn net.Conn, profileID string) *tuiSession {
	s := &tuiSession{
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

func (s *tuiSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *tuiSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *tuiSession) TextOut() chan<- string  { return s.textOut }
func (s *tuiSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "tui"}
}
func (s *tuiSession) Close() error { return s.conn.Close() }

func (s *tuiSession) readLoop() {
	defer close(s.audioIn)
	buf := make([]byte, 4096)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			return
		}
		frame := make([]byte, n)
		copy(frame, buf[:n])
		s.audioIn <- frame
	}
}

func (s *tuiSession) writeLoop() {
	for frame := range s.audioOut {
		if _, err := s.conn.Write(frame); err != nil {
			return
		}
	}
}
