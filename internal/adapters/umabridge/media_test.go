package umabridge

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	msgtypes "hop.top/aps/internal/core/messenger"
)

const (
	testMessageSID = "MM10000000000000000000000000000002"
	testMediaSID   = "ME10000000000000000000000000000001"
)

func incomingMMS(t *testing.T, h *harness) url.Values {
	t.Helper()
	form := incomingSMS(testMessageSID, "")
	form.Set("NumMedia", "1")
	form.Set("MediaContentType0", "image/jpeg")
	form.Set("MediaUrl0", h.twilio.srv.URL+"/2010-04-01/Accounts/"+testAccount+"/Messages/"+testMessageSID+"/Media/"+testMediaSID)
	return form
}

func TestInboundMMSIsStagedAndReachesTheActionAsAFileAttachment(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("\xff\xd8\xff\xe0jpeg-bytes")
	h := newHarnessWith(t, smsService(), replyWith("got it"), func(c *Config) { c.MediaDir = dir })
	h.twilio.setMedia(map[string][]byte{testMediaSID: payload})

	assert.Equal(t, http.StatusOK, h.postSigned(t, incomingMMS(t, h)))

	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	msg := h.exec.calls()[0].Message
	require.Len(t, msg.Attachments, 1)
	att := msg.Attachments[0]
	assert.Equal(t, "image", att.Type)
	assert.Equal(t, "image/jpeg", att.MimeType)
	assert.Equal(t, int64(len(payload)), att.SizeBytes)
	require.True(t, strings.HasPrefix(att.URL, "file://"), att.URL)
	path := strings.TrimPrefix(att.URL, "file://")
	assert.True(t, strings.HasPrefix(path, dir+string(filepath.Separator)), "media stays under the media dir")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestInboundMMSWithoutMediaDirIsSummaryOnly(t *testing.T) {
	h := newHarness(t, smsService(), func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		return &msgtypes.ExecutionResult{Status: "success"}, nil
	})
	h.twilio.setMedia(map[string][]byte{testMediaSID: []byte("bytes")})
	assert.Equal(t, http.StatusOK, h.postSigned(t, incomingMMS(t, h)))

	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	assert.Empty(t, h.exec.calls()[0].Message.Attachments)
}
