package umabridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- Twilio's documented webhook signature algorithm
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

const (
	testAccount = "AC00000000000000000000000000000001"
	testToken   = "fake-auth-token" // #nosec G101 -- fixture value
	testHookURL = "https://hooks.example.com/twilio/sms"
	testNumber  = "+15558675310"
	testSender  = "+15551230000"
)

// fakeTwilio is a Twilio Messages API stand-in: it records each create and
// answers it as Twilio does.
type fakeTwilio struct {
	t   *testing.T
	srv *httptest.Server

	mu    sync.Mutex
	posts []twilioPost
	seq   int
	media map[string][]byte // media SID -> bytes, set before the webhook is posted
	// reject, when non-zero, answers every message create with this status
	// and a Twilio error body.
	reject int
}

type twilioPost struct {
	Path     string
	User     string
	Password string
	Form     url.Values
}

func newFakeTwilio(t *testing.T) *fakeTwilio {
	t.Helper()
	f := &fakeTwilio{t: t}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTwilio) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/Media/") {
		f.mu.Lock()
		data, ok := f.media[r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]]
		f.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(data)
		return
	}
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/Messages.json") {
		http.Error(w, "unexpected", http.StatusNotFound)
		return
	}
	_ = r.ParseForm()
	user, pass, _ := r.BasicAuth()
	f.mu.Lock()
	reject := f.reject
	f.mu.Unlock()
	if reject != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reject)
		_, _ = w.Write([]byte(`{"code":21211,"message":"The 'To' number is not a valid phone number.","more_info":"https://www.twilio.com/docs/errors/21211","status":400}`))
		return
	}
	f.mu.Lock()
	f.seq++
	n := f.seq
	f.posts = append(f.posts, twilioPost{Path: r.URL.Path, User: user, Password: pass, Form: r.PostForm})
	f.mu.Unlock()
	sid := fmt.Sprintf("SM%032x", n)
	uri := "/2010-04-01/Accounts/" + testAccount + "/Messages/" + sid
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"account_sid": testAccount, "api_version": "2010-04-01", "body": r.PostForm.Get("Body"),
		"date_created": time.Now().UTC().Format(time.RFC1123Z), "date_updated": time.Now().UTC().Format(time.RFC1123Z),
		"date_sent": nil, "direction": "outbound-api", "error_code": nil, "error_message": nil,
		"from": r.PostForm.Get("From"), "messaging_service_sid": nil, "num_media": "0", "num_segments": "1",
		"price": nil, "sid": sid, "status": "queued", "to": r.PostForm.Get("To"), "uri": uri + ".json",
		"subresource_uris": map[string]string{"media": uri + "/Media.json"},
	})
}

func (f *fakeTwilio) sent() []twilioPost {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]twilioPost(nil), f.posts...)
}

// stubRouter routes every message to assistant=triage.
type stubRouter struct{}

func (stubRouter) ResolveMessageRoute(context.Context, *msgtypes.NormalizedMessage) (msgtypes.ExecutionRoute, error) {
	return msgtypes.ExecutionRoute{ProfileID: "assistant", ActionName: "triage"}, nil
}

// stubExecutor records handoffs and answers from fn.
type stubExecutor struct {
	mu       sync.Mutex
	handoffs []msgtypes.ExecutionHandoff
	fn       func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error)
}

func (e *stubExecutor) ExecuteMessage(_ context.Context, h msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	e.mu.Lock()
	e.handoffs = append(e.handoffs, h)
	fn := e.fn
	e.mu.Unlock()
	return fn(h)
}

func (e *stubExecutor) calls() []msgtypes.ExecutionHandoff {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]msgtypes.ExecutionHandoff(nil), e.handoffs...)
}

func replyWith(text string) func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	return func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		return &msgtypes.ExecutionResult{Status: "success", Output: text, Reply: &msgtypes.DeliveryRequest{Text: text}}, nil
	}
}

func smsService() *core.ServiceConfig {
	return &core.ServiceConfig{
		ID: "sms-uma", Type: "message", Adapter: core.AdapterUma, Profile: "assistant",
		Env: map[string]string{
			core.EnvTwilioAccountSID: testAccount,
			core.EnvTwilioAuthToken:  "secret:TWILIO_AUTH_TOKEN",
		},
		Options: map[string]string{
			core.OptionUmaAdapter: "twilio.sms",
			"from":                testNumber,
			"webhook_url":         testHookURL,
			"default_action":      "triage",
			"receive":             "webhook",
			"reply":               "text",
			"allowed_numbers":     testSender,
		},
	}
}

