package umabridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	"hop.top/uma/adapter/messaging/meta"
)

func TestTwilioWhatsAppKeepsAPSAddressPrefixesAndRepliesWithThem(t *testing.T) {
	svc := smsService()
	svc.ID = "wa-twilio"
	svc.Options[core.OptionUmaAdapter] = "twilio.whatsapp"
	// aps ids keep the prefix, so the allowlist does too, as on the native
	// Twilio WhatsApp provider.
	svc.Options["allowed_numbers"] = "whatsapp:" + testSender
	h := newHarness(t, svc, replyWith("hello back"))

	form := incomingSMS("SM20000000000000000000000000000001", "hola")
	form.Set("From", "whatsapp:"+testSender)
	form.Set("To", "whatsapp:"+testNumber)
	assert.Equal(t, http.StatusOK, h.postSigned(t, form).StatusCode)

	waitFor(t, "twilio create", func() bool { return len(h.twilio.sent()) == 1 })
	msg := h.exec.calls()[0].Message
	assert.Equal(t, "whatsapp", msg.Platform)
	assert.Equal(t, "whatsapp:"+testSender, msg.Sender.ID, "aps keeps the whatsapp: prefix in ids")
	assert.Equal(t, "whatsapp:"+testNumber, msg.Channel.ID)
	assert.Equal(t, "hola", msg.Text)

	post := h.twilio.sent()[0]
	assert.Equal(t, "whatsapp:"+testSender, post.Form.Get("To"))
	assert.Equal(t, "whatsapp:"+testNumber, post.Form.Get("From"))
	assert.Equal(t, "hello back", post.Form.Get("Body"))
}

// fakeGraph is a Graph API stand-in: it records message creates.
type fakeGraph struct {
	srv *httptest.Server

	mu    sync.Mutex
	posts []graphPost
}

type graphPost struct {
	Path string
	Auth string
	Body map[string]any
}

func newFakeGraph(t *testing.T) *fakeGraph {
	t.Helper()
	g := &fakeGraph{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		g.mu.Lock()
		g.posts = append(g.posts, graphPost{Path: r.URL.Path, Auth: r.Header.Get("Authorization"), Body: body})
		n := len(g.posts)
		g.mu.Unlock()
		to, _ := body["to"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"messaging_product":"whatsapp","contacts":[{"input":%q,"wa_id":%q}],"messages":[{"id":"wamid.TEST%04d","message_status":"accepted"}]}`, to, to, n)
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGraph) sent() []graphPost {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]graphPost(nil), g.posts...)
}

const (
	metaPhoneNumberID = "106540352242922"
	metaAppSecret     = "app-secret"  // #nosec G101 -- fixture value
	metaVerifyToken   = "verify-me"   // #nosec G101 -- fixture value
	metaAccessToken   = "graph-token" // #nosec G101 -- fixture value
	metaSender        = "15551230000"
)

func metaService() *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: "wa-cloud", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvWhatsAppToken:  "secret:WA_TOKEN",
			core.EnvWhatsAppSecret: "secret:WA_SECRET",
			core.EnvWhatsAppVerify: "secret:WA_VERIFY",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "meta.whatsapp",
			"phone_number_id":     metaPhoneNumberID,
			"default_action":      "triage",
			"receive":             "webhook",
			"reply":               "text",
			"allowed_numbers":     metaSender,
		},
	}
}

func metaStore(name string) (string, bool) {
	v, ok := map[string]string{"WA_TOKEN": metaAccessToken, "WA_SECRET": metaAppSecret, "WA_VERIFY": metaVerifyToken}[name]
	return v, ok
}

func metaInbound(id, text string) []byte {
	return fmt.Appendf(nil, `{"object":"whatsapp_business_account","entry":[{"id":"102290129340398","changes":[{"value":{"messaging_product":"whatsapp","metadata":{"display_phone_number":"15550783881","phone_number_id":%q},"contacts":[{"profile":{"name":"Peer"},"wa_id":%q}],"messages":[{"from":%q,"id":%q,"timestamp":"%d","type":"text","text":{"body":%q}}]},"field":"messages"}]}]}`,
		metaPhoneNumberID, metaSender, metaSender, id, time.Now().Unix(), text)
}

func TestMetaWhatsAppVerifiesSubscriptionAndRepliesThroughGraph(t *testing.T) {
	graph := newFakeGraph(t)
	h := newHarnessWith(t, metaService(), replyWith("hi from aps"), func(c *Config) {
		c.Store = metaStore
		c.Options = Options{GraphBaseURL: graph.srv.URL}
	})

	// Meta's GET subscription verification: answered by the adapter, with the
	// verify token resolved from the secret binding.
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {metaVerifyToken}, "hub.challenge": {"1158201444"}}
	resp, err := http.Get(h.hook.URL + "?" + q.Encode())
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1158201444", strings.TrimSpace(string(body)))

	q.Set("hub.verify_token", "wrong")
	resp, err = http.Get(h.hook.URL + "?" + q.Encode())
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	raw := metaInbound("wamid.IN0001", "hello cloud")
	post := func(sig string) int {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.hook.URL, strings.NewReader(string(raw)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(meta.SignatureHeader, sig)
		r, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = r.Body.Close()
		return r.StatusCode
	}
	assert.Equal(t, http.StatusUnauthorized, post(meta.Sign([]byte("other-secret"), raw)))
	assert.Empty(t, h.exec.calls(), "an unsigned event never reaches the action")
	assert.Equal(t, http.StatusOK, post(meta.Sign([]byte(metaAppSecret), raw)))

	waitFor(t, "graph send", func() bool { return len(graph.sent()) == 1 })
	msg := h.exec.calls()[0].Message
	assert.Equal(t, "whatsapp", msg.Platform)
	assert.Equal(t, "wamid.IN0001", msg.ID)
	assert.Equal(t, metaSender, msg.Sender.ID)
	assert.Equal(t, metaPhoneNumberID, msg.Channel.ID)
	assert.Equal(t, "hello cloud", msg.Text)

	sent := graph.sent()[0]
	assert.Equal(t, "/v24.0/"+metaPhoneNumberID+"/messages", sent.Path)
	assert.Equal(t, "Bearer "+metaAccessToken, sent.Auth)
	assert.Equal(t, metaSender, sent.Body["to"])
	assert.Equal(t, "hi from aps", sent.Body["text"].(map[string]any)["body"])
}
