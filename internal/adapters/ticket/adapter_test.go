package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/msgroute"
	"hop.top/aps/internal/core/protocol"
	"hop.top/kit/go/ai/ext"
)

const emailWebhookBody = `{
	"message_id": "<abc@example.com>",
	"from": "Alice <alice@example.com>",
	"to": "support@example.com",
	"subject": "Cannot log in",
	"body": "Password reset link never arrives."
}`

type fakeCore struct {
	protocol.APSCore
	inputs []protocol.RunInput
	status protocol.RunStatus
	output string
}

func (f *fakeCore) ExecuteRun(_ context.Context, input protocol.RunInput, _ protocol.StreamWriter) (*protocol.RunState, error) {
	f.inputs = append(f.inputs, input)
	status := f.status
	if status == "" {
		status = protocol.RunStatusCompleted
	}
	output := f.output
	if output == "" {
		output = "handled"
	}
	return &protocol.RunState{ProfileID: input.ProfileID, ActionID: input.ActionID, Status: status, Output: output}, nil
}

func saveTicketService(t *testing.T, service *core.ServiceConfig) {
	t.Helper()
	if err := core.SaveService(service); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
}

func newTestMux(t *testing.T, apsCore protocol.APSCore) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	if err := NewAdapter().RegisterRoutes(mux, apsCore); err != nil {
		t.Fatalf("RegisterRoutes: %v", err)
	}
	return mux
}

