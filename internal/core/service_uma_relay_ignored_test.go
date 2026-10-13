package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// genericAuthOptions are the generic webhook auth options relay.messages and
// http.mail read.
var genericAuthOptions = map[string]string{
	OptionAuthScheme: "hmac-sha256", OptionSignatureSecretEnv: "SIG", OptionAuthTokenEnv: "TOK",
	OptionAuthHeader: "X-Sig", OptionTimestampHeader: "X-Ts", OptionRequireTimestamp: "true",
	OptionTimestampTolerance: "2m", OptionReplayIDHeader: "X-Id", OptionRequireReplayCheck: "true",
	OptionAuthPayload: "{body}", OptionAuthAllowUnsigned: "true", OptionAuthMinSecretBytes: "8",
}

func TestUmaIgnoredOptions_GenericAuthIsReadByRelayAndMail(t *testing.T) {
	for _, base := range []func() *ServiceConfig{umaRelayService, umaMailService} {
		service := base()
		for key, value := range genericAuthOptions {
			service.Options[key] = value
		}
		assert.Empty(t, ignoredKeys(service), service.Options[OptionUmaAdapter])
	}
}

func TestUmaIgnoredOptions_GenericAuthIsIgnoredByProviderAdapters(t *testing.T) {
	for _, base := range []func() *ServiceConfig{umaTwilioSMSService, umaMetaWhatsAppService} {
		for key, value := range genericAuthOptions {
			service := base()
			service.Options[key] = value
			assert.Contains(t, ignoredKeys(service), key, "%s on %s", key, service.Options[OptionUmaAdapter])
		}
	}
}

func TestUmaIgnoredOptions_LiteralSecretsAreNeverRead(t *testing.T) {
	for _, base := range []func() *ServiceConfig{umaRelayService, umaMailService} {
		service := base()
		service.Options[OptionAuthToken] = "tok"
		service.Options[OptionSignatureSecret] = "sec"
		assert.ElementsMatch(t, []string{OptionAuthToken, OptionSignatureSecret}, ignoredKeys(service),
			service.Options[OptionUmaAdapter])
	}
}

func TestUmaIgnoredOptions_AdapterSpecificOptions(t *testing.T) {
	relayOnly := map[string]string{
		OptionMedium: "sms", OptionCallbackURL: "https://g.example.com/r", OptionCallbackHeader: "X-Key",
	}
	mailOnly := map[string]string{
		OptionRecipients: "a@example.com", OptionFormats: "json", OptionRecipientHeader: "X-To",
		OptionTrustedAuthservID: "mx.example.com", OptionAuthResultsField: "ar",
	}
	for key, value := range relayOnly {
		for _, base := range []func() *ServiceConfig{umaTwilioSMSService, umaMetaWhatsAppService, umaMailService} {
			service := base()
			service.Options[key] = value
			assert.Contains(t, ignoredKeys(service), key, "%s on %s", key, service.Options[OptionUmaAdapter])
		}
		service := umaRelayService()
		service.Options[key] = value
		assert.NotContains(t, ignoredKeys(service), key)
	}
	for key, value := range mailOnly {
		for _, base := range []func() *ServiceConfig{umaTwilioSMSService, umaMetaWhatsAppService, umaRelayService} {
			service := base()
			service.Options[key] = value
			assert.Contains(t, ignoredKeys(service), key, "%s on %s", key, service.Options[OptionUmaAdapter])
		}
		service := umaMailService()
		service.Options[key] = value
		assert.NotContains(t, ignoredKeys(service), key)
	}

	relay := umaRelayService()
	relay.Options["webhook_url"] = "https://hooks.example.com/r"
	mail := umaMailService()
	mail.Options["webhook_url"] = "https://hooks.example.com/m"
	mail.Options["from"] = "+15558675310"
	assert.Equal(t, []string{"webhook_url"}, ignoredKeys(relay), "relay signs no URL")
	assert.Equal(t, []string{"from", "webhook_url"}, ignoredKeys(mail), "http.mail reads recipients, not from")
}

func TestUmaIgnoredOptions_MailReplyIsDisabled(t *testing.T) {
	mail := umaMailService()
	assert.Empty(t, UmaIgnoredOptions(mail), "reply none produces nothing to refuse")

	for _, mode := range []string{"text", "auto"} {
		mail.Options["reply"] = mode
		hits := UmaIgnoredOptions(mail)
		require.Len(t, hits, 1, mode)
		assert.Equal(t, "reply", hits[0].Option)
		assert.Equal(t, "reply", hits[0].Flag)
		assert.False(t, hits[0].Default)
		assert.Contains(t, hits[0].Advice, "receives only")
		assert.Contains(t, hits[0].Advice, "--reply none")
	}

	delete(mail.Options, "reply")
	hits := UmaIgnoredOptions(mail)
	require.Len(t, hits, 1, "an unset reply mode means text")
	assert.True(t, hits[0].Default)
	assert.Equal(t, "text", hits[0].Value)

	relay := umaRelayService()
	delete(relay.Options, "reply")
	assert.Empty(t, UmaIgnoredOptions(relay), "relay replies by callback")
}
