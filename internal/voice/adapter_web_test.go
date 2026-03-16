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

func TestWebAdapter_AcceptsWebSocketConnection(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)

	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions
	assert.Equal(t, "web", sess.Meta().ChannelType)
	assert.Equal(t, "profile-1", sess.Meta().ProfileID)
	sess.Close()
}

func TestWebAdapter_UpgradeFailure_DoesNotPanic(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	_, _ = adapter.Accept()
	defer adapter.Close()

	// Plain HTTP GET to /ws — not a WebSocket upgrade, so gorilla returns 400
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebAdapter_NotFoundForNonWSPath(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	_, _ = adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	resp, err := srv.Client().Get(srv.URL + "/other")
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestWebSession_AudioIn_ReceivesFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("audio-in")
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

func TestWebSession_AudioOut_SendsFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("audio-out")
	sess.AudioOut() <- frame

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, frame, msg)
	sess.Close()
}

func TestWebSession_TextOut_SendsTextFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	sess.TextOut() <- "hello agent"

	conn.SetReadDeadline(time.Now().Add(time.Second))
	msgType, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, msgType)
	assert.Equal(t, "hello agent", string(msg))
	sess.Close()
}
