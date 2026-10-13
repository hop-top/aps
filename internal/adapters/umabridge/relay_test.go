package umabridge

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
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
	msgtypes "hop.top/aps/internal/core/messenger"
)

const (
	relaySecret   = "relay-signing-secret-0123456789" // #nosec G101 -- fixture value
	callbackToken = "callback-token-abcdef"           // #nosec G101 -- fixture value
)

// secrets is a profile secret store holding fixed values.
func secrets(values map[string]string) core.SecretLookup {
	return func(name string) (string, bool) {
		v, ok := values[name]
		return v, ok
	}
}

// useHostSettings writes the global aps configuration with the uma relay
// callback hosts, http allowed for the test servers, under a throwaway
// config home.
func useHostSettings(t *testing.T, hosts []string) {
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

// gateway is the operator's relay: its callback endpoint records each reply.
type gateway struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []callbackCall
	// answer, when set, answers instead of the default 200.
	answer func(w http.ResponseWriter)
}

type callbackCall struct {
	Header http.Header
	Body   map[string]any
}

func newGateway(t *testing.T) *gateway {
	t.Helper()
	g := &gateway{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		g.mu.Lock()
		g.got = append(g.got, callbackCall{Header: r.Header.Clone(), Body: body})
		answer := g.answer
		g.mu.Unlock()
		if answer != nil {
			answer(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"message_id":"gw-out-1"}`)
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *gateway) calls() []callbackCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]callbackCall(nil), g.got...)
}

func (g *gateway) host() string {
	u, _ := url.Parse(g.srv.URL)
	return u.Host
}

// relayService is a generic SMS relay service moved to uma with aps's
// legacy HMAC layout: X-APS-Signature over the body, an unsigned
// X-APS-Timestamp and X-APS-Delivery-ID accepted by consent.
func relayService(callbackURL string) *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: "relay-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			"RELAY_SIGNING_SECRET":     "secret:RELAY_SIGNING_SECRET",
			core.EnvRelayCallbackToken: "secret:RELAY_CALLBACK_TOKEN",
		},
		Options: map[string]string{
			core.OptionUmaAdapter:         core.UmaAdapterRelay,
			core.OptionMedium:             "sms",
			"from":                        testNumber,
			core.OptionCallbackURL:        callbackURL,
			core.OptionAuthScheme:         core.AuthSchemeHMACSHA256,
			core.OptionSignatureSecretEnv: "RELAY_SIGNING_SECRET",
			core.OptionRequireTimestamp:   "true",
			core.OptionRequireReplayCheck: "true",
			core.OptionAuthAllowUnsigned:  "true",
			"default_action":              "triage",
			"receive":                     "webhook",
			"reply":                       "text",
			"allowed_numbers":             testSender,
		},
	}
}

var relaySecrets = secrets(map[string]string{"RELAY_SIGNING_SECRET": relaySecret, "RELAY_CALLBACK_TOKEN": callbackToken})

// bridgeHarness runs a bridge for any adapter and serves its webhook.
type bridgeHarness struct {
	hook *httptest.Server
	exec *stubExecutor
	mu   sync.Mutex
	res  []resultRecord
}

func (h *bridgeHarness) outcomes() []resultRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]resultRecord(nil), h.res...)
}

func newBridgeHarness(t *testing.T, svc *core.ServiceConfig, store core.SecretLookup,
	fn func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error), mutate func(*Config),
) *bridgeHarness {
	t.Helper()
	b, h, err := startBridge(t, svc, store, fn, mutate)
	require.NoError(t, err)
	require.NotNil(t, b)
	return h
}

func startBridge(t *testing.T, svc *core.ServiceConfig, store core.SecretLookup,
	fn func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error), mutate func(*Config),
) (*Bridge, *bridgeHarness, error) {
	t.Helper()
	h := &bridgeHarness{exec: &stubExecutor{fn: fn}}
	cfg := Config{
		Service: svc, Router: stubRouter{}, Executor: h.exec, Store: store, Poll: 10 * time.Millisecond,
		OnResult: func(r *msgtypes.RuntimeResult, err error) {
			h.mu.Lock()
			h.res = append(h.res, resultRecord{r, err})
			h.mu.Unlock()
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	b, err := New(t.Context(), cfg)
	if err != nil {
		return nil, nil, err
	}
	h.hook = httptest.NewServer(b.Webhook())
	t.Cleanup(h.hook.Close)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			assert.NoError(t, err, "bridge run")
		case <-time.After(15 * time.Second):
			t.Error("bridge did not stop")
		}
		assert.NoError(t, b.Close(context.Background()))
	})
	return b, h, nil
}

func signHex(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// relayRequest is one webhook as aps's native generic relay accepted it.
type relayRequest struct {
	body      []byte
	signature string // empty: unsigned
	timestamp string
	delivery  string
}

func legacyRelay(secret, delivery string, payload map[string]any) relayRequest {
	body, _ := json.Marshal(payload)
	return relayRequest{
		body: body, signature: "sha256=" + signHex(secret, body),
		timestamp: strconv.FormatInt(time.Now().Unix(), 10), delivery: delivery,
	}
}

// postRelay posts req as the gateway would and returns the status and answer.
func (h *bridgeHarness) postRelay(t *testing.T, req relayRequest) (int, string) {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.hook.URL, bytes.NewReader(req.body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")
	if req.signature != "" {
		r.Header.Set("X-APS-Signature", req.signature)
	}
	if req.timestamp != "" {
		r.Header.Set("X-APS-Timestamp", req.timestamp)
	}
	if req.delivery != "" {
		r.Header.Set("X-APS-Delivery-ID", req.delivery)
	}
	resp, err := http.DefaultClient.Do(r)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	answer, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(answer)
}

func (h *bridgeHarness) relayStatus(t *testing.T, req relayRequest) int {
	t.Helper()
	status, _ := h.postRelay(t, req)
	return status
}

// twilioStyle is the payload of a gateway that posts Twilio's field names.
func twilioStyle(sid, body string) map[string]any {
	return map[string]any{"From": testSender, "To": testNumber, "Body": body, "MessageSid": sid}
}

func TestRelay_LegacySignedWebhookRunsActionAndRepliesByCallback(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	h := newBridgeHarness(t, relayService(gw.srv.URL+"/uma/reply"), relaySecrets, replyWith("on it"), nil)

	status, answer := h.postRelay(t, legacyRelay(relaySecret, "d-1", twilioStyle("SM-1", "where is my order")))
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"status":"accepted"}`, answer, "the answer carries no reply")

	waitFor(t, "callback", func() bool { return len(gw.calls()) == 1 })
	calls := h.exec.calls()
	require.Len(t, calls, 1)
	msg := calls[0].Message
	assert.Equal(t, "sms", calls[0].Provider)
	assert.Equal(t, "SM-1", msg.ID)
	assert.Equal(t, "sms", msg.Platform)
	assert.Equal(t, testSender, msg.Sender.ID)
	assert.Equal(t, testNumber, msg.Channel.ID)
	assert.Equal(t, "where is my order", msg.Text)
	assert.Equal(t, "relay.messages", msg.PlatformMetadata["uma_adapter"])

	cb := gw.calls()[0]
	assert.Equal(t, "Bearer "+callbackToken, cb.Header.Get("Authorization"), "callback credential from its binding")
	assert.NotEmpty(t, cb.Header.Get("Idempotency-Key"))
	assert.Equal(t, cb.Body["operation_id"], cb.Header.Get("Idempotency-Key"))
	assert.Equal(t, "sms", cb.Body["medium"])
	assert.Equal(t, testSender, cb.Body["to"])
	assert.Equal(t, "on it", cb.Body["body"])
	assert.Equal(t, "SM-1", cb.Body["in_reply_to"])

	waitFor(t, "result", func() bool { return len(h.outcomes()) == 1 })
	out := h.outcomes()[0]
	require.NoError(t, out.err)
	require.NotNil(t, out.result.Delivery)
	assert.Equal(t, "accepted", out.result.Delivery.Status)
}

func TestRelay_UnsignedForgedAndStaleRequestsNeverReachTheAction(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	h := newBridgeHarness(t, relayService(gw.srv.URL+"/r"), relaySecrets, replyWith("on it"), nil)

	good := legacyRelay(relaySecret, "d-2", twilioStyle("SM-2", "hi"))
	unsigned := good
	unsigned.signature = ""
	forged := legacyRelay("another-secret-of-enough-length", "d-3", twilioStyle("SM-3", "hi"))
	stale := good
	stale.delivery = "d-4"
	stale.timestamp = strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	noStamp := good
	noStamp.delivery, noStamp.timestamp = "d-5", ""
	tampered := good
	tampered.delivery = "d-6"
	tampered.body = []byte(strings.Replace(string(good.body), "hi", "ho", 1))

	for name, req := range map[string]relayRequest{
		"unsigned": unsigned, "forged": forged, "stale": stale, "no timestamp": noStamp, "tampered": tampered,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, http.StatusUnauthorized, h.relayStatus(t, req))
		})
	}
	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, h.exec.calls())
	assert.Empty(t, gw.calls())
}

