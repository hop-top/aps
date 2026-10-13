package core

import (
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The uma adapters that take generic webhooks: a relay the operator runs posts
// to them, authenticated by uma's webhookauth.
const (
	UmaAdapterRelay    = "relay.messages"
	UmaAdapterHTTPMail = "http.mail"
)

// EnvRelayCallbackToken is the binding of the credential relay.messages
// presents to the gateway's callback.
const EnvRelayCallbackToken = "RELAY_CALLBACK_TOKEN"

// The mediums a relay.messages service relays, which are also the aps
// platforms of its messages.
const (
	UmaMediumSMS      = "sms"
	UmaMediumWhatsApp = "whatsapp"
)

// uma's webhook auth schemes, as UmaWebhookAuth.Scheme names them.
const (
	UmaSchemeBearer  = "bearer"
	UmaSchemeToken   = "token"
	UmaSchemeHMAC    = "hmac"
	UmaSchemeEd25519 = "ed25519"
)

// Options of relay.messages and http.mail services.
const (
	OptionMedium            = "medium"
	OptionCallbackURL       = "callback_url"
	OptionCallbackHeader    = "callback_header"
	OptionRecipients        = "recipients"
	OptionFormats           = "formats"
	OptionRecipientHeader   = "recipient_header"
	OptionTrustedAuthservID = "trusted_authserv_id"
	OptionAuthResultsField  = "authentication_results_field"
)

// Generic webhook auth options beyond the scheme and secrets, read by the
// native validator and, mapped onto uma's webhookauth, by relay.messages and
// http.mail. The auth_payload, auth_allow_unsigned and auth_min_secret_bytes
// options exist for uma only: they state what aps's native layout leaves
// implicit.
const (
	OptionAuthHeader         = "auth_header"
	OptionTimestampHeader    = "timestamp_header"
	OptionRequireTimestamp   = "require_timestamp"
	OptionTimestampTolerance = "timestamp_tolerance"
	OptionReplayIDHeader     = "replay_id_header"
	OptionRequireReplayCheck = "require_replay_check"
	OptionAuthPayload        = "auth_payload"
	OptionAuthAllowUnsigned  = "auth_allow_unsigned"
	OptionAuthMinSecretBytes = "auth_min_secret_bytes" //nolint:gosec // option key, not a credential
)

// UmaWebhookAuth is the request authentication of a uma service that takes
// generic webhooks: uma's webhookauth.Config as aps derives it from the
// service's generic webhook auth options, so that a sender aps verifies today
// verifies unchanged. It holds no secret: SecretBinding names the env binding
// the key is resolved from.
type UmaWebhookAuth struct {
	// Scheme is uma's name: bearer, token, hmac or ed25519.
	Scheme         string
	Header         string
	Encoding       string
	Prefix         string
	PrefixOptional bool
	Payload        string
	KeyEncoding    string
	// MinSecretBytes is zero for uma's default (16).
	MinSecretBytes int
	// TimestampHeader turns the freshness check on.
	TimestampHeader    string
	TimestampFormat    string
	TimestampTolerance time.Duration
	// TimestampUnsigned and ReplayUnsigned are the operator's consent to a
	// timestamp or replay id the signature does not cover.
	TimestampUnsigned bool
	// ReplayHeader turns replay rejection by that header on.
	ReplayHeader   string
	ReplayUnsigned bool
	// SecretBinding is the env binding of the shared token, HMAC key or
	// Ed25519 public key.
	SecretBinding string
}

// Header and layout defaults of aps's native generic webhook auth
// (internal/core/messenger/validation.go).
const (
	headerAPSToken       = "X-APS-Token" //nolint:gosec // a header name, not a credential
	headerAPSSignature   = "X-APS-Signature"
	headerAPSTimestamp   = "X-APS-Timestamp"
	headerAPSDelivery    = "X-APS-Delivery-ID"
	headerEd25519        = "X-Signature-Ed25519"
	headerEd25519Stamp   = "X-Signature-Timestamp"
	defaultTolerance     = 5 * time.Minute
	maxUmaTolerance      = 24 * time.Hour
	umaTimestampFormat   = "unix_or_rfc3339"
	umaAuthAdviceSuffix  = " (uma has no unauthenticated mode)"
	consentAdvice        = "set --option " + OptionAuthAllowUnsigned + "=true to accept it unsigned, as aps did"
	signedPayloadAdvice  = "or sign it and set --option " + OptionAuthPayload + "="
	payloadPlaceholders  = "{body} once, and {timestamp} or {id} only when a timestamp or replay header is set"
	callbackHostsSetting = "uma.relay.callback_hosts"
	encodingHex          = "hex"
	payloadBody          = "{body}"
	replyNone            = "none"
	schemeHTTP           = "http"
)

// UmaWebhookAuthFor maps a service's generic webhook auth options onto uma's
// webhook authentication, with aps's header names, encodings and signed
// layout. It returns the issues that keep the mapping from being used: a
// missing scheme or binding, an option uma refuses, or a timestamp or replay
// id the signature does not cover and the operator did not consent to.
func UmaWebhookAuthFor(options map[string]string) (UmaWebhookAuth, []string) {
	var (
		a      UmaWebhookAuth
		issues []string
	)
	opt := func(key string) string { return strings.TrimSpace(options[key]) }
	scheme := ApsAuthScheme(options)
	tokenEnv, sigEnv := opt(OptionAuthTokenEnv), opt(OptionSignatureSecretEnv)
	var tsHeader string
	switch scheme {
	case "":
		return a, []string{"requires webhook auth: set --auth-scheme with --auth-token-env or --signature-secret-env" +
			umaAuthAdviceSuffix}
	case AuthSchemeBearer:
		a.Scheme, a.Header, a.SecretBinding = UmaSchemeBearer, "Authorization", tokenEnv
	case AuthSchemeToken:
		a.Scheme, a.Header, a.SecretBinding = UmaSchemeToken, headerAPSToken, tokenEnv
	case AuthSchemeHMACSHA256:
		a.Scheme, a.Header, a.SecretBinding = UmaSchemeHMAC, headerAPSSignature, sigEnv
		a.Encoding, a.Prefix, a.PrefixOptional, a.Payload = encodingHex, "sha256=", true, payloadBody
	case AuthSchemeEd25519:
		a.Scheme, a.Header, a.SecretBinding = UmaSchemeEd25519, headerEd25519, sigEnv
		a.Encoding, a.KeyEncoding, a.Payload = encodingHex, encodingHex, "{timestamp}{body}"
		tsHeader = headerEd25519Stamp
	case AuthSchemeSlack:
		return a, []string{"auth_scheme " + AuthSchemeSlack + " is Slack's; use bearer, token, hmac-sha256 or ed25519"}
	default:
		// validateGenericWebhookAuth reports the unknown scheme.
		return a, nil
	}
	signed := a.Scheme == UmaSchemeHMAC || a.Scheme == UmaSchemeEd25519
	if a.SecretBinding == "" {
		flag := "--auth-token-env"
		if signed {
			flag = "--signature-secret-env"
		}
		issues = append(issues, "reads its webhook secret from a binding only: set "+flag+" NAME and --env NAME=secret:NAME")
	}
	if h := opt(OptionAuthHeader); h != "" {
		a.Header = h
	}
	if h := opt(OptionTimestampHeader); h != "" {
		tsHeader = h
	} else if tsHeader == "" && truthyServiceOption(options[OptionRequireTimestamp]) {
		tsHeader = headerAPSTimestamp
	}
	if tsHeader != "" {
		a.TimestampHeader, a.TimestampFormat, a.TimestampTolerance = tsHeader, umaTimestampFormat, defaultTolerance
		if raw := opt(OptionTimestampTolerance); raw != "" {
			d, err := time.ParseDuration(raw)
			switch {
			case err != nil || d <= 0:
				issues = append(issues, "option "+OptionTimestampTolerance+" must be a positive duration such as 5m")
			case d > maxUmaTolerance:
				issues = append(issues, "option "+OptionTimestampTolerance+" must be at most 24h under uma")
			default:
				a.TimestampTolerance = d
			}
		}
	}
	if h := opt(OptionReplayIDHeader); h != "" {
		a.ReplayHeader = h
	} else if truthyServiceOption(options[OptionRequireReplayCheck]) {
		a.ReplayHeader = headerAPSDelivery
	}
	if p := opt(OptionAuthPayload); p != "" {
		if !signed {
			issues = append(issues, "option "+OptionAuthPayload+" applies to hmac-sha256 and ed25519, which sign; "+a.Scheme+" signs nothing")
		} else {
			a.Payload = p
		}
	}
	uses := map[string]bool{}
	if signed {
		var err error
		if uses, err = payloadPlaceholdersOf(a.Payload, a.TimestampHeader != "", a.ReplayHeader != ""); err != nil {
			issues = append(issues, "option "+OptionAuthPayload+" "+err.Error()+": it takes "+payloadPlaceholders)
		}
	}
	consent := truthyServiceOption(options[OptionAuthAllowUnsigned])
	unsigned := func(what, header, placeholder string) bool {
		if consent {
			return true
		}
		advice := consentAdvice
		if signed {
			advice += ", " + signedPayloadAdvice + placeholder
		}
		issues = append(issues, fmt.Sprintf("the %s header %s is not covered by the %s signature; %s",
			what, header, a.Scheme, advice))
		return false
	}
	if a.TimestampHeader != "" && !uses["timestamp"] {
		a.TimestampUnsigned = unsigned("timestamp", a.TimestampHeader, "{timestamp}.{body}")
	}
	if a.ReplayHeader != "" && !uses["id"] {
		a.ReplayUnsigned = unsigned("replay id", a.ReplayHeader, "{id}.{body}")
	}
	if raw := opt(OptionAuthMinSecretBytes); raw != "" {
		n, err := strconv.Atoi(raw)
		switch {
		case a.Scheme == UmaSchemeEd25519:
			issues = append(issues, "option "+OptionAuthMinSecretBytes+" does not apply to ed25519: its keys are 32 bytes")
		case err != nil || n < 1:
			issues = append(issues, "option "+OptionAuthMinSecretBytes+" must be a positive number of bytes")
		default:
			a.MinSecretBytes = n
		}
	}
	return a, issues
}

// payloadPlaceholdersOf parses a signed-payload template as uma does: literal
// text, {body} exactly once, {timestamp} and {id} when enabled, "{{" and
// "}}" as literal braces. It reports the placeholders used.
func payloadPlaceholdersOf(template string, timestamp, id bool) (map[string]bool, error) {
	enabled := map[string]bool{"body": true, "timestamp": timestamp, "id": id}
	uses := map[string]bool{}
	body := 0
	for i := 0; i < len(template); i++ {
		switch template[i] {
		case '{':
			if i+1 < len(template) && template[i+1] == '{' {
				i++
				continue
			}
			end := strings.IndexByte(template[i:], '}')
			if end < 0 {
				return nil, fmt.Errorf("has an unclosed placeholder")
			}
			name := template[i+1 : i+end]
			if !enabled[name] {
				return nil, fmt.Errorf("names {%s}, which is not available", name)
			}
			if name == "body" {
				body++
			}
			uses[name] = true
			i += end
		case '}':
			if i+1 < len(template) && template[i+1] == '}' {
				i++
				continue
			}
			return nil, fmt.Errorf("has an unmatched }")
		}
	}
	if body != 1 {
		return nil, fmt.Errorf("must contain {body} exactly once")
	}
	return uses, nil
}

// ApsAuthScheme is the aps auth_scheme a service's generic webhook auth
// options name or imply: bearer from a token option, hmac-sha256 from a
// signature option, as the native validator infers it.
func ApsAuthScheme(options map[string]string) string {
	opt := func(key string) string { return strings.TrimSpace(options[key]) }
	if scheme := strings.ToLower(opt(OptionAuthScheme)); scheme != "" {
		return scheme
	}
	switch {
	case opt(OptionAuthTokenEnv) != "" || opt(OptionAuthToken) != "":
		return AuthSchemeBearer
	case opt(OptionSignatureSecretEnv) != "" || opt(OptionSignatureSecret) != "":
		return AuthSchemeHMACSHA256
	}
	return ""
}

// validateUmaWebhookAuth checks a generic-webhook uma service's auth and its
// secret binding.
func validateUmaWebhookAuth(key string, service *ServiceConfig, result *ServiceValidationResult) {
	auth, issues := UmaWebhookAuthFor(service.Options)
	for _, issue := range issues {
		result.Issues = append(result.Issues, "uma "+key+" "+issue)
	}
	if auth.SecretBinding != "" {
		requireEnv(service.Env, result, auth.SecretBinding)
	}
}

// validateUmaRelay checks a relay.messages service.
func validateUmaRelay(service *ServiceConfig, result *ServiceValidationResult) {
	o := service.Options
	switch strings.TrimSpace(o[OptionMedium]) {
	case UmaMediumSMS, UmaMediumWhatsApp:
	default:
		result.Issues = append(result.Issues, "uma relay.messages requires option medium (sms or whatsapp)")
	}
	if strings.TrimSpace(o["from"]) == "" {
		result.Issues = append(result.Issues, "uma relay.messages requires option from (the number the gateway reports as to)")
	}
	validateUmaWebhookAuth(UmaAdapterRelay, service, result)
	callback := strings.TrimSpace(o[OptionCallbackURL])
	replies := !strings.EqualFold(strings.TrimSpace(o[optionReply]), replyNone)
	switch {
	case callback == "" && replies:
		result.Issues = append(result.Issues, "uma relay.messages requires option callback_url to reply (replies leave by "+
			"the gateway's callback), or --reply none to receive only")
	case callback != "":
		if issue := umaCallbackIssue(callback, LoadUmaHostConfig().Relay); issue != "" {
			result.Issues = append(result.Issues, "uma relay.messages option callback_url "+issue)
		}
		requireEnv(service.Env, result, EnvRelayCallbackToken)
	}
	if h := strings.TrimSpace(o[OptionCallbackHeader]); h != "" && !callbackHeaderName(h) {
		result.Issues = append(result.Issues, "uma relay.messages option callback_header must be a header name the "+
			"gateway reads its credential from, not one the request sets itself")
	}
}

// umaCallbackIssue judges a callback URL as uma's relay host does, against
// the host settings: empty when allowed.
func umaCallbackIssue(raw string, host UmaRelayHostConfig) string {
	u, err := url.Parse(raw)
	switch {
	case err != nil || u.Host == "" || u.Opaque != "":
		return "must be an absolute https URL"
	case u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		return "must carry no credentials, query or fragment"
	case u.Scheme == schemeHTTP && !host.AllowHTTP:
		return "must be https (http needs uma.relay.allow_http, for local development only)"
	case u.Scheme != "https" && u.Scheme != schemeHTTP:
		return "must be an absolute https URL"
	}
	for _, h := range host.CallbackHosts {
		if strings.EqualFold(strings.TrimSpace(h), u.Host) {
			return ""
		}
	}
	return "names host " + u.Host + ", which is not in the global setting " + callbackHostsSetting +
		" (no host is allowed by default)"
}

// reservedCallbackHeaders are set by uma's relay itself or owned by the
// transport.
var reservedCallbackHeaders = map[string]bool{
	"host": true, "content-length": true, "content-type": true, "transfer-encoding": true, "connection": true,
	"idempotency-key": true, "upgrade": true, "te": true, "trailer": true, "accept": true,
}

func callbackHeaderName(name string) bool {
	if len(name) > 128 || reservedCallbackHeaders[strings.ToLower(name)] {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !tokenChar(name[i]) {
			return false
		}
	}
	return true
}

// tokenChar reports whether c may appear in an HTTP field name (RFC 9110
// 5.6.2 tchar).
func tokenChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}

