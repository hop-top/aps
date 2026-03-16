package voice_test

import (
	"net/http/httptest"
	"strings"
	"testing"

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
