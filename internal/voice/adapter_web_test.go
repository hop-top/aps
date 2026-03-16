package voice_test

import (
	"net/http/httptest"
	"strings"
	"testing"

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