func TestRelay_ReplayedDeliveryIsRefused(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	h := newBridgeHarness(t, relayService(gw.srv.URL+"/r"), relaySecrets, replyWith("on it"), nil)

	req := legacyRelay(relaySecret, "d-replay", twilioStyle("SM-7", "hi"))
	require.Equal(t, http.StatusOK, h.relayStatus(t, req))
	assert.Equal(t, http.StatusConflict, h.relayStatus(t, req), "same delivery id")

	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	time.Sleep(200 * time.Millisecond)
	assert.Len(t, h.exec.calls(), 1)
}

func TestRelay_CallbackHostAllowlistReachesUma(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	svc := relayService(gw.srv.URL + "/r")

	_, _, err := startBridge(t, svc, relaySecrets, replyWith("x"), func(c *Config) {
		c.Host = &core.UmaHostConfig{Relay: core.UmaRelayHostConfig{AllowHTTP: true}}
	})
	require.Error(t, err, "uma refuses a callback host its host settings do not list")
	assert.Contains(t, err.Error(), "callback.url")

	useHostSettings(t, []string{"gateway.example.com"})
	_, _, err = startBridge(t, svc, relaySecrets, replyWith("x"), nil)
	require.Error(t, err, "aps refuses it first, from the global setting")
	assert.Contains(t, err.Error(), "uma.relay.callback_hosts")
}