func postJSON(mux http.Handler, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAdapter_ImplementsExt(t *testing.T) {
	var _ ext.Extension = (*Adapter)(nil)
	a := NewAdapter()
	if a.Meta().Name != "ticket" {
		t.Fatalf("Meta.Name = %q, want ticket", a.Meta().Name)
	}
	if err := a.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if a.Status() != "running" {
		t.Fatalf("status = %q, want running", a.Status())
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if a.Status() != "stopped" {
		t.Fatalf("status = %q, want stopped", a.Status())
	}
}

// The route the adapter mounts must be the route core reports to operators.
func TestAdapter_MountsServiceWebhookPathAndExecutesAction(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID:      "support-inbox",
		Type:    "ticket",
		Adapter: AdapterEmail,
		Profile: "inbox",
		Options: map[string]string{core.OptionDefaultAction: "triage"},
	}
	saveTicketService(t, service)
	apsCore := &fakeCore{output: "ack"}
	mux := newTestMux(t, apsCore)

	rec := postJSON(mux, core.ServiceWebhookPath(service), emailWebhookBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 1 {
		t.Fatalf("ExecuteRun calls = %d, want 1", len(apsCore.inputs))
	}
	input := apsCore.inputs[0]
	if input.ProfileID != "inbox" || input.ActionID != "triage" {
		t.Fatalf("run input = %s/%s, want inbox/triage", input.ProfileID, input.ActionID)
	}
	var payload map[string]any
	if err := json.Unmarshal(input.Payload, &payload); err != nil {
		t.Fatalf("payload is not JSON: %v; raw=%s", err, input.Payload)
	}
	if payload["title"] != "Cannot log in" || payload["body"] != "Password reset link never arrives." {
		t.Fatalf("payload missing normalized ticket fields: %s", input.Payload)
	}
	if payload["service_id"] != "support-inbox" {
		t.Fatalf("payload service_id = %v, want support-inbox", payload["service_id"])
	}
	author, _ := payload["author"].(map[string]any)
	if author["email"] != "alice@example.com" {
		t.Fatalf("payload author = %v, want alice@example.com", payload["author"])
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v; raw=%s", err, rec.Body.String())
	}
	if response["status"] != "success" || response["body"] != "ack" {
		t.Fatalf("response = %v, want status success body ack", response)
	}
	if response["ticket_id"] != "<abc@example.com>" {
		t.Fatalf("response ticket_id = %v", response["ticket_id"])
	}

	updated, err := core.LoadService(service.ID)
	if err != nil {
		t.Fatalf("LoadService: %v", err)
	}
	if updated.LastInbound == nil || updated.LastInbound.Status != "received" {
		t.Fatalf("inbound event not recorded: %+v", updated.LastInbound)
	}
	if updated.LastOutbound == nil || updated.LastOutbound.Status != "success" {
		t.Fatalf("outbound event not recorded: %+v", updated.LastOutbound)
	}
}

func TestAdapter_FailedActionIsReported(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID:      "support-inbox",
		Type:    "ticket",
		Adapter: AdapterEmail,
		Profile: "inbox",
		Options: map[string]string{core.OptionDefaultAction: "triage"},
	}
	saveTicketService(t, service)
	mux := newTestMux(t, &fakeCore{status: protocol.RunStatusFailed, output: "boom"})

	rec := postJSON(mux, core.ServiceWebhookPath(service), emailWebhookBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	if response["status"] != "failed" || response["body"] != "boom" {
		t.Fatalf("response = %v, want failed/boom", response)
	}
}

func TestAdapter_UnknownServiceIs404(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mux := newTestMux(t, &fakeCore{})

	rec := postJSON(mux, "/services/ghost/ticket/email", emailWebhookBody, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdapter_NoCatchAllRoute(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)

	for _, path := range []string{"/tickets/email/webhook", "/services/ticket/email", "/ticket/email"} {
		rec := postJSON(mux, path, emailWebhookBody, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, rec.Code)
		}
	}
	if len(apsCore.inputs) != 0 {
		t.Fatalf("no action may run without a service; ran %d", len(apsCore.inputs))
	}
}

func TestAdapter_RejectsTypeAndAdapterMismatch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveTicketService(t, &core.ServiceConfig{
		ID: "chat", Type: "message", Adapter: "telegram", Profile: "inbox",
		Options: map[string]string{core.OptionDefaultAction: "reply"},
	})
	saveTicketService(t, &core.ServiceConfig{
		ID: "issues", Type: "ticket", Adapter: AdapterJira, Profile: "inbox",
		Options: map[string]string{core.OptionDefaultAction: "triage"},
	})
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)

	if rec := postJSON(mux, "/services/chat/ticket/email", emailWebhookBody, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("message service on ticket route: status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if rec := postJSON(mux, "/services/issues/ticket/email", emailWebhookBody, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("adapter mismatch: status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 0 {
		t.Fatalf("no action may run on mismatch; ran %d", len(apsCore.inputs))
	}
}

func TestAdapter_GetIsMethodNotAllowed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mux := newTestMux(t, &fakeCore{})
	req := httptest.NewRequest(http.MethodGet, "/services/support-inbox/ticket/email", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

// Generic service auth (core/messenger ServiceValidator) gates the route:
// a bearer token configured on the service must be presented.
func TestAdapter_EnforcesServiceBearerToken(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("SUPPORT_INBOX_TOKEN", "s3cret")
	service := &core.ServiceConfig{
		ID: "support-inbox", Type: "ticket", Adapter: AdapterEmail, Profile: "inbox",
		Options: map[string]string{
			core.OptionDefaultAction: "triage",
			"auth_token_env":         "SUPPORT_INBOX_TOKEN",
		},
	}
	saveTicketService(t, service)
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)
	path := core.ServiceWebhookPath(service)

	if rec := postJSON(mux, path, emailWebhookBody, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if rec := postJSON(mux, path, emailWebhookBody, map[string]string{"Authorization": "Bearer wrong"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 0 {
		t.Fatalf("unauthenticated requests must not execute; ran %d", len(apsCore.inputs))
	}
	if rec := postJSON(mux, path, emailWebhookBody, map[string]string{"Authorization": "Bearer s3cret"}); rec.Code != http.StatusOK {
		t.Fatalf("valid token: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 1 {
		t.Fatalf("authenticated request should execute once; ran %d", len(apsCore.inputs))
	}
}

func TestAdapter_AllowedSendersGate(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID: "support-inbox", Type: "ticket", Adapter: AdapterEmail, Profile: "inbox",
		Options: map[string]string{
			core.OptionDefaultAction: "triage",
			"allowed_senders":        "bob@example.com, *@corp.example",
		},
	}
	saveTicketService(t, service)
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)
	path := core.ServiceWebhookPath(service)

	if rec := postJSON(mux, path, emailWebhookBody, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("unlisted sender: status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	corp := strings.Replace(emailWebhookBody, "alice@example.com", "carol@corp.example", 1)
	if rec := postJSON(mux, path, corp, nil); rec.Code != http.StatusOK {
		t.Fatalf("glob-listed sender: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 1 {
		t.Fatalf("ExecuteRun calls = %d, want 1", len(apsCore.inputs))
	}
}

func TestAdapter_InvalidPayloadIs400(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID: "support-inbox", Type: "ticket", Adapter: AdapterEmail, Profile: "inbox",
		Options: map[string]string{core.OptionDefaultAction: "triage"},
	}
	saveTicketService(t, service)
	mux := newTestMux(t, &fakeCore{})
	path := core.ServiceWebhookPath(service)

	if rec := postJSON(mux, path, "not json", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON: status = %d, want 400", rec.Code)
	}
	if rec := postJSON(mux, path, `{"subject":"no sender"}`, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("unnormalizable payload: status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdapter_UnroutedServiceIsReported(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID: "support-inbox", Type: "ticket", Adapter: AdapterEmail, Profile: "inbox",
	}
	saveTicketService(t, service)
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)

	rec := postJSON(mux, core.ServiceWebhookPath(service), emailWebhookBody, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 0 {
		t.Fatalf("unrouted ticket must not execute; ran %d", len(apsCore.inputs))
	}
}

func routeTableFixture(t *testing.T) *msgroute.Config {
	t.Helper()
	return &msgroute.Config{Routes: []msgroute.Route{
		{Match: "*@vip.example", Profile: "concierge", Action: "escalate"},
		{Match: msgroute.TerminalMatch, Action: "triage"},
	}}
}

func TestAdapter_RouteTableDispatchesBySender(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := &core.ServiceConfig{
		ID: "support-inbox", Type: "ticket", Adapter: AdapterEmail, Profile: "inbox",
		Routing: routeTableFixture(t),
	}
	saveTicketService(t, service)
	apsCore := &fakeCore{}
	mux := newTestMux(t, apsCore)
	path := core.ServiceWebhookPath(service)

	if rec := postJSON(mux, path, emailWebhookBody, nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	vip := strings.Replace(emailWebhookBody, "alice@example.com", "ceo@vip.example", 1)
	if rec := postJSON(mux, path, vip, nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(apsCore.inputs) != 2 {
		t.Fatalf("ExecuteRun calls = %d, want 2", len(apsCore.inputs))
	}
	if got := apsCore.inputs[0]; got.ProfileID != "inbox" || got.ActionID != "triage" {
		t.Fatalf("unknown sender routed to %s/%s, want inbox/triage", got.ProfileID, got.ActionID)
	}
	if got := apsCore.inputs[1]; got.ProfileID != "concierge" || got.ActionID != "escalate" {
		t.Fatalf("vip sender routed to %s/%s, want concierge/escalate", got.ProfileID, got.ActionID)
	}
}