// validateUmaHTTPMail checks an http.mail service. Replies are not an issue:
// the adapter receives only, so the bridge refuses them and `service add`
// warns.
func validateUmaHTTPMail(service *ServiceConfig, result *ServiceValidationResult) {
	o := service.Options
	recipients := splitServiceList(o[OptionRecipients])
	if len(recipients) == 0 {
		result.Issues = append(result.Issues, "uma http.mail requires option recipients (the addresses it accepts mail for)")
	}
	seen := map[string]bool{}
	for _, r := range recipients {
		norm, ok := bareAddress(r)
		switch {
		case !ok:
			result.Issues = append(result.Issues, fmt.Sprintf("uma http.mail option recipients: %q is not a bare address", r))
		case seen[norm]:
			result.Issues = append(result.Issues, fmt.Sprintf("uma http.mail option recipients: %q is listed twice", r))
		}
		seen[norm] = true
	}
	rfc822 := false
	for _, f := range splitServiceList(o[OptionFormats]) {
		switch f {
		case "json":
		case "rfc822":
			rfc822 = true
		default:
			result.Issues = append(result.Issues, "uma http.mail option formats takes json and rfc822")
		}
	}
	if h := strings.TrimSpace(o[OptionRecipientHeader]); h != "" && !rfc822 {
		result.Issues = append(result.Issues, "uma http.mail option recipient_header applies to the rfc822 format; add it to option formats")
	}
	trusted := strings.TrimSpace(o[OptionTrustedAuthservID])
	if trusted != "" && strings.ContainsAny(trusted, " \t;\"()\\<>") {
		result.Issues = append(result.Issues, "uma http.mail option trusted_authserv_id must be one token")
	}
	if strings.TrimSpace(o[OptionAuthResultsField]) != "" && trusted == "" {
		result.Issues = append(result.Issues, "uma http.mail option authentication_results_field needs trusted_authserv_id")
	}
	validateUmaWebhookAuth(UmaAdapterHTTPMail, service, result)
	if strings.TrimSpace(o[OptionAllowedSenders]) == "" {
		result.Warnings = append(result.Warnings, "uma http.mail service has no allowed senders; any sender can route inbound mail")
	}
}

// UmaMailRecipients lists an http.mail service's recipient addresses.
func UmaMailRecipients(options map[string]string) []string {
	return splitServiceList(options[OptionRecipients])
}

// UmaMailFormats lists an http.mail service's body formats; empty is json.
func UmaMailFormats(options map[string]string) []string {
	if formats := splitServiceList(options[OptionFormats]); len(formats) > 0 {
		return formats
	}
	return []string{"json"}
}

// bareAddress reports whether s is a bare mailbox address and returns it with
// the domain lowercased, the comparison uma makes.
func bareAddress(s string) (string, bool) {
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Name != "" || addr.Address != s {
		return "", false
	}
	at := strings.LastIndexByte(s, '@')
	return s[:at] + "@" + strings.ToLower(s[at+1:]), true
}

func splitServiceList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