func TestRelay_CallbackRedirectIsNotFollowed(t *testing.T) {
	elsewhere := newGateway(t)
	gw := newGateway(t)
	gw.answer = func(w http.ResponseWriter) {
		w.Header().Set("Location", elsewhere.srv.URL+"/steal")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
	useHostSettings(t, []string{gw.host(), elsewhere.host()})
	h := newBridgeHarness(t, relayService(gw.srv.URL+"/r"), relaySecrets, replyWith("on it"), nil)

	require.Equal(t, http.StatusOK, h.relayStatus(t, legacyRelay(relaySecret, "d-8", twilioStyle("SM-8", "hi"))))
	waitFor(t, "callback", func() bool { return len(gw.calls()) == 1 })
	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, elsewhere.calls(), "the callback credential never follows a redirect")
}

func TestRelay_SecretComesFromTheBindingOnly(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	svc := relayService(gw.srv.URL + "/r")
	const literal = "a-literal-secret-in-the-options"
	svc.Options[core.OptionSignatureSecret] = literal
	h := newBridgeHarness(t, svc, relaySecrets, replyWith("on it"), nil)

	assert.Equal(t, http.StatusUnauthorized,
		h.relayStatus(t, legacyRelay(literal, "d-9", twilioStyle("SM-9", "hi"))),
		"a secret written into the options is never read")
	assert.Equal(t, http.StatusOK,
		h.relayStatus(t, legacyRelay(relaySecret, "d-10", twilioStyle("SM-10", "hi"))))
}

func TestRelay_ShortSecretRefusesToStartUnlessLowered(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	short := secrets(map[string]string{"RELAY_SIGNING_SECRET": "short-secret", "RELAY_CALLBACK_TOKEN": callbackToken})

	_, _, err := startBridge(t, relayService(gw.srv.URL+"/r"), short, replyWith("x"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_min_secret_bytes=12")
	assert.NotContains(t, err.Error(), "short-secret", "the secret is never part of the error")

	svc := relayService(gw.srv.URL + "/r")
	svc.Options[core.OptionAuthMinSecretBytes] = "12"
	h := newBridgeHarness(t, svc, short, replyWith("on it"), nil)
	assert.Equal(t, http.StatusOK,
		h.relayStatus(t, legacyRelay("short-secret", "d-11", twilioStyle("SM-11", "hi"))))
}

func TestRelay_MediaPassesThroughAndAMissingIDIsGenerated(t *testing.T) {
	gw := newGateway(t)
	useHostSettings(t, []string{gw.host()})
	h := newBridgeHarness(t, relayService(gw.srv.URL+"/r"), relaySecrets, replyWith("on it"), nil)

	payload := map[string]any{
		"from": testSender, "to": testNumber, "text": "look", "WaId": "15551230000",
		"media": []any{"https://cdn.example.com/a.jpg", map[string]any{"url": "https://cdn.example.com/b", "content_type": "image/png"}},
	}
	require.Equal(t, http.StatusOK, h.relayStatus(t, legacyRelay(relaySecret, "d-12", payload)))
	waitFor(t, "action", func() bool { return len(h.exec.calls()) == 1 })
	msg := h.exec.calls()[0].Message
	assert.Equal(t, "look", msg.Text)
	assert.NotEqual(t, "15551230000", msg.ID, "WaId is the sender, not a message id")
	assert.True(t, strings.HasPrefix(msg.ID, "g"), "generated id, got %q", msg.ID)
	require.Len(t, msg.Attachments, 2)
	assert.Equal(t, "https://cdn.example.com/a.jpg", msg.Attachments[0].URL)
	assert.Equal(t, "image", msg.Attachments[1].Type)
	assert.Equal(t, "image/png", msg.Attachments[1].MimeType)
}
