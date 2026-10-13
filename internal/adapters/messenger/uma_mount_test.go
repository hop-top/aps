package messenger

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- Twilio's documented webhook signature algorithm
	"encoding/base64"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/adapters/umabridge"
	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
)

const (
	umaAccount  = "AC00000000000000000000000000000001"
	umaToken    = "fake-auth-token" // #nosec G101 -- fixture value
	umaHookURL  = "https://hooks.example.com/twilio/sms"
	umaNumber   = "+15558675310"
	umaSender   = "+15551230000"
	umaStranger = "+15559998888"
)

// umaCore is the APSCore the messenger adapter runs actions on: it records
// each run and answers with a canned output. Only ExecuteRun is wired.
type umaCore struct {
	protocol.APSCore

	mu     sync.Mutex
	runs   []protocol.RunInput
	output string
}

func (c *umaCore) ExecuteRun(_ context.Context, in protocol.RunInput, _ protocol.StreamWriter) (*protocol.RunState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runs = append(c.runs, in)
	return &protocol.RunState{
		ProfileID: in.ProfileID, ActionID: in.ActionID,
		Status: protocol.RunStatusCompleted, Output: c.output,
	}, nil
}

func (c *umaCore) calls() []protocol.RunInput {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]protocol.RunInput(nil), c.runs...)
}

// umaTwilio stands in for the Twilio Messages API.
type umaTwilio struct {
	srv *httptest.Server

	mu    sync.Mutex
	posts []url.Values
}

func newUmaTwilio(t *testing.T) *umaTwilio {
	t.Helper()
	f := &umaTwilio{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/Messages.json") {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		f.mu.Lock()
		f.posts = append(f.posts, r.PostForm)
		n := len(f.posts)
		f.mu.Unlock()
		sid := fmt.Sprintf("SM%032x", n)
		uri := "/2010-04-01/Accounts/" + umaAccount + "/Messages/" + sid
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account_sid": umaAccount, "api_version": "2010-04-01", "body": r.PostForm.Get("Body"),
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

func (f *umaTwilio) sent() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.posts...)
}

// umaSignTwilio computes X-Twilio-Signature independently of the adapter.
func umaSignTwilio(token, rawURL string, form url.Values) string {
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

func umaIncoming(sid, from, body string) url.Values {
	return url.Values{
		"MessageSid": {sid}, "SmsSid": {sid}, "AccountSid": {umaAccount}, "SmsStatus": {"received"},
		"From": {from}, "To": {umaNumber}, "Body": {body}, "NumMedia": {"0"}, "NumSegments": {"1"},
		"ApiVersion": {"2010-04-01"},
	}
}

func umaSMSService(id string) *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: id, Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvTwilioAccountSID: umaAccount,
			core.EnvTwilioAuthToken:  "secret:TWILIO_AUTH_TOKEN",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "twilio.sms",
			"from":                umaNumber,
			"webhook_url":         umaHookURL,
			"default_action":      "triage",
			"receive":             "webhook",
			"reply":               "text",
			"allowed_numbers":     umaSender,
		},
	}
}

type umaRoutes struct {
	adapter *Adapter
	mux     *http.ServeMux
	core    *umaCore
	twilio  *umaTwilio
}

// newUmaRoutes registers the messenger adapter's routes on a fresh mux with
// an isolated data dir, the fake Twilio REST origin and a fake action core.
func newUmaRoutes(t *testing.T) *umaRoutes {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", "")
	t.Setenv("TWILIO_AUTH_TOKEN", umaToken)
	r := &umaRoutes{core: &umaCore{output: "on it"}, twilio: newUmaTwilio(t), mux: http.NewServeMux()}
	r.adapter = NewAdapter(WithUmaOptions(umabridge.Options{TwilioBaseURL: r.twilio.srv.URL}))
	require.NoError(t, r.adapter.Start(t.Context(), nil))
	require.NoError(t, r.adapter.RegisterRoutes(r.mux, r.core))
	t.Cleanup(func() { assert.NoError(t, r.adapter.Stop()) })
	return r
}

func (r *umaRoutes) do(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.mux.ServeHTTP(rec, req)
	return rec
}

// postForm sends a Twilio form webhook signed over signedURL.
func (r *umaRoutes) postForm(t *testing.T, id string, form url.Values, signedURL string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/"+id+"/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", umaSignTwilio(umaToken, signedURL, form))
	return r.do(t, req)
}

func umaWait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 15*time.Second, 10*time.Millisecond, what)
}

func TestUmaRoute_SignedWebhookRunsActionAndRepliesThroughUma(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))

	rec := r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000001", umaSender, "where is my order"), umaHookURL)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Header().Get("Content-Type"), "xml")
	assert.Contains(t, rec.Body.String(), "<Response>", "uma answers Twilio with TwiML, not the aps JSON envelope")
	assert.NotContains(t, rec.Body.String(), "<Message>", "the reply is a Messages API call, never inline TwiML")

	umaWait(t, "twilio create", func() bool { return len(r.twilio.sent()) == 1 })
	runs := r.core.calls()
	require.Len(t, runs, 1)
	assert.Equal(t, "assistant", runs[0].ProfileID)
	assert.Equal(t, "triage", runs[0].ActionID)
	sent := r.twilio.sent()[0]
	assert.Equal(t, umaSender, sent.Get("To"))
	assert.Equal(t, umaNumber, sent.Get("From"))
	assert.Equal(t, "on it", sent.Get("Body"))
}

