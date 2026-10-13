package cli

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

const (
	serveRelaySecret = "relay-signing-secret-0123456789" // #nosec G101 -- fixture value
	serveCallbackTok = "callback-token-abcdef"           // #nosec G101 -- fixture value
	serveMailToken   = "mail-relay-token-0123456789"     // #nosec G101 -- fixture value
)

// relayGateway is the operator's gateway: it serves the callback uma posts
// replies to, and records each one.
type relayGateway struct {
	srv  *httptest.Server
	mu   sync.Mutex
	hits []relayHit
}

type relayHit struct {
	header http.Header
	body   map[string]any
}

func newRelayGateway(t *testing.T) *relayGateway {
	t.Helper()
	g := &relayGateway{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		g.mu.Lock()
		g.hits = append(g.hits, relayHit{header: r.Header.Clone(), body: body})
		g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"message_id":"gw-1"}`)
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *relayGateway) received() []relayHit {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]relayHit(nil), g.hits...)
}

// allowCallbackHosts writes the global aps configuration: the uma relay may
// post to hosts, over http for the test servers.
func allowCallbackHosts(t *testing.T, hosts ...string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	body := "uma:\n  relay:\n    allow_http: true\n    callback_hosts:\n"
	for _, h := range hosts {
		body += "      - \"" + h + "\"\n"
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "aps"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "aps", "config.yaml"), []byte(body), 0o600))
}

func hostOf(rawURL string) string {
	u, _ := url.Parse(rawURL)
	return u.Host
}

// saveUmaRelay saves a generic SMS relay moved to uma with aps's legacy HMAC
// layout: the sender keeps X-APS-Signature, X-APS-Timestamp and
// X-APS-Delivery-ID.
func saveUmaRelay(t *testing.T, callbackURL string) {
	t.Helper()
	t.Setenv("RELAY_SIGNING_SECRET", serveRelaySecret)
	t.Setenv("RELAY_CALLBACK_TOKEN", serveCallbackTok)
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "relay-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			"RELAY_SIGNING_SECRET":     "secret:RELAY_SIGNING_SECRET",
			core.EnvRelayCallbackToken: "secret:RELAY_CALLBACK_TOKEN",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: core.UmaAdapterRelay, core.OptionMedium: "sms", "from": serveNumber,
			core.OptionCallbackURL: callbackURL, core.OptionAuthScheme: core.AuthSchemeHMACSHA256,
			core.OptionSignatureSecretEnv: "RELAY_SIGNING_SECRET", core.OptionRequireTimestamp: "true",
			core.OptionRequireReplayCheck: "true", core.OptionAuthAllowUnsigned: "true",
			"default_action": "triage", "receive": "webhook", "reply": "text", "allowed_numbers": serveSender,
		},
	}))
}

func relayHeaders(secret, delivery string, body []byte) map[string]string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return map[string]string{
		"Content-Type":      "application/json",
		"X-APS-Signature":   "sha256=" + hex.EncodeToString(m.Sum(nil)),
		"X-APS-Timestamp":   strconv.FormatInt(time.Now().Unix(), 10),
		"X-APS-Delivery-ID": delivery,
	}
}

// TestServeUma_Relay_LegacySenderIsAnsweredByCallback drives a generic relay
// through the real serve router with the payload and headers aps's native
// generic relay took: uma verifies the legacy HMAC, the action runs, and the
// reply goes to the gateway's callback with an Idempotency-Key.
func TestServeUma_Relay_LegacySenderIsAnsweredByCallback(t *testing.T) {
	gw := newRelayGateway(t)
	srv, _, _ := newUmaServer(t, "")
	allowCallbackHosts(t, hostOf(gw.srv.URL))
	saveUmaRelay(t, gw.srv.URL+"/uma/reply")
	hook := srv.URL + "/services/relay-uma/webhook"

	body := []byte(`{"From":"` + serveSender + `","To":"` + serveNumber + `","Body":"where is my order","MessageSid":"SM-R1"}`)
	status, answer, header := doServe(t, http.MethodPost, hook, body, relayHeaders(serveRelaySecret, "d-1", body))
	require.Equal(t, http.StatusOK, status, answer)
	assert.JSONEq(t, `{"status":"accepted"}`, answer)
	assert.NotEmpty(t, header.Get("X-Request-Id"), "the serve middleware still wraps the route")

	serveWait(t, "callback", func() bool { return len(gw.received()) == 1 })
	hit := gw.received()[0]
	assert.Equal(t, "Bearer "+serveCallbackTok, hit.header.Get("Authorization"))
	assert.NotEmpty(t, hit.header.Get("Idempotency-Key"))
	assert.Equal(t, map[string]any{
		"medium": "sms", "to": serveSender, "body": "routed", "in_reply_to": "SM-R1",
		"operation_id": hit.header.Get("Idempotency-Key"), "attempt_id": hit.body["attempt_id"],
	}, hit.body)

	serveWait(t, "service events", func() bool {
		svc, err := core.LoadService("relay-uma")
		return err == nil && svc.LastInbound != nil && svc.LastOutbound != nil
	})
	svc, err := core.LoadService("relay-uma")
	require.NoError(t, err)
	assert.Equal(t, "SM-R1", svc.LastInbound.MessageID)
	assert.Equal(t, "accepted", svc.LastOutbound.Status)

	// Refused before anything runs: unsigned, forged, replayed.
	unsigned := relayHeaders(serveRelaySecret, "d-2", body)
	delete(unsigned, "X-APS-Signature")
	status, _, _ = doServe(t, http.MethodPost, hook, body, unsigned)
	assert.Equal(t, http.StatusUnauthorized, status, "unsigned")
	status, _, _ = doServe(t, http.MethodPost, hook, body, relayHeaders("another-secret-0123456789", "d-3", body))
	assert.Equal(t, http.StatusUnauthorized, status, "forged")
	status, _, _ = doServe(t, http.MethodPost, hook, body, relayHeaders(serveRelaySecret, "d-1", body))
	assert.Equal(t, http.StatusConflict, status, "replayed delivery id")
	time.Sleep(200 * time.Millisecond)
	assert.Len(t, gw.received(), 1, "nothing refused reached the action")
}

func TestServeUma_Relay_CallbackHostOutsideTheAllowlistIsNotServed(t *testing.T) {
	gw := newRelayGateway(t)
	srv, _, _ := newUmaServer(t, "")
	allowCallbackHosts(t, "gateway.example.com")
	saveUmaRelay(t, gw.srv.URL+"/uma/reply")

	body := []byte(`{"from":"` + serveSender + `","to":"` + serveNumber + `","body":"hi","message_id":"m-1"}`)
	status, answer, _ := doServe(t, http.MethodPost, srv.URL+"/services/relay-uma/webhook", body,
		relayHeaders(serveRelaySecret, "d-1", body))
	assert.Equal(t, http.StatusServiceUnavailable, status, answer)
	assert.NotContains(t, answer, gw.srv.URL, "the cause stays in the log")
	assert.Empty(t, gw.received())
}

// saveUmaMail saves the email relay moved to http.mail. Its reply mode is
// left unset, so the action's reply is refused.
func saveUmaMail(t *testing.T) {
	t.Helper()
	t.Setenv("MAIL_RELAY_TOKEN", serveMailToken)
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "mail-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{"MAIL_RELAY_TOKEN": "secret:MAIL_RELAY_TOKEN"},
		Options: map[string]string{
			core.OptionUmaAdapter: core.UmaAdapterHTTPMail, core.OptionRecipients: "support@example.com",
			core.OptionFormats: "json,rfc822", core.OptionAuthScheme: core.AuthSchemeBearer,
			core.OptionAuthTokenEnv: "MAIL_RELAY_TOKEN", core.OptionAllowedSenders: "*@example.org",
			"default_action": "triage", "receive": "webhook",
		},
	}))
}

// TestServeUma_HTTPMail_ReceivesStagesAndRefusesTheReply posts the native
// email relay's JSON and a raw message through the real serve router: both
// run the action with the attachment bytes staged under the data dir and
// the sender unverified, and the action's reply is refused, never sent.
func TestServeUma_HTTPMail_ReceivesStagesAndRefusesTheReply(t *testing.T) {
	srv, _, _ := newUmaServer(t, "")
	saveUmaMail(t)
	stdin := filepath.Join(t.TempDir(), "stdin.json")
	profileDir, err := core.GetProfileDir("assistant")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "actions", "triage.sh"),
		[]byte("cat > '"+stdin+"'\nprintf routed"), 0o755)) // #nosec G306 -- an action script
	hook := srv.URL + "/services/mail-uma/webhook"
	bearer := map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + serveMailToken}

	payload, _ := json.Marshal(map[string]any{
		"from": "Alice <alice@example.org>", "to": "support@example.com", "subject": "Order 42", "body": "where is it?",
		"attachments": []any{map[string]any{
			"type": "file", "name": "note.txt", "mime_type": "text/plain",
			"content": base64.StdEncoding.EncodeToString([]byte("hello attachment")),
		}},
	})
	status, _, _ := doServe(t, http.MethodPost, hook, payload, map[string]string{"Content-Type": "application/json"})
	assert.Equal(t, http.StatusUnauthorized, status, "no credential")

	status, answer, _ := doServe(t, http.MethodPost, hook, payload, bearer)
	require.Equal(t, http.StatusOK, status, answer)
	assert.JSONEq(t, `{"accepted":1}`, answer, "no reply rides the answer")

	serveWait(t, "refused reply recorded", func() bool {
		svc, err := core.LoadService("mail-uma")
		return err == nil && svc.LastOutbound != nil
	})
	svc, err := core.LoadService("mail-uma")
	require.NoError(t, err)
	assert.Equal(t, "reply_disabled", svc.LastOutbound.Status)
	assert.Contains(t, svc.LastOutbound.Detail, "receives only")
	assert.Equal(t, "alice@example.org", svc.LastInbound.SenderID)

	raw, err := os.ReadFile(stdin) // #nosec G304 -- test temp file
	require.NoError(t, err)
	var input struct {
		Platform         string         `json:"platform"`
		PlatformMetadata map[string]any `json:"platform_metadata"`
		Attachments      []struct {
			URL string `json:"url"`
		} `json:"attachments"`
	}
	require.NoError(t, json.Unmarshal(raw, &input), string(raw))
	assert.Equal(t, "email", input.Platform)
	assert.Equal(t, "unverified", input.PlatformMetadata["sender_assurance"])
	require.Len(t, input.Attachments, 1, string(raw))
	staged := strings.TrimPrefix(input.Attachments[0].URL, "file://")
	dataDir, err := core.GetDataDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(staged, filepath.Join(dataDir, "messenger", "uma", "mail-uma")), staged)
	content, err := os.ReadFile(staged) // #nosec G304 -- path under the test data dir
	require.NoError(t, err)
	assert.Equal(t, "hello attachment", string(content))

	rows := runConversationQuery(t, "list", "--service", "mail-uma")
	require.Len(t, rows, 1)
	assert.Equal(t, "inbound", rows[0]["last_direction"], "no outbound turn for a refused reply")

	rawMail := "From: bob@example.org\r\nTo: support@example.com\r\nSubject: Raw\r\n\r\nraw body\r\n"
	status, _, _ = doServe(t, http.MethodPost, hook, []byte(rawMail),
		map[string]string{"Content-Type": "message/rfc822", "Authorization": "Bearer " + serveMailToken})
	require.Equal(t, http.StatusOK, status)
	serveWait(t, "second message", func() bool {
		svc, err := core.LoadService("mail-uma")
		return err == nil && svc.LastInbound != nil && svc.LastInbound.SenderID == "bob@example.org"
	})
}