func tokenStore(token string) core.SecretLookup {
	return func(name string) (string, bool) {
		if name == "TWILIO_AUTH_TOKEN" {
			return token, true
		}
		return "", false
	}
}

type harness struct {
	bridge *Bridge
	hook   *httptest.Server
	exec   *stubExecutor
	twilio *fakeTwilio
	// stop ends the bridge as a process exit would: its engine, and every
	// message queued or recorded in it, are gone. Idempotent.
	stop func()

	mu      sync.Mutex
	results []resultRecord
}

type resultRecord struct {
	result *msgtypes.RuntimeResult
	err    error
}

func (h *harness) outcomes() []resultRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]resultRecord(nil), h.results...)
}

func newHarness(t *testing.T, svc *core.ServiceConfig, fn func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error)) *harness {
	return newHarnessWith(t, svc, fn, nil)
}

func newHarnessWith(t *testing.T, svc *core.ServiceConfig, fn func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error), mutate func(*Config)) *harness {
	t.Helper()
	return newHarnessOn(t, newFakeTwilio(t), svc, fn, mutate)
}

// newHarnessOn is newHarnessWith against a given provider stand-in, so a
// second harness can play the same service after a restart.
func newHarnessOn(t *testing.T, twilio *fakeTwilio, svc *core.ServiceConfig, fn func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error), mutate func(*Config)) *harness {
	t.Helper()
	h := &harness{twilio: twilio, exec: &stubExecutor{fn: fn}}
	cfg := Config{
		Service:  svc,
		Router:   stubRouter{},
		Executor: h.exec,
		Store:    tokenStore(testToken),
		Options:  Options{TwilioBaseURL: h.twilio.srv.URL},
		Poll:     10 * time.Millisecond,
		OnResult: func(r *msgtypes.RuntimeResult, err error) {
			h.mu.Lock()
			h.results = append(h.results, resultRecord{r, err})
			h.mu.Unlock()
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	b, err := New(t.Context(), cfg)
	require.NoError(t, err)
	h.bridge = b
	h.hook = httptest.NewServer(b.Webhook())
	t.Cleanup(h.hook.Close)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()
	var once sync.Once
	h.stop = func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				assert.NoError(t, err, "bridge run")
			case <-time.After(15 * time.Second):
				t.Error("bridge did not stop")
			}
			assert.NoError(t, b.Close(context.Background()))
		})
	}
	t.Cleanup(h.stop)
	return h
}

// signTwilio computes X-Twilio-Signature independently of the adapter.
func signTwilio(token, rawURL string, form url.Values) string {
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

func incomingSMS(sid, body string) url.Values {
	return url.Values{
		"MessageSid": {sid}, "SmsSid": {sid}, "AccountSid": {testAccount}, "SmsStatus": {"received"},
		"From": {testSender}, "To": {testNumber}, "Body": {body}, "NumMedia": {"0"}, "NumSegments": {"1"},
		"ApiVersion": {"2010-04-01"},
	}
}

func (h *harness) post(t *testing.T, form url.Values, signature string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.hook.URL, strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", signature)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func (h *harness) postSigned(t *testing.T, form url.Values) *http.Response {
	t.Helper()
	return h.post(t, form, signTwilio(testToken, testHookURL, form))
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 15*time.Second, 10*time.Millisecond, what)
}