func TestUmaRoute_BadSignatureNeverRunsAnAction(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	form := umaIncoming("SM10000000000000000000000000000002", umaSender, "hi")

	forged := r.postForm(t, "sms-uma", form, "https://evil.example.com/twilio/sms")
	assert.Equal(t, http.StatusForbidden, forged.Code, "signature over another URL")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/sms-uma/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	assert.Equal(t, http.StatusForbidden, r.do(t, req).Code, "no signature")

	// A later good message is the only one that runs: the refusals above
	// reached neither the action nor the provider.
	ok := r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000003", umaSender, "again"), umaHookURL)
	require.Equal(t, http.StatusOK, ok.Code)
	umaWait(t, "one reply", func() bool { return len(r.twilio.sent()) == 1 })
	assert.Len(t, r.core.calls(), 1)
}

func TestUmaRoute_SignatureCoversConfiguredURLNotForwardedHeaders(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	form := umaIncoming("SM10000000000000000000000000000004", umaSender, "behind a proxy")

	// The request arrives for another host and scheme than the one Twilio
	// signed; only the service's webhook_url counts.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://10.0.0.7:8080/services/sms-uma/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", umaSignTwilio(umaToken, umaHookURL, form))
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", "internal.example.net")
	assert.Equal(t, http.StatusOK, r.do(t, req).Code)

	// A signature built from the forwarded headers does not verify.
	form = umaIncoming("SM10000000000000000000000000000005", umaSender, "forwarded")
	req = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/sms-uma/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", umaSignTwilio(umaToken, "http://internal.example.net/services/sms-uma/webhook", form))
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", "internal.example.net")
	assert.Equal(t, http.StatusForbidden, r.do(t, req).Code)
}

func TestUmaRoute_RequestQueryIsPartOfTheSignedURL(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	form := umaIncoming("SM10000000000000000000000000000006", umaSender, "status")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/sms-uma/webhook?op=abc&attempt=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", umaSignTwilio(umaToken, umaHookURL+"?op=abc&attempt=1", form))
	assert.Equal(t, http.StatusOK, r.do(t, req).Code, "status callbacks carry the query uma put on the callback URL")
}

func TestUmaRoute_AllowedNumbersStillApply(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))

	stranger := r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000007", umaStranger, "let me in"), umaHookURL)
	assert.Equal(t, http.StatusOK, stranger.Code, "uma has accepted a signed event; the allowlist is aps's, applied after")

	known := r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000008", umaSender, "me"), umaHookURL)
	require.Equal(t, http.StatusOK, known.Code)
	umaWait(t, "reply to the allowed sender", func() bool { return len(r.twilio.sent()) == 1 })
	calls := r.core.calls()
	require.Len(t, calls, 1, "the stranger's message never ran an action")
	assert.Equal(t, umaSender, r.twilio.sent()[0].Get("To"))
}

func TestUmaRoute_RecordsServiceEvents(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))

	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000009", umaSender, "events"), umaHookURL).Code)

	umaWait(t, "inbound and outbound events", func() bool {
		svc, err := core.LoadService("sms-uma")
		return err == nil && svc.LastInbound != nil && svc.LastOutbound != nil
	})
	svc, err := core.LoadService("sms-uma")
	require.NoError(t, err)
	assert.Equal(t, "SM10000000000000000000000000000009", svc.LastInbound.MessageID)
	assert.Equal(t, "sms", svc.LastInbound.Platform)
	assert.Equal(t, umaSender, svc.LastInbound.SenderID)
	assert.Equal(t, "accepted", svc.LastOutbound.Status)
}

func TestUmaRoute_ServiceAddedAfterRouteRegistrationIsServed(t *testing.T) {
	r := newUmaRoutes(t)

	missing := r.postForm(t, "late-sms", umaIncoming("SM1", umaSender, "hi"), umaHookURL)
	assert.Equal(t, http.StatusNotFound, missing.Code)

	require.NoError(t, core.SaveService(umaSMSService("late-sms")))
	ok := r.postForm(t, "late-sms", umaIncoming("SM10000000000000000000000000000010", umaSender, "hi"), umaHookURL)
	assert.Equal(t, http.StatusOK, ok.Code)
}

func TestUmaRoute_ChangedServiceConfigurationTakesEffect(t *testing.T) {
	r := newUmaRoutes(t)
	svc := umaSMSService("sms-uma")
	require.NoError(t, core.SaveService(svc))
	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000011", umaSender, "one"), umaHookURL).Code)
	umaWait(t, "first reply", func() bool { return len(r.twilio.sent()) == 1 })

	svc.Options["from"] = "+15550001111"
	require.NoError(t, core.SaveService(svc))
	// The number the provider now delivers to is the new one.
	second := umaIncoming("SM10000000000000000000000000000012", umaSender, "two")
	second.Set("To", "+15550001111")
	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", second, umaHookURL).Code)
	umaWait(t, "second reply", func() bool { return len(r.twilio.sent()) == 2 })
	assert.Equal(t, "+15550001111", r.twilio.sent()[1].Get("From"))
}

