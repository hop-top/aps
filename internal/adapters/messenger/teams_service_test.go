package messenger

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infracloudio/msbotbuilder-go/schema"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

type teamsAuthenticatorStub struct {
	err error
}

func (s teamsAuthenticatorStub) AuthenticateTeamsRequest(context.Context, schema.Activity, string, string) error {
	return s.err
}

func teamsTestValidator(authErr error) *msgtypes.ServiceValidator {
	validator := msgtypes.NewServiceValidator()
	validator.Hooks[string(msgtypes.PlatformTeams)] = msgtypes.TeamsAuthHook{
		Authenticator: teamsAuthenticatorStub{err: authErr},
	}
	return validator
}

func saveTeamsService(t *testing.T, options map[string]string) {
	t.Helper()
	if options == nil {
		options = map[string]string{}
	}
	if _, ok := options["default_action"]; !ok {
		options["default_action"] = "assistant=handle_teams"
	}
	if _, ok := options["reply"]; !ok {
		options["reply"] = "text"
	}
	service := &core.ServiceConfig{
		ID:      "teams-support",
		Type:    "message",
		Adapter: "teams",
		Profile: "assistant",
		Env: map[string]string{
			"TEAMS_APP_ID":       "bot-app-id",
			"TEAMS_APP_PASSWORD": "bot-secret",
		},
		Options: options,
	}
	if err := core.SaveService(service); err != nil {
		t.Fatalf("save teams service: %v", err)
	}
}

const teamsChannelActivity = `{
	"type": "message",
	"id": "1750000000001",
	"timestamp": "2026-08-27T09:00:00Z",
	"serviceUrl": "https://smba.trafficmanager.net/emea/",
	"channelId": "msteams",
	"from": {"id": "29:2ghijkl", "name": "Adele Vance"},
	"conversation": {"conversationType": "channel", "id": "19:channel@thread.tacv2;messageid=1749000000000"},
	"recipient": {"id": "28:bot-app-id", "name": "Support Bot"},
	"text": "<at>Support Bot</at> please summarize",
	"channelData": {
		"tenant": {"id": "tenant-guid"},
		"team": {"id": "19:team@thread.tacv2"},
		"channel": {"id": "19:channel@thread.tacv2"}
	}
}`

func teamsRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/services/teams-support/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer bot-framework-jwt")
	return req
}

func TestTeamsService_AcknowledgesNonMessageActivities(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveTeamsService(t, nil)

	executor := &fakeActionExecutor{}
	transport := &fakeTeamsTransport{}
	handler := newServiceTestHandler(executor,
		WithServiceValidator(teamsTestValidator(nil)),
		WithTeamsTransport(transport))

	body := `{"type":"conversationUpdate","id":"999","serviceUrl":"https://smba.trafficmanager.net/emea/","from":{"id":"29:2ghijkl"},"conversation":{"id":"19:group"},"membersAdded":[{"id":"28:bot-app-id"}]}`
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, teamsRequest(body), "teams-support", "teams")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ignored") {
		t.Fatalf("body = %q, want ignored acknowledgement", rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
	}
	if transport.calls != 0 {
		t.Fatalf("transport calls = %d, want 0", transport.calls)
	}
}

func TestTeamsService_RejectsMissingAuthorization(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveTeamsService(t, nil)

	handler := newServiceTestHandler(&fakeActionExecutor{},
		WithServiceValidator(teamsTestValidator(nil)),
		WithTeamsTransport(&fakeTeamsTransport{}))

	req := teamsRequest(teamsChannelActivity)
	req.Header.Del("Authorization")
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "teams-support", "teams")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestTeamsService_RejectsInvalidToken(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveTeamsService(t, nil)

	handler := newServiceTestHandler(&fakeActionExecutor{},
		WithServiceValidator(teamsTestValidator(errors.New("bad token"))),
		WithTeamsTransport(&fakeTeamsTransport{}))

	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, teamsRequest(teamsChannelActivity), "teams-support", "teams")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestTeamsService_ExecutesAndDeliversThreadedReply(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveTeamsService(t, map[string]string{
		"default_action":   "assistant=handle_teams",
		"reply":            "text",
		"allowed_channels": "19:channel@thread.tacv2",
	})

	executor := &fakeActionExecutor{output: "summary ready"}
	transport := &fakeTeamsTransport{}
	handler := newServiceTestHandler(executor,
		WithServiceValidator(teamsTestValidator(nil)),
		WithTeamsTransport(transport))

	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, teamsRequest(teamsChannelActivity), "teams-support", "teams")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.ProfileID != "assistant" {
		t.Fatalf("executor profile = %q, want assistant", executor.input.ProfileID)
	}
	if transport.calls != 1 {
		t.Fatalf("transport calls = %d, want 1 delivery", transport.calls)
	}
	wantURL := "https://smba.trafficmanager.net/emea/v3/conversations/19:channel@thread.tacv2;messageid=1749000000000/activities/1750000000001"
	if transport.gotURL.String() != wantURL {
		t.Errorf("delivery url = %q, want %q", transport.gotURL.String(), wantURL)
	}
	if transport.gotActivity.Text != "summary ready" {
		t.Errorf("delivered text = %q, want executor output", transport.gotActivity.Text)
	}
	if transport.gotActivity.From.ID != "28:bot-app-id" {
		t.Errorf("delivered from = %q, want bot id", transport.gotActivity.From.ID)
	}
}
