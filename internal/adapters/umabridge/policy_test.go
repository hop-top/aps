package umabridge

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/uma/model"
	"hop.top/uma/policy"
	"hop.top/uma/ports"
)

func TestAuthorizerAdmitsOnlyTheBridgeActorsForTheirActions(t *testing.T) {
	cases := []struct {
		name      string
		principal model.Principal
		action    string
		allow     bool
	}{
		{"service configures channel", model.Principal{Tenant: tenant, Actor: actorService}, model.ActionChannelConfigure, true},
		{"service receives", model.Principal{Tenant: tenant, Actor: actorService}, model.ActionChannelReceive, true},
		{"service cannot reply", model.Principal{Tenant: tenant, Actor: actorService}, model.ActionMessageReply, false},
		{"service cannot export", model.Principal{Tenant: tenant, Actor: actorService}, model.ActionPrivacyExport, false},
		{"bot replies", model.Principal{Tenant: tenant, Actor: actorBot}, model.ActionMessageReply, true},
		{"bot polls", model.Principal{Tenant: tenant, Actor: actorBot}, model.ActionSubscriptionPoll, true},
		{"bot cannot configure", model.Principal{Tenant: tenant, Actor: actorBot}, model.ActionChannelConfigure, false},
		{"bot cannot redact", model.Principal{Tenant: tenant, Actor: actorBot}, model.ActionPrivacyRedact, false},
		{"bot cannot merge contacts", model.Principal{Tenant: tenant, Actor: actorBot}, model.ActionContactMerge, false},
		{"stranger", model.Principal{Tenant: tenant, Actor: "someone"}, model.ActionMessageReply, false},
		{"other tenant", model.Principal{Tenant: "other", Actor: actorBot}, model.ActionMessageReply, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := authorizer{}.Authorize(context.Background(), policy.Authorization{Principal: c.principal, Action: c.action})
			require.NoError(t, err)
			assert.Equal(t, c.allow, d.Allow)
			assert.NotEmpty(t, d.ReasonCode)
		})
	}
}

func req(purpose string, kinds ...ports.CredentialKind) ports.CredentialRequest {
	return ports.CredentialRequest{Purpose: purpose, Accept: kinds}
}

func TestCredentialsResolveSecretBindingsPerPurpose(t *testing.T) {
	purposes := map[string]string{"twilio.rest": "TOKEN", "twilio.webhook": "TOKEN", "meta.graph": "GRAPH"}
	store := func(name string) (string, bool) {
		v, ok := map[string]string{"STORED": "from-store"}[name]
		return v, ok
	}
	c := newCredentials(purposes, map[string]string{
		"TOKEN": "secret:STORED",
		"GRAPH": "literal-token",
	}, store)
	c.environ = func(name string) (string, bool) {
		v, ok := map[string]string{"ENVONLY": "from-env", "UNBOUND": "ambient"}[name]
		return v, ok
	}

	got, err := c.Resolve(context.Background(), req("twilio.rest", ports.KindSecret, ports.KindPassword))
	require.NoError(t, err)
	assert.Equal(t, ports.Secret{Bytes: []byte("from-store")}, got)

	got, err = c.Resolve(context.Background(), req("twilio.webhook", ports.KindSecret))
	require.NoError(t, err)
	assert.Equal(t, ports.Secret{Bytes: []byte("from-store")}, got)

	got, err = c.Resolve(context.Background(), req("meta.graph", ports.KindOAuth2, ports.KindSecret))
	require.NoError(t, err)
	assert.Equal(t, ports.Secret{Bytes: []byte("literal-token")}, got, "secret preferred when accepted")

	got, err = c.Resolve(context.Background(), req("meta.graph", ports.KindOAuth2))
	require.NoError(t, err)
	assert.Equal(t, ports.OAuth2{AccessToken: "literal-token"}, got)

	c.env["TOKEN"] = "secret:ENVONLY"
	got, err = c.Resolve(context.Background(), req("twilio.rest", ports.KindSecret))
	require.NoError(t, err)
	assert.Equal(t, ports.Secret{Bytes: []byte("from-env")}, got, "reference falls through to the environment")
}

func TestCredentialsFailClosed(t *testing.T) {
	c := newCredentials(map[string]string{"twilio.rest": "TOKEN"}, map[string]string{"TOKEN": "secret:GONE"}, func(string) (string, bool) { return "", false })
	c.environ = func(name string) (string, bool) { return "ambient-" + name, name != "GONE" && name != "TOKEN" }

	_, err := c.Resolve(context.Background(), req("twilio.rest", ports.KindSecret))
	var merr *model.Error
	require.True(t, errors.As(err, &merr))
	assert.Equal(t, model.Unauthorized, merr.Code, "a reference that resolves to nothing never falls back to the binding key")
	assert.NotContains(t, err.Error(), "ambient")

	_, err = c.Resolve(context.Background(), req("unknown.purpose", ports.KindSecret))
	require.True(t, errors.As(err, &merr))
	assert.Equal(t, model.Unsupported, merr.Code)

	c.env["TOKEN"] = "literal"
	_, err = c.Resolve(context.Background(), req("twilio.rest", ports.KindClientCert))
	require.True(t, errors.As(err, &merr))
	assert.Equal(t, model.Unsupported, merr.Code, "a kind outside Accept is unsupported")
}

func TestCredentialsUnboundBindingReadsTheEnvironment(t *testing.T) {
	c := newCredentials(map[string]string{"twilio.rest": "TOKEN"}, nil, nil)
	c.environ = func(name string) (string, bool) { return "env-" + name, name == "TOKEN" }
	got, err := c.Resolve(context.Background(), req("twilio.rest", ports.KindSecret))
	require.NoError(t, err)
	assert.Equal(t, ports.Secret{Bytes: []byte("env-TOKEN")}, got)
}

func TestCredentialsNeverRenderTheSecret(t *testing.T) {
	c := newCredentials(map[string]string{"twilio.rest": "TOKEN"}, map[string]string{"TOKEN": "hunter2"}, nil)
	got, err := c.Resolve(context.Background(), req("twilio.rest", ports.KindSecret))
	require.NoError(t, err)
	assert.NotContains(t, got.(ports.Secret).String(), "hunter2")
}