func TestInboundSMSRunsActionAndRepliesThroughMessagesAPI(t *testing.T) {
	h := newHarness(t, smsService(), replyWith("on it"))

	resp := h.postSigned(t, incomingSMS("SM10000000000000000000000000000001", "where is my order"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	waitFor(t, "twilio create", func() bool { return len(h.twilio.sent()) == 1 })

	calls := h.exec.calls()
	require.Len(t, calls, 1)
	handoff := calls[0]
	assert.Equal(t, "sms-uma", handoff.ServiceID)
	assert.Equal(t, "sms", handoff.Provider)
	assert.Equal(t, "assistant", handoff.ProfileID)
	assert.Equal(t, "triage", handoff.ActionName)
	msg := handoff.Message
	require.NotNil(t, msg)
	assert.Equal(t, "SM10000000000000000000000000000001", msg.ID)
	assert.Equal(t, "sms", msg.Platform)
	assert.Equal(t, testSender, msg.Sender.ID)
	assert.Equal(t, testNumber, msg.Channel.ID)
	assert.Equal(t, "direct", msg.Channel.Type)
	assert.Equal(t, "where is my order", msg.Text)
	assert.Equal(t, "sms-uma", msg.PlatformMetadata["messenger_name"])
	assert.Equal(t, "twilio.sms", msg.PlatformMetadata["uma_adapter"])

	post := h.twilio.sent()[0]
	assert.Equal(t, "/2010-04-01/Accounts/"+testAccount+"/Messages.json", post.Path)
	assert.Equal(t, testAccount, post.User, "REST basic auth user is the account SID")
	assert.Equal(t, testToken, post.Password, "REST credential resolved from the secret binding")
	assert.Equal(t, testSender, post.Form.Get("To"))
	assert.Equal(t, testNumber, post.Form.Get("From"))
	assert.Equal(t, "on it", post.Form.Get("Body"))

	waitFor(t, "result recorded", func() bool { return len(h.outcomes()) == 1 })
	out := h.outcomes()[0]
	require.NoError(t, out.err)
	require.NotNil(t, out.result.Delivery)
	assert.Equal(t, "accepted", out.result.Delivery.Status)
}

func TestInboundWithBadSignatureNeverReachesTheAction(t *testing.T) {
	h := newHarness(t, smsService(), replyWith("on it"))
	form := incomingSMS("SM10000000000000000000000000000002", "hi")

	for name, signature := range map[string]string{
		"wrong": signTwilio("another-token", testHookURL, form),
		"empty": "",
	} {
		t.Run(name, func(t *testing.T) {
			resp := h.post(t, form, signature)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, h.exec.calls())
	assert.Empty(t, h.twilio.sent())
}

func TestSenderOutsideTheAllowlistNeverRunsTheAction(t *testing.T) {
	h := newHarness(t, smsService(), replyWith("on it"))
	stranger := incomingSMS("SM10000000000000000000000000000020", "let me in")
	stranger.Set("From", "+15559998888")

	// uma verified and recorded the event, so the provider is answered; the
	// allowlist is aps's and refuses it before routing.
	assert.Equal(t, http.StatusOK, h.postSigned(t, stranger).StatusCode)
	waitFor(t, "refusal recorded", func() bool { return len(h.outcomes()) == 1 })
	out := h.outcomes()[0]
	require.Error(t, out.err)
	assert.True(t, msgtypes.IsSenderNotAllowed(out.err), "got %v", out.err)
	assert.Nil(t, out.result)
	assert.Empty(t, h.exec.calls())

	// The bridge keeps serving: the allowed sender still gets through.
	assert.Equal(t, http.StatusOK, h.postSigned(t, incomingSMS("SM10000000000000000000000000000021", "hello")).StatusCode)
	waitFor(t, "reply", func() bool { return len(h.twilio.sent()) == 1 })
	assert.Equal(t, testSender, h.twilio.sent()[0].Form.Get("To"))
	assert.Len(t, h.exec.calls(), 1)
}

func TestAllowedNumbersAndSendersAreCheckedOnTheNormalizedMessage(t *testing.T) {
	tests := map[string]struct {
		option string
		value  string
		sender string
		allow  bool
	}{
		"number listed":      {"allowed_numbers", testSender, testSender, true},
		"number not listed":  {"allowed_numbers", "+15550000001", testSender, false},
		"sender not listed":  {core.OptionAllowedSenders, "+15550000001", testSender, false},
		"channel not listed": {"allowed_channels", "+15550000001", testSender, false},
		"channel is ours":    {"allowed_channels", testNumber, testSender, true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc := smsService()
			delete(svc.Options, "allowed_numbers")
			svc.Options[tc.option] = tc.value
			h := newHarness(t, svc, replyWith("ok"))
			assert.Equal(t, http.StatusOK, h.postSigned(t, incomingSMS("SM10000000000000000000000000000022", "hi")).StatusCode)
			waitFor(t, "result", func() bool { return len(h.outcomes()) == 1 })
			if tc.allow {
				require.NoError(t, h.outcomes()[0].err)
				assert.Len(t, h.exec.calls(), 1)
				return
			}
			assert.True(t, msgtypes.IsSenderNotAllowed(h.outcomes()[0].err))
			assert.Empty(t, h.exec.calls())
		})
	}
}

func TestActionWithoutReplySendsNothing(t *testing.T) {
	h := newHarness(t, smsService(), func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		return &msgtypes.ExecutionResult{Status: "success", Output: "done"}, nil
	})
	h.postSigned(t, incomingSMS("SM10000000000000000000000000000003", "quiet"))

	waitFor(t, "result recorded", func() bool { return len(h.outcomes()) == 1 })
	require.NoError(t, h.outcomes()[0].err)
	assert.Empty(t, h.twilio.sent())
}

func TestActionFailureIsReportedAndBridgeKeepsServing(t *testing.T) {
	failures := 0
	h := newHarness(t, smsService(), func(msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
		if failures == 0 {
			failures++
			return nil, errors.New("action crashed")
		}
		return &msgtypes.ExecutionResult{Status: "success", Reply: &msgtypes.DeliveryRequest{Text: "second ok"}}, nil
	})

	h.postSigned(t, incomingSMS("SM10000000000000000000000000000004", "first"))
	waitFor(t, "first result", func() bool { return len(h.outcomes()) == 1 })
	first := h.outcomes()[0]
	require.Error(t, first.err)
	var rerr *msgtypes.RuntimeError
	require.ErrorAs(t, first.err, &rerr)
	assert.Equal(t, msgtypes.RuntimeStageExecute, rerr.Stage)
	assert.Empty(t, h.twilio.sent(), "a failed action sends no reply")

	h.postSigned(t, incomingSMS("SM10000000000000000000000000000005", "second"))
	waitFor(t, "second reply", func() bool { return len(h.twilio.sent()) == 1 })
	assert.Equal(t, "second ok", h.twilio.sent()[0].Form.Get("Body"))
	assert.Len(t, h.exec.calls(), 2, "the failed message is not run again")
}

func TestRedeliveredProviderEventRunsTheActionOnce(t *testing.T) {
	h := newHarness(t, smsService(), replyWith("once"))
	form := incomingSMS("SM10000000000000000000000000000006", "dup")

	assert.Equal(t, http.StatusOK, h.postSigned(t, form).StatusCode)
	assert.Equal(t, http.StatusOK, h.postSigned(t, form).StatusCode)

	waitFor(t, "reply", func() bool { return len(h.twilio.sent()) >= 1 })
	time.Sleep(300 * time.Millisecond)
	assert.Len(t, h.exec.calls(), 1)
	assert.Len(t, h.twilio.sent(), 1)
}

func TestNewRejectsAServiceUmaCannotRun(t *testing.T) {
	tests := map[string]func(*core.ServiceConfig){
		"no webhook url":     func(s *core.ServiceConfig) { delete(s.Options, "webhook_url") },
		"webhook url query":  func(s *core.ServiceConfig) { s.Options["webhook_url"] = testHookURL + "?x=1" },
		"unknown adapter":    func(s *core.ServiceConfig) { s.Options[core.OptionUmaAdapter] = "twilio.fax" },
		"missing token bind": func(s *core.ServiceConfig) { delete(s.Env, core.EnvTwilioAuthToken) },
		"native adapter":     func(s *core.ServiceConfig) { s.Adapter = "sms" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			svc := smsService()
			mutate(svc)
			_, err := New(t.Context(), Config{Service: svc, Router: stubRouter{}, Executor: &stubExecutor{}, Store: tokenStore(testToken)})
			require.Error(t, err)
		})
	}
	_, err := New(t.Context(), Config{Service: smsService(), Executor: &stubExecutor{}})
	require.Error(t, err, "router required")
}

func TestNewFailsWhenAccountSIDBindingResolvesToNothing(t *testing.T) {
	svc := smsService()
	svc.Env[core.EnvTwilioAccountSID] = "secret:NOT_THERE"
	_, err := New(t.Context(), Config{Service: svc, Router: stubRouter{}, Executor: &stubExecutor{}, Store: tokenStore(testToken)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), core.EnvTwilioAccountSID)
}

func (f *fakeTwilio) rejectWith(status int) {
	f.mu.Lock()
	f.reject = status
	f.mu.Unlock()
}

func (f *fakeTwilio) setMedia(m map[string][]byte) {
	f.mu.Lock()
	f.media = m
	f.mu.Unlock()
}
