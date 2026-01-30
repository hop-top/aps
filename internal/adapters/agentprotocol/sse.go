package agentprotocol

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	closed  bool
	done    chan struct{}
}

func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEWriter{
		w:       w,
		flusher: flusher,
		done:    make(chan struct{}),
	}, nil
}

func (s *SSEWriter) Write(event string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("writer closed")
	}

	select {
	case <-s.done:
		return fmt.Errorf("writer closed")
	default:
		fmt.Fprintf(s.w, "event: %s\n", event)
		fmt.Fprintf(s.w, "data: %s\n\n", data)
		s.flusher.Flush()
		return nil
	}
}

func (s *SSEWriter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)

	return nil
}

func (s *SSEWriter) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.flusher.Flush()
	}
}

func (s *SSEWriter) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *SSEWriter) WriteEvent(eventType string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("writer closed")
	}

	fmt.Fprintf(s.w, "event: %s\n", eventType)
	fmt.Fprintf(s.w, "data: %v\n\n", data)
	s.flusher.Flush()
	return nil
}

func (s *SSEWriter) WriteComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("writer closed")
	}

	fmt.Fprintf(s.w, ": %s\n\n", comment)
	s.flusher.Flush()
	return nil
}

func (s *SSEWriter) KeepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.WriteComment("keepalive"); err != nil {
				return
			}
		case <-s.done:
			return
		}
	}
}
