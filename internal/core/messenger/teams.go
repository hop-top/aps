package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/infracloudio/msbotbuilder-go/connector/auth"
	"github.com/infracloudio/msbotbuilder-go/schema"

	"hop.top/aps/internal/core"
)

// TeamsTokenAuthenticator verifies the Bot Framework Authorization token
// attached to an inbound Teams activity.
type TeamsTokenAuthenticator interface {
	AuthenticateTeamsRequest(ctx context.Context, activity schema.Activity, authHeader, appID string) error
}

// TeamsAuthHook validates inbound Microsoft Teams requests. Teams does not
// sign payloads; authenticity rides on the Bot Framework JWT (signature via
// the published JWKS, issuer, audience = bot app ID, serviceUrl claim), so
// the hook validates the raw request itself instead of declaring header
// requirements.
type TeamsAuthHook struct {
	Authenticator TeamsTokenAuthenticator
}

// AuthRequirements declares no header requirements: Teams authenticity is
// established by validating the Bot Framework JWT on the raw request.
func (TeamsAuthHook) AuthRequirements(ServiceValidationConfig) AuthRequirements {
	return AuthRequirements{}
}

// ValidateProviderRequest authenticates the inbound activity's Bot Framework
// JWT against the bot's Microsoft App ID.
func (h TeamsAuthHook) ValidateProviderRequest(ctx context.Context, input RequestValidationInput) error {
	appID := teamsAppID(input.Service)
	if appID == "" {
		return ErrMissingSecret("TEAMS_APP_ID")
	}
	authHeader := strings.TrimSpace(input.Headers.Get("Authorization"))
	if authHeader == "" {
		return ErrAuthFailed(input.Service.ID, "missing Bot Framework authorization header")
	}
	var activity schema.Activity
	if err := json.Unmarshal(input.Body, &activity); err != nil {
		return ErrAuthFailed(input.Service.ID, "invalid Bot Framework activity payload")
	}
	authenticator := h.Authenticator
	if authenticator == nil {
		authenticator = sharedTeamsAuthenticator()
	}
	if err := authenticator.AuthenticateTeamsRequest(ctx, activity, authHeader, appID); err != nil {
		return ErrAuthFailed(input.Service.ID, "invalid Bot Framework token")
	}
	return nil
}

// sdkTeamsAuthenticator adapts the Bot Framework SDK JWT validator. The
// validator checks signature against the Bot Framework OpenID JWKS, issuer,
// audience (app ID), token expiry, and that the serviceurl claim matches the
// activity's ServiceURL.
type sdkTeamsAuthenticator struct {
	validator auth.TokenValidator
}

func (a sdkTeamsAuthenticator) AuthenticateTeamsRequest(ctx context.Context, activity schema.Activity, authHeader, appID string) error {
	if _, err := a.validator.AuthenticateRequest(ctx, activity, authHeader, auth.SimpleCredentialProvider{AppID: appID}, ""); err != nil {
		return fmt.Errorf("bot framework token validation: %w", err)
	}
	return nil
}

var (
	teamsAuthenticatorOnce sync.Once
	teamsAuthenticator     TeamsTokenAuthenticator
)

// sharedTeamsAuthenticator reuses one SDK validator so its JWKS cache spans
// requests.
func sharedTeamsAuthenticator() TeamsTokenAuthenticator {
	teamsAuthenticatorOnce.Do(func() {
		teamsAuthenticator = sdkTeamsAuthenticator{validator: auth.NewJwtTokenValidator()}
	})
	return teamsAuthenticator
}

// teamsAppID resolves the bot's Microsoft App ID: explicit options first,
// then the service env binding, then the ambient environment.
func teamsAppID(service ServiceValidationConfig) string {
	lookup := core.ProfileSecretLookup(service.Profile)
	return firstConfigured(
		resolveConfiguredSecret(service.Options["app_id"], lookup),
		getenv(service.Options["app_id_env"]),
		serviceEnvLiteral(service.Env, "TEAMS_APP_ID"),
		resolveSecretName(serviceEnvSecretName(service.Env, "TEAMS_APP_ID"), lookup),
		getenv("TEAMS_APP_ID"),
	)
}
