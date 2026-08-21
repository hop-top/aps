package messenger

import (
	"testing"

	"hop.top/aps/internal/core"
)

// An unresolvable "secret:NAME" must not silently fall back to the process
// environment under the binding key. Before this guard, a service configured
// with secret:twilio_sid whose secret was missing would authenticate with
// whatever TWILIO_ACCOUNT_SID happened to be set in the ambient environment.
func TestResolveServiceEnvDoesNotFallBackToBindingKey(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "WRONG-ambient-account")

	service := &core.ServiceConfig{
		Profile: "",
		Env:     map[string]string{"TWILIO_ACCOUNT_SID": "secret:twilio_sid"},
	}

	if got := resolveServiceEnv(service, "TWILIO_ACCOUNT_SID"); got != "" {
		t.Fatalf("resolveServiceEnv returned %q for an unresolved secret ref; want empty", got)
	}
}

func TestResolveServiceEnvResolvesSecretRefFromEnv(t *testing.T) {
	t.Setenv("MY_SECRET_NAME", "real-token")

	service := &core.ServiceConfig{Env: map[string]string{"SLACK_BOT_TOKEN": "secret:MY_SECRET_NAME"}}

	if got := resolveServiceEnv(service, "SLACK_BOT_TOKEN"); got != "real-token" {
		t.Fatalf("resolveServiceEnv = %q, want real-token", got)
	}
}

func TestResolveServiceEnvLiteralWins(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "ambient-token")

	service := &core.ServiceConfig{Env: map[string]string{"SLACK_BOT_TOKEN": "xoxb-literal"}}

	if got := resolveServiceEnv(service, "SLACK_BOT_TOKEN"); got != "xoxb-literal" {
		t.Fatalf("resolveServiceEnv = %q, want xoxb-literal", got)
	}
}

// With no binding at all, the ambient environment remains the source. This is
// the pre-existing contract for services that never declared the key, and is
// deliberately preserved.
func TestResolveServiceEnvUnboundKeyUsesProcessEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "ambient-token")

	if got := resolveServiceEnv(&core.ServiceConfig{Env: map[string]string{}}, "TELEGRAM_BOT_TOKEN"); got != "ambient-token" {
		t.Fatalf("empty env map: got %q, want ambient-token", got)
	}
	if got := resolveServiceEnv(nil, "TELEGRAM_BOT_TOKEN"); got != "ambient-token" {
		t.Fatalf("nil service: got %q, want ambient-token", got)
	}
}

// Reproduces the published example at docs/user/messengers.md, which uses
// lowercase store-style names. These are secret-store keys, not env vars.
func TestResolveServiceEnvDocumentedLowercaseExample(t *testing.T) {
	service := &core.ServiceConfig{Env: map[string]string{
		"TWILIO_ACCOUNT_SID": "secret:twilio_sid",
		"TWILIO_AUTH_TOKEN":  "secret:twilio_token",
	}}

	t.Setenv("twilio_sid", "AC-real")
	t.Setenv("twilio_token", "tok-real")

	if got := resolveServiceEnv(service, "TWILIO_ACCOUNT_SID"); got != "AC-real" {
		t.Fatalf("sid = %q, want AC-real", got)
	}
	if got := resolveServiceEnv(service, "TWILIO_AUTH_TOKEN"); got != "tok-real" {
		t.Fatalf("token = %q, want tok-real", got)
	}
}
