package voice_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestTwilioAdapter_AcceptsMediaStream(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)

	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions
	assert.Equal(t, "twilio", sess.Meta().ChannelType)
	assert.Equal(t, "+15551234567", sess.Meta().CallerID)
	sess.Close()
}

func TestTwilioAdapter_NotFoundForOtherPaths(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	_, _ = adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	resp, err := srv.Client().Get(srv.URL + "/other")
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestTwilioAdapter_UpgradeFailure_DoesNotPanic(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	_, _ = adapter.Accept()
	defer adapter.Close()

	// Plain HTTP GET to /twilio/media-stream — not a WebSocket upgrade
	req := httptest.NewRequest(http.MethodGet, "/twilio/media-stream", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTwilioSession_AudioIn_ReceivesFrame(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("mulaw-audio")
	err = conn.WriteMessage(websocket.BinaryMessage, frame)
	assert.NoError(t, err)

	select {
	case received := <-sess.AudioIn():
		assert.Equal(t, frame, received)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for audio frame")
	}
	sess.Close()
}

func TestTwilioSession_AudioOut_SendsFrame(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("response-mulaw")
	sess.AudioOut() <- frame

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, frame, msg)
	sess.Close()
}
