package cli

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- Twilio's documented webhook signature algorithm
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/adapters/messenger"
	"hop.top/aps/internal/adapters/umabridge"
	"hop.top/aps/internal/cli/service"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/protocol"
)

const (
	serveAccount = "AC00000000000000000000000000000001"
	serveToken   = "fake-auth-token" // #nosec G101 -- fixture value
	servePublic  = "https://hooks.example.com/twilio/sms"
	serveNumber  = "+15558675310"
	serveSender  = "+15551230000"

	serveAppSecret = "app-secret"  // #nosec G101 -- fixture value
	serveVerify    = "verify-me"   // #nosec G101 -- fixture value
	serveGraphTok  = "graph-token" // #nosec G101 -- fixture value
	servePhoneID   = "106540352242922"
	serveWAPeer    = "15551230000"
)

// providerFake records the REST calls uma makes to a provider origin.
type providerFake struct {
	srv *httptest.Server

	mu    sync.Mutex
	calls []map[string]string
}

func (f *providerFake) sent() []map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]string(nil), f.calls...)
}

// newTwilioFake answers Messages.json creates as Twilio does and records
// To, From and Body.
func newTwilioFake(t *testing.T) *providerFake {
	t.Helper()
	f := &providerFake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/Messages.json") {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		f.mu.Lock()
		f.calls = append(f.calls, map[string]string{"To": r.PostForm.Get("To"), "From": r.PostForm.Get("From"), "Body": r.PostForm.Get("Body")})
		n := len(f.calls)
		f.mu.Unlock()
		sid := fmt.Sprintf("SM%032x", n)
		uri := "/2010-04-01/Accounts/" + serveAccount + "/Messages/" + sid
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account_sid": serveAccount, "api_version": "2010-04-01", "body": r.PostForm.Get("Body"),
			"date_created": time.Now().UTC().Format(time.RFC1123Z), "date_updated": time.Now().UTC().Format(time.RFC1123Z),
			"date_sent": nil, "direction": "outbound-api", "error_code": nil, "error_message": nil,
			"from": r.PostForm.Get("From"), "messaging_service_sid": nil, "num_media": "0", "num_segments": "1",
			"price": nil, "sid": sid, "status": "queued", "to": r.PostForm.Get("To"), "uri": uri + ".json",
			"subresource_uris": map[string]string{"media": uri + "/Media.json"},
		})
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// newGraphFake answers message creates as the Graph API does and records To
// and the text body.
func newGraphFake(t *testing.T) *providerFake {
	t.Helper()
	f := &providerFake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			To   string `json:"to"`
			Text struct {
				Body string `json:"body"`
			} `json:"text"`
		}
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.calls = append(f.calls, map[string]string{"To": body.To, "Body": body.Text.Body, "Auth": r.Header.Get("Authorization")})
		n := len(f.calls)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"messaging_product":"whatsapp","contacts":[{"input":%q,"wa_id":%q}],"messages":[{"id":"wamid.TEST%04d","message_status":"accepted"}]}`, body.To, body.To, n)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// signTwilioForm computes X-Twilio-Signature independently of uma and aps.
func signTwilioForm(token, rawURL string, form url.Values) string {
	keys := make([]string, 0, len(form))
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := rawURL
	for _, k := range keys {
		s += k + form.Get(k)
	}
	m := hmac.New(sha1.New, []byte(token))
	m.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

func signMetaBody(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func smsForm(sid, body string) url.Values {
	return url.Values{
		"MessageSid": {sid}, "SmsSid": {sid}, "AccountSid": {serveAccount}, "SmsStatus": {"received"},
		"From": {serveSender}, "To": {serveNumber}, "Body": {body}, "NumMedia": {"0"}, "NumSegments": {"1"},
		"ApiVersion": {"2010-04-01"},
	}
}

// newUmaServer builds the real serve handler (kit router, middleware, the
// default adapter set but for a messenger adapter pointed at the provider
// fakes) behind a TCP listener, with an isolated data dir.
func newUmaServer(t *testing.T, token string) (*httptest.Server, *providerFake, *providerFake) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", "")
	t.Setenv("TWILIO_AUTH_TOKEN", serveToken)
	twilio, graph := newTwilioFake(t), newGraphFake(t)
	requireTestProfileAction(t, "assistant", "triage")

	mgr := adapters.NewManager()
	mgr.Add(messenger.NewAdapter(messenger.WithUmaOptions(umabridge.Options{
		TwilioBaseURL: twilio.srv.URL, GraphBaseURL: graph.srv.URL,
	})))
	require.NoError(t, mgr.InitAll(context.Background()))
	t.Cleanup(func() { assert.Empty(t, mgr.CloseAll()) })
	apsCore, err := protocol.NewAPSAdapter()
	require.NoError(t, err)
	handler, err := buildServerHandler(mgr, apsCore, token)
	require.NoError(t, err)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, twilio, graph
}

func saveUmaSMS(t *testing.T) {
	t.Helper()
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "sms-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvTwilioAccountSID: serveAccount,
			core.EnvTwilioAuthToken:  "secret:TWILIO_AUTH_TOKEN",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "twilio.sms", "from": serveNumber, "webhook_url": servePublic,
			"default_action": "triage", "receive": "webhook", "reply": "text", "allowed_numbers": serveSender,
		},
	}))
}

func saveUmaWhatsApp(t *testing.T) {
	t.Helper()
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "wa-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvWhatsAppToken: serveGraphTok, core.EnvWhatsAppSecret: serveAppSecret, core.EnvWhatsAppVerify: serveVerify,
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "meta.whatsapp", "phone_number_id": servePhoneID,
			"default_action": "triage", "receive": "webhook", "reply": "text", "allowed_numbers": serveWAPeer,
		},
	}))
}

func serveWait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 15*time.Second, 10*time.Millisecond, what)
}

func doServe(t *testing.T, method, rawURL string, body []byte, headers map[string]string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, rawURL, strings.NewReader(string(body)))
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out), resp.Header
}

// TestServeUma_TwilioSMS_KeepsThePublicWebhookURL drives a signed Twilio
// form through the real serve router: the service keeps the URL it had when
// aps served it natively, the signature is checked by uma over the
// configured public URL (not the host the request reached, forwarded or not),
// and the action's reply goes out through the Messages API.
func TestServeUma_TwilioSMS_KeepsThePublicWebhookURL(t *testing.T) {
	srv, twilio, _ := newUmaServer(t, "")
	saveUmaSMS(t)
	path := core.ServiceWebhookPath(&core.ServiceConfig{ID: "sms-uma", Type: "message"})
	require.Equal(t, "/services/sms-uma/webhook", path)

	form := smsForm("SM10000000000000000000000000000001", "where is my order")
	status, body, header := doServe(t, http.MethodPost, srv.URL+path, []byte(form.Encode()), map[string]string{
		"Content-Type":       "application/x-www-form-urlencoded",
		"X-Twilio-Signature": signTwilioForm(serveToken, servePublic, form),
		"X-Forwarded-Proto":  "http",
		"X-Forwarded-Host":   "internal.example.net",
	})
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, header.Get("Content-Type"), "xml")
	assert.Contains(t, body, "<Response>")
	assert.NotEmpty(t, header.Get("X-Request-Id"), "the serve middleware still wraps the route")

	serveWait(t, "reply through the Messages API", func() bool { return len(twilio.sent()) == 1 })
	assert.Equal(t, map[string]string{"To": serveSender, "From": serveNumber, "Body": "routed"}, twilio.sent()[0])

	serveWait(t, "service events", func() bool {
		svc, err := core.LoadService("sms-uma")
		return err == nil && svc.LastInbound != nil && svc.LastOutbound != nil
	})
}

func TestServeUma_TwilioSMS_BadSignatureIsRefusedAndNothingRuns(t *testing.T) {
	srv, twilio, _ := newUmaServer(t, "")
	saveUmaSMS(t)
	form := smsForm("SM10000000000000000000000000000002", "hi")
	post := func(sig string) int {
		status, _, _ := doServe(t, http.MethodPost, srv.URL+"/services/sms-uma/webhook", []byte(form.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded", "X-Twilio-Signature": sig,
		})
		return status
	}
	assert.Equal(t, http.StatusForbidden, post(""))
	assert.Equal(t, http.StatusForbidden, post(signTwilioForm("wrong-token", servePublic, form)))
	assert.Equal(t, http.StatusForbidden, post(signTwilioForm(serveToken, "https://evil.example.com/twilio/sms", form)))

	// A signed message afterwards is the only one answered.
	good := smsForm("SM10000000000000000000000000000003", "later")
	status, _, _ := doServe(t, http.MethodPost, srv.URL+"/services/sms-uma/webhook", []byte(good.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded", "X-Twilio-Signature": signTwilioForm(serveToken, servePublic, good),
	})
	require.Equal(t, http.StatusOK, status)
	serveWait(t, "one reply", func() bool { return len(twilio.sent()) == 1 })
	assert.Equal(t, "routed", twilio.sent()[0]["Body"])
}

func TestServeUma_TwilioSMS_OversizedBodyIsRefusedBeforeVerification(t *testing.T) {
	srv, twilio, _ := newUmaServer(t, "")
	saveUmaSMS(t)
	form := smsForm("SM10000000000000000000000000000004", strings.Repeat("a", 2<<20))

	status, _, _ := doServe(t, http.MethodPost, srv.URL+"/services/sms-uma/webhook", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded", "X-Twilio-Signature": signTwilioForm(serveToken, servePublic, form),
	})

	assert.Equal(t, http.StatusRequestEntityTooLarge, status)
	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, twilio.sent())
}

// TestServeUma_WhatsAppCloud_VerifiesAndSignsOverTheRawBody covers what the
// native route never could: Meta's GET subscription verification at the
// service URL, and an HMAC over the exact bytes Meta sent (whitespace and
// non-ASCII included, which a re-encoded body would not reproduce).
func TestServeUma_WhatsAppCloud_VerifiesAndSignsOverTheRawBody(t *testing.T) {
	srv, _, graph := newUmaServer(t, "")
	saveUmaWhatsApp(t)
	hook := srv.URL + "/services/wa-uma/webhook"

	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {serveVerify}, "hub.challenge": {"1158201444"}}
	status, body, _ := doServe(t, http.MethodGet, hook+"?"+q.Encode(), nil, nil)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Equal(t, "1158201444", strings.TrimSpace(body))
	q.Set("hub.verify_token", "wrong")
	status, body, _ = doServe(t, http.MethodGet, hook+"?"+q.Encode(), nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
	assert.NotContains(t, body, serveVerify)

	raw := []byte(fmt.Sprintf("{ \"object\" : \"whatsapp_business_account\",\n\t\"entry\":[{\"id\":\"102290129340398\",\"changes\":[{\"value\":{\"messaging_product\":\"whatsapp\",\"metadata\":{\"display_phone_number\":\"15550783881\",\"phone_number_id\":%q},\"contacts\":[{\"profile\":{\"name\":\"Zoë\"},\"wa_id\":%q}],\"messages\":[{\"from\":%q,\"id\":\"wamid.IN0001\",\"timestamp\":\"%d\",\"type\":\"text\",\"text\":{\"body\":\"café ☕\"}}]},\"field\":\"messages\"}]}]}  \n",
		servePhoneID, serveWAPeer, serveWAPeer, time.Now().Unix()))
	post := func(sig string, payload []byte) int {
		s, _, _ := doServe(t, http.MethodPost, hook, payload, map[string]string{"Content-Type": "application/json", "X-Hub-Signature-256": sig})
		return s
	}
	tampered := append([]byte(nil), raw...)
	tampered[3] = '\t'
	assert.Equal(t, http.StatusUnauthorized, post(signMetaBody(serveAppSecret, raw), tampered), "one changed byte breaks the signature")
	assert.Equal(t, http.StatusUnauthorized, post(signMetaBody("other-secret", raw), raw))
	assert.Empty(t, graph.sent())

	require.Equal(t, http.StatusOK, post(signMetaBody(serveAppSecret, raw), raw))
	serveWait(t, "reply through the Graph API", func() bool { return len(graph.sent()) == 1 })
	sent := graph.sent()[0]
	assert.Equal(t, serveWAPeer, sent["To"])
	assert.Equal(t, "routed", sent["Body"])
	assert.Equal(t, "Bearer "+serveGraphTok, sent["Auth"])
}

// TestServeUma_NativeServicesKeepTheirRoutes pins that mounting uma changed
// nothing for the native providers at the same URL.
func TestServeUma_NativeServicesKeepTheirRoutes(t *testing.T) {
	srv, _, _ := newUmaServer(t, "")
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "tg", Type: "message", Adapter: "telegram", Profile: "assistant",
		Options: map[string]string{"default_action": "triage", "reply": "none"},
	}))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "wa-native", Type: "message", Adapter: "whatsapp", Profile: "assistant",
		Options: map[string]string{"default_action": "triage", "reply": "none"},
	}))

	for _, id := range []string{"tg", "wa-native"} {
		status, body, header := doServe(t, http.MethodGet, srv.URL+"/services/"+id+"/webhook?hub.mode=subscribe", nil, nil)
		assert.Equal(t, http.StatusMethodNotAllowed, status, id)
		assert.Equal(t, "POST", header.Get("Allow"), id)
		assert.Equal(t, "Method Not Allowed\n", body, id)
	}
	status, _, _ := doServe(t, http.MethodPost, srv.URL+"/services/tg/webhook", []byte(`{"message":{"message_id":1,"from":{"id":456,"first_name":"Alice"},"chat":{"id":-1001234567890,"type":"group"},"text":"hello"}}`),
		map[string]string{"Content-Type": "application/json"})
	assert.Equal(t, http.StatusOK, status)
}

// TestServeUma_AuthTokenGatesTheWebhookLikeEveryAdapterRoute documents that
// --auth-token wraps the whole adapter mux, uma services included: a provider
// cannot send a bearer token, so a gated server cannot take provider webhooks.
func TestServeUma_AuthTokenGatesTheWebhookLikeEveryAdapterRoute(t *testing.T) {
	srv, twilio, _ := newUmaServer(t, "operator-token")
	saveUmaSMS(t)
	form := smsForm("SM10000000000000000000000000000005", "hi")
	status, _, _ := doServe(t, http.MethodPost, srv.URL+"/services/sms-uma/webhook", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded", "X-Twilio-Signature": signTwilioForm(serveToken, servePublic, form),
	})
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, twilio.sent())
}

// runConversationQuery runs `aps service conversation <args>` in this
// process, as the CLI runs separately from the server: it opens the history
// file by path and shares nothing else with it.
func runConversationQuery(t *testing.T, args ...string) []map[string]any {
	t.Helper()
	root := &cobra.Command{Use: "aps"}
	root.PersistentFlags().String("profile", "", "profile id")
	root.PersistentFlags().String("workspace", "", "workspace id")
	root.PersistentFlags().Bool("dry-run", false, "preview side effects")
	root.PersistentFlags().String("format", "", "output format")
	root.AddCommand(service.NewServiceCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"service", "conversation"}, append(args, "--format", "json")...))
	require.NoError(t, root.Execute(), out.String())
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &rows), out.String())
	return rows
}

// A uma service's exchange is in the same history file as a native one's,
// so `aps service conversation` reads it with no engine to reach.
func TestServeUma_TwilioSMS_ConversationQueryShowsTheExchange(t *testing.T) {
	srv, twilio, _ := newUmaServer(t, "")
	saveUmaSMS(t)
	form := smsForm("SM10000000000000000000000000000003", "where is my order")
	status, body, _ := doServe(t, http.MethodPost, srv.URL+"/services/sms-uma/webhook", []byte(form.Encode()), map[string]string{
		"Content-Type":       "application/x-www-form-urlencoded",
		"X-Twilio-Signature": signTwilioForm(serveToken, servePublic, form),
	})
	require.Equal(t, http.StatusOK, status, body)
	serveWait(t, "reply", func() bool { return len(twilio.sent()) == 1 })

	var conversations []map[string]any
	serveWait(t, "both turns listed", func() bool {
		conversations = runConversationQuery(t, "list", "--service", "sms-uma")
		return len(conversations) == 1 && conversations[0]["turn_count"] == float64(2)
	})
	assert.Equal(t, "sms", conversations[0]["platform"])
	assert.Equal(t, serveNumber, conversations[0]["channel_id"])
	assert.Equal(t, "outbound", conversations[0]["last_direction"])
	assert.Equal(t, "routed", conversations[0]["last_text"])

	turns := runConversationQuery(t, "show", conversations[0]["conversation_id"].(string))
	require.Len(t, turns, 2)
	assert.Equal(t, "inbound", turns[0]["direction"])
	assert.Equal(t, "where is my order", turns[0]["text"])
	assert.Equal(t, serveSender, turns[0]["sender_id"])
	assert.Equal(t, "SM10000000000000000000000000000003", turns[0]["message_id"])
	assert.Equal(t, "outbound", turns[1]["direction"])
	assert.Equal(t, "routed", turns[1]["text"])
	assert.Equal(t, "triage", turns[1]["action_name"])
	assert.Equal(t, "sms-uma", turns[1]["service_id"])
}
