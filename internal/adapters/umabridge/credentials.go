package umabridge

import (
	"context"
	"fmt"

	"hop.top/aps/internal/core"
	"hop.top/uma/model"
	"hop.top/uma/ports"
)

// credentials is the engine's ports.Credentials over a service's secret
// bindings. A uma credential purpose maps to one env binding of the service;
// the binding value is a literal or a "secret:NAME" reference, resolved on
// every request against the profile secret store and then the environment
// (core.ResolveSecretValue), never cached here and never from the binding
// key's own name when the reference resolves to nothing.
type credentials struct {
	purposes map[string]string // uma purpose -> env binding name
	env      map[string]string // the service's env bindings
	store    core.SecretLookup
	environ  core.SecretLookup
}

func newCredentials(purposes, env map[string]string, store core.SecretLookup) *credentials {
	return &credentials{purposes: purposes, env: env, store: store, environ: core.EnvSecretLookup}
}

var _ ports.Credentials = (*credentials)(nil)

// value resolves a binding. An unbound name falls back to the process
// environment, as the aps-native providers do; a bound reference that
// resolves to nothing stays empty.
func (c *credentials) value(binding string) (string, bool) {
	bound, ok := c.env[binding]
	if !ok || bound == "" {
		return c.environ(binding)
	}
	return core.ResolveSecretValue(bound, c.store, c.environ)
}

func (c *credentials) Resolve(_ context.Context, req ports.CredentialRequest) (ports.Credential, error) {
	binding, ok := c.purposes[req.Purpose]
	if !ok {
		return nil, model.NewError(model.Unsupported, "no credential for purpose")
	}
	secret, found := c.value(binding)
	if !found {
		return nil, model.NewError(model.Unauthorized, "credential binding "+binding+" is not available")
	}
	switch {
	case req.Accepts(ports.KindSecret):
		return ports.Secret{Bytes: []byte(secret)}, nil
	case req.Accepts(ports.KindOAuth2):
		return ports.OAuth2{AccessToken: secret}, nil
	default:
		return nil, model.NewError(model.Unsupported, "credential kind not available for purpose")
	}
}

// defaultMinSecretBytes is uma's webhookauth default: the shortest secret it
// accepts for bearer, token and HMAC unless the service lowers it.
const defaultMinSecretBytes = 16

// checkWebhookSecret is a readiness check of a generic-webhook service's
// secret, made when the bridge starts. uma resolves the secret on every
// request and refuses one that is missing or too short with a 503 and no
// reason; the bridge refuses to start instead, saying which binding and
// which option to look at. The secret itself is never part of the error.
// Ed25519 keys are left to uma, which checks their decoded size.
func checkWebhookSecret(options map[string]string, binding func(string) (string, bool)) error {
	auth, _ := core.UmaWebhookAuthFor(options)
	switch auth.Scheme {
	case core.UmaSchemeBearer, core.UmaSchemeToken, core.UmaSchemeHMAC:
	default:
		return nil
	}
	secret, ok := binding(auth.SecretBinding)
	if !ok || secret == "" {
		return fmt.Errorf("webhook secret binding %s resolves to nothing", auth.SecretBinding)
	}
	minimum := auth.MinSecretBytes
	if minimum == 0 {
		minimum = defaultMinSecretBytes
	}
	if len(secret) < minimum {
		return fmt.Errorf("webhook secret bound to %s is %d bytes; uma requires at least %d. Use a longer secret, "+
			"or set --option %s=%d deliberately", auth.SecretBinding, len(secret), minimum, core.OptionAuthMinSecretBytes, len(secret))
	}
	return nil
}
