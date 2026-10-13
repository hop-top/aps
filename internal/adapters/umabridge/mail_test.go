package umabridge

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

const mailToken = "mail-relay-token-0123456789" // #nosec G101 -- fixture value

var mailSecrets = secrets(map[string]string{"MAIL_RELAY_TOKEN": mailToken})

// mailService is the native email relay moved to uma: bearer auth from a
// binding, receive only.
func mailService() *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: "mail-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{"MAIL_RELAY_TOKEN": "secret:MAIL_RELAY_TOKEN"},
		Options: map[string]string{
			core.OptionUmaAdapter:     core.UmaAdapterHTTPMail,
			core.OptionRecipients:     "support@example.com,help@example.com",
			core.OptionFormats:        "json,rfc822",
			core.OptionAuthScheme:     core.AuthSchemeBearer,
			core.OptionAuthTokenEnv:   "MAIL_RELAY_TOKEN",
			core.OptionAllowedSenders: "*@example.org",
			"default_action":          "triage",
			"receive":                 "webhook",
			"reply":                   "none",
		},
	}
}

// postMail posts body as the relay would and returns the status and answer.
func (h *bridgeHarness) postMail(t *testing.T, contentType, token string, body []byte) (int, string) {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.hook.URL, bytes.NewReader(body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", contentType)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(r)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	answer, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(answer)
}

func (h *bridgeHarness) mailStatus(t *testing.T, contentType, token string, body []byte) int {
	t.Helper()
	status, _ := h.postMail(t, contentType, token, body)
	return status
}

func jsonMail(t *testing.T, fields map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(fields)
	require.NoError(t, err)
	return body
}

// nativeMail is the flat JSON the native email relay takes: the same
// members http.mail reads by default.
func nativeMail(subject, body string, attachments ...map[string]any) map[string]any {
	m := map[string]any{"from": "Alice <alice@example.org>", "to": "support@example.com", "subject": subject, "body": body}
	if len(attachments) > 0 {
		m["attachments"] = attachments
	}
	return m
}

func TestMail_JSONRunsTheActionWithStagedAndDescribedAttachments(t *testing.T) {
	media := t.TempDir()
	h := newBridgeHarness(t, mailService(), mailSecrets, noReply, func(c *Config) { c.MediaDir = media })

	payload := nativeMail("Order 42", "where is it?",
		map[string]any{
			"type": "file", "name": "note.txt", "mime_type": "text/plain",
			"content": base64.StdEncoding.EncodeToString([]byte("hello attachment")),
		},
		map[string]any{"type": "image", "url": "https://cdn.example.org/x.png", "mime_type": "image/png", "size_bytes": 9},
	)
	status, answer := h.postMail(t, "application/json", mailToken, jsonMail(t, payload))
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"accepted":1}`, answer, "the answer carries no reply")

	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	call := h.exec.calls()[0]
	msg := call.Message
	assert.Equal(t, "email", call.Provider)
	assert.Equal(t, "email", msg.Platform)
	assert.Equal(t, "alice@example.org", msg.Sender.ID)
	assert.Equal(t, "support@example.com", msg.Channel.ID, "the recipient, as the native relay keyed it")
	require.NotNil(t, msg.Thread)
	assert.Equal(t, msgtypes.Thread{ID: "Order 42", Type: msgtypes.ThreadTypeTopic}, *msg.Thread)
	assert.Equal(t, "where is it?", msg.Text)
	assert.Equal(t, "Order 42", msg.PlatformMetadata["subject"])
	assert.Equal(t, "unverified", msg.PlatformMetadata["sender_assurance"], "nothing a request says is evidence")

	require.Len(t, msg.Attachments, 2)
	staged := msg.Attachments[0]
	require.True(t, strings.HasPrefix(staged.URL, "file://"+media+"/inbound/"), staged.URL)
	data, err := os.ReadFile(strings.TrimPrefix(staged.URL, "file://"))
	require.NoError(t, err)
	assert.Equal(t, "hello attachment", string(data))
	assert.Equal(t, "text/plain", staged.MimeType)
	described := msg.Attachments[1]
	assert.Equal(t, msgtypes.Attachment{Type: "image", URL: "https://cdn.example.org/x.png", MimeType: "image/png", SizeBytes: 9}, described,
		"a URL is passed through, never fetched")
}

func TestMail_RFC822StagesTheMessageAttachments(t *testing.T) {
	media := t.TempDir()
	h := newBridgeHarness(t, mailService(), mailSecrets, noReply, func(c *Config) { c.MediaDir = media })

	raw := strings.Join([]string{
		"From: Bob <bob@example.org>",
		"To: help@example.com",
		"Subject: Invoice",
		"Message-ID: <m1@example.org>",
		"MIME-Version: 1.0",
		`Content-Type: multipart/mixed; boundary="b1"`,
		"",
		"--b1",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"see attached",
		"--b1",
		"Content-Type: application/pdf",
		`Content-Disposition: attachment; filename="inv.pdf"`,
		"Content-Transfer-Encoding: base64",
		"",
		base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 fake")),
		"--b1--",
		"",
	}, "\r\n")
	require.Equal(t, http.StatusOK, h.mailStatus(t, "message/rfc822", mailToken, []byte(raw)))

	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	msg := h.exec.calls()[0].Message
	assert.Equal(t, "bob@example.org", msg.Sender.ID)
	assert.Equal(t, "help@example.com", msg.Channel.ID)
	assert.Equal(t, "see attached", strings.TrimSpace(msg.Text))
	assert.NotEqual(t, "<m1@example.org>", msg.ID, "the Message-ID is the sender's, not an identity")
	require.Len(t, msg.Attachments, 1)
	data, err := os.ReadFile(strings.TrimPrefix(msg.Attachments[0].URL, "file://"))
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 fake", string(data))
	assert.True(t, strings.HasPrefix(msg.Attachments[0].URL, "file://"+filepath.Join(media, "inbound")))
}

func TestMail_UnauthenticatedRequestsNeverReachTheAction(t *testing.T) {
	h := newBridgeHarness(t, mailService(), mailSecrets, noReply, nil)
	body := jsonMail(t, nativeMail("s", "b"))
	assert.Equal(t, http.StatusUnauthorized, h.mailStatus(t, "application/json", "", body))
	assert.Equal(t, http.StatusUnauthorized, h.mailStatus(t, "application/json", "wrong-token-0123456789", body))
	assert.Equal(t, http.StatusUnsupportedMediaType,
		h.mailStatus(t, "application/x-www-form-urlencoded", mailToken, []byte("from=a%40example.org")),
		"form bodies are not read")
	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, h.exec.calls())
}

func TestMail_TrustedAuthservIDRaisesTheAssurance(t *testing.T) {
	svc := mailService()
	svc.Options[core.OptionTrustedAuthservID] = "mx.example.com"
	svc.Options[core.OptionAuthResultsField] = "auth_results"
	h := newBridgeHarness(t, svc, mailSecrets, noReply, nil)

	payload := nativeMail("s", "b")
	payload["auth_results"] = "mx.example.com; spf=pass smtp.mailfrom=example.org; dkim=pass header.d=example.org; " +
		"dmarc=pass header.from=example.org"
	require.Equal(t, http.StatusOK, h.mailStatus(t, "application/json", mailToken, jsonMail(t, payload)))
	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	assert.Equal(t, "dmarc_pass", h.exec.calls()[0].Message.PlatformMetadata["sender_assurance"])
}

// TestMail_ReplyIsRefusedNotSent: an action that replies on a receive-only
// service sends nothing, records no outbound turn, and says so in the log and
// the result the host turns into the service's last outbound event.
func TestMail_ReplyIsRefusedNotSent(t *testing.T) {
	var logs bytes.Buffer
	previous := logging.GetLogger()
	logging.SetLogger(logging.NewLoggerTo(&logs))
	t.Cleanup(func() { logging.SetLogger(previous) })

	history := openHistory(t, historyFile(t))
	svc := mailService()
	svc.Options["reply"] = "text"
	h := newBridgeHarness(t, svc, mailSecrets, replyWith("thanks, on it"), withHistory(history))

	require.Equal(t, http.StatusOK, h.mailStatus(t, "application/json", mailToken, jsonMail(t, nativeMail("Order 7", "hi"))))
	waitFor(t, "result", func() bool { return len(h.outcomes()) == 1 })

	out := h.outcomes()[0]
	require.NoError(t, out.err, "a refused reply is not a failure of the message")
	require.NotNil(t, out.result)
	assert.Nil(t, out.result.Delivery, "nothing was delivered")
	assert.Empty(t, out.result.DeliveryAttempts, "nothing was attempted")
	require.NotNil(t, out.result.Result)
	assert.Equal(t, StatusReplyDisabled, out.result.Result.Status)
	assert.Nil(t, out.result.Result.Reply)
	assert.Empty(t, outboundTurns(turnsOf(t, history, out.result.Message)), "no turn for a reply nobody received")

	line := logs.String()
	assert.Contains(t, line, "uma reply not sent")
	assert.Contains(t, line, "mail-uma")
	assert.Contains(t, line, out.result.Message.ID)
	assert.NotContains(t, line, "thanks, on it", "the reply text stays out of the log")
}

func noReply(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	return &msgtypes.ExecutionResult{Status: "success"}, nil
}
