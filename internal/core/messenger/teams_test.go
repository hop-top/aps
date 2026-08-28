package messenger

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/infracloudio/msbotbuilder-go/schema"
)

type fakeTeamsAuthenticator struct {
	err        error
	called     bool
	gotHeader  string
	gotAppID   string
	gotService string
}

func (f *fakeTeamsAuthenticator) AuthenticateTeamsRequest(_ context.Context, activity schema.Activity, authHeader, appID string) error {
	f.called = true
	f.gotHeader = authHeader
	f.gotAppID = appID
	f.gotService = activity.ServiceURL
	return f.err
}

func teamsValidationInput(headers http.Header, body []byte) RequestValidationInput {
	return RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "teams-support",
			Adapter: string(PlatformTeams),
			Env: map[string]string{
				"TEAMS_APP_ID": "bot-app-id",
			},
		},
		Method:  http.MethodPost,
		Headers: headers,
		Body:    body,
	}
}

func TestTeamsAuthHook_ValidRequest(t *testing.T) {
	fake := &fakeTeamsAuthenticator{}
	hook := TeamsAuthHook{Authenticator: fake}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer token-value")
	body := []byte(`{"type":"message","serviceUrl":"https://smba.trafficmanager.net/amer/"}`)

	if err := hook.ValidateProviderRequest(context.Background(), teamsValidationInput(headers, body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fake.called {
		t.Fatal("authenticator was not called")
	}
	if fake.gotHeader != "Bearer token-value" {
		t.Errorf("authHeader = %q", fake.gotHeader)
	}
	if fake.gotAppID != "bot-app-id" {
		t.Errorf("appID = %q, want resolved TEAMS_APP_ID", fake.gotAppID)
	}
	if fake.gotService != "https://smba.trafficmanager.net/amer/" {
		t.Errorf("activity.ServiceURL = %q, want parsed from body", fake.gotService)
	}
}

func TestTeamsAuthHook_RejectsInvalidToken(t *testing.T) {
	fake := &fakeTeamsAuthenticator{err: errors.New("boom")}
	hook := TeamsAuthHook{Authenticator: fake}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer bad")

	err := hook.ValidateProviderRequest(context.Background(), teamsValidationInput(headers, []byte(`{"type":"message"}`)))
	if !IsAuthFailed(err) {
		t.Fatalf("err = %v, want auth failed", err)
	}
}

func TestTeamsAuthHook_MissingAuthorizationHeader(t *testing.T) {
	fake := &fakeTeamsAuthenticator{}
	hook := TeamsAuthHook{Authenticator: fake}

	err := hook.ValidateProviderRequest(context.Background(), teamsValidationInput(http.Header{}, []byte(`{"type":"message"}`)))
	if !IsAuthFailed(err) {
		t.Fatalf("err = %v, want auth failed", err)
	}
	if fake.called {
		t.Fatal("authenticator must not run without an authorization header")
	}
}

func TestTeamsAuthHook_MissingAppID(t *testing.T) {
	hook := TeamsAuthHook{Authenticator: &fakeTeamsAuthenticator{}}

	input := teamsValidationInput(http.Header{}, []byte(`{"type":"message"}`))
	input.Service.Env = nil

	err := hook.ValidateProviderRequest(context.Background(), input)
	var msgErr *MessengerError
	if !errors.As(err, &msgErr) || msgErr.Code != ErrCodeMissingSecret {
		t.Fatalf("err = %v, want missing secret", err)
	}
}

func TestTeamsAuthHook_InvalidBody(t *testing.T) {
	hook := TeamsAuthHook{Authenticator: &fakeTeamsAuthenticator{}}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer token")

	err := hook.ValidateProviderRequest(context.Background(), teamsValidationInput(headers, []byte("not-json")))
	if !IsAuthFailed(err) {
		t.Fatalf("err = %v, want auth failed", err)
	}
}

func TestServiceValidator_TeamsHookRegistered(t *testing.T) {
	validator := NewServiceValidator()
	summary := validator.DescribeAuth(ServiceValidationConfig{
		ID:      "teams-support",
		Adapter: string(PlatformTeams),
	})
	if summary.Provider != string(PlatformTeams) {
		t.Fatalf("provider = %q, want teams", summary.Provider)
	}
	if !summary.ProviderValidated {
		t.Fatal("teams hook must validate the raw request itself")
	}
}