func TestUmaRoute_UnavailableWhenTheBridgeCannotStart(t *testing.T) {
	r := newUmaRoutes(t)
	svc := umaSMSService("sms-uma")
	delete(svc.Env, core.EnvTwilioAccountSID)
	require.NoError(t, core.SaveService(svc))

	rec := r.postForm(t, "sms-uma", umaIncoming("SM1", umaSender, "hi"), umaHookURL)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "the provider retries a 5xx; nothing about the failure leaks")
	assert.NotContains(t, rec.Body.String(), "TWILIO", "binding names stay in the server log")
	assert.Empty(t, r.core.calls())
}

func TestUmaRoute_NotServedAfterTheAdapterStops(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000013", umaSender, "up"), umaHookURL).Code)

	require.NoError(t, r.adapter.Stop())

	rec := r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000014", umaSender, "down"), umaHookURL)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func umaMetaService() *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: "wa-cloud", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvWhatsAppToken:  "graph-token",
			core.EnvWhatsAppSecret: "app-secret",
			core.EnvWhatsAppVerify: "verify-me",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "meta.whatsapp",
			"phone_number_id":     "106540352242922",
			"default_action":      "triage",
			"receive":             "webhook",
			"reply":               "text",
		},
	}
}

func TestUmaRoute_WhatsAppCloudSubscriptionVerificationAnswersGET(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaMetaService()))
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {"verify-me"}, "hub.challenge": {"1158201444"}}

	rec := r.do(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/services/wa-cloud/webhook?"+q.Encode(), nil))
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "1158201444", strings.TrimSpace(rec.Body.String()))

	q.Set("hub.verify_token", "wrong")
	rec = r.do(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/services/wa-cloud/webhook?"+q.Encode(), nil))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "verify-me")
}

// The native providers keep their routes exactly: GET and every other method
// besides POST answer what the mux answered before uma services existed.
func TestUmaRoute_NativeServicesKeepTheirMethodRefusals(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID: "wa-native", Type: "message", Adapter: "whatsapp", Profile: "assistant",
		Options: map[string]string{"default_action": "triage"},
	}))

	for _, id := range []string{"wa-native", "does-not-exist"} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch} {
			t.Run(id+"/"+method, func(t *testing.T) {
				rec := r.do(t, httptest.NewRequestWithContext(t.Context(), method, "/services/"+id+"/webhook?hub.mode=subscribe", nil))
				assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
				assert.Equal(t, "POST", rec.Header().Get("Allow"))
				if method != http.MethodHead {
					assert.Equal(t, "Method Not Allowed\n", rec.Body.String())
				}
				assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
				assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
			})
		}
	}
}

func TestUmaRoute_PostToUnknownOrWrongTypeKeepsItsErrors(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(&core.ServiceConfig{ID: "a-ticket", Type: "ticket", Adapter: "email", Profile: "assistant"}))

	for id, want := range map[string]int{"nope": http.StatusNotFound, "a-ticket": http.StatusBadRequest} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/"+id+"/webhook", strings.NewReader("{}"))
		assert.Equal(t, want, r.do(t, req).Code, id)
	}
}

func TestUmaRoute_HandlerWithoutAUmaMountStillRefusesUmaServices(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	handler := NewHandler(NewMessageRouterWithExecutor(&serviceRouteResolver{base: &mockResolver{}}, NewNormalizer(), &fakeActionExecutor{}), NewNormalizer(), nil)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/services/sms-uma/webhook", strings.NewReader("From=%2B15551230000&Body=hi"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "sms-uma", core.AdapterUma)

	assert.Equal(t, http.StatusNotImplemented, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "uma")
}

func TestUmaRoute_RepliesLandInTheServicesConversationHistory(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))
	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000015", umaSender, "remember me"), umaHookURL).Code)
	umaWait(t, "reply", func() bool { return len(r.twilio.sent()) == 1 })

	// The conversation identity is the one the native SMS path builds from
	// the same sender, number and service.
	probe := &msgtypes.NormalizedMessage{
		Platform: string(msgtypes.PlatformSMS),
		Sender:   msgtypes.Sender{ID: umaSender},
		Channel:  msgtypes.Channel{ID: umaNumber},
		PlatformMetadata: map[string]any{
			"messenger_name": "sms-uma",
		},
	}
	var turns []msgtypes.ConversationTurn
	umaWait(t, "both turns recorded", func() bool {
		turns = recentTurnsFor(t, r.adapter.conversationStore(), probe)
		return len(turns) == 2
	})
	assert.Equal(t, []string{msgtypes.TurnDirectionInbound, msgtypes.TurnDirectionOutbound}, directionsOf(turns))
	assert.Equal(t, "remember me", turns[0].Text)
	assert.Equal(t, "on it", turns[1].Text)
	assert.Equal(t, "sms-uma", turns[0].ServiceID)
}
