package core

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// umaPlatformEmail is the aps platform of http.mail messages.
const umaPlatformEmail = "email"

// optionReply is the reply mode option, set by `service add --reply`.
const optionReply = "reply"

// AdapterUma is the message adapter name of services whose provider I/O is
// hosted by an in-process uma engine instead of an aps-native provider.
const AdapterUma = "uma"

// OptionUmaAdapter names the uma adapter a uma-bridged message service runs
// (a key of UmaAdapterCatalogue).
const OptionUmaAdapter = "uma_adapter"

// Env binding names read by the uma bridge. They match the aps-native
// providers' names, so an operator moving a service keeps its bindings.
const (
	EnvTwilioAccountSID = "TWILIO_ACCOUNT_SID"
	EnvTwilioAuthToken  = "TWILIO_AUTH_TOKEN" //nolint:gosec // binding name, not a credential
	EnvWhatsAppToken    = "WHATSAPP_ACCESS_TOKEN"
	EnvWhatsAppSecret   = "WHATSAPP_APP_SECRET" //nolint:gosec // binding name, not a credential
	EnvWhatsAppVerify   = "WHATSAPP_VERIFY_TOKEN"
)

// UmaAdapterSpec is the static contract of one uma adapter as a message
// service: the env bindings and options it cannot run without. It lives here,
// not in the bridge package, so `service add` validates a uma service without
// linking uma.
type UmaAdapterSpec struct {
	// Key is the uma adapter type key.
	Key string
	// Platform is the aps platform name its messages carry.
	Platform string
	// RequiredEnv lists env bindings that must be present.
	RequiredEnv []string
	// OptionalEnv lists env bindings that are used when present.
	OptionalEnv []string
	// ReceiveOnly marks an adapter with no send path: an action's reply on
	// its service is refused, never sent.
	ReceiveOnly bool
	// GenericWebhook marks an adapter that takes webhooks from a relay the
	// operator runs, authenticated by the generic webhook auth options
	// (UmaWebhookAuthFor) instead of a provider signature.
	GenericWebhook bool
	// platform derives Platform from the options, when the adapter serves
	// more than one.
	platform func(options map[string]string) string
	// validate checks the adapter's options and bindings.
	validate func(service *ServiceConfig, result *ServiceValidationResult)
}

// PlatformFor is the aps platform the messages of a service of this adapter
// carry.
func (s UmaAdapterSpec) PlatformFor(options map[string]string) string {
	if s.platform != nil {
		return s.platform(options)
	}
	return s.Platform
}

var umaAdapterSpecs = map[string]UmaAdapterSpec{
	"twilio.sms": {
		Key:         "twilio.sms",
		Platform:    "sms",
		RequiredEnv: []string{EnvTwilioAccountSID, EnvTwilioAuthToken},
		validate: func(svc *ServiceConfig, r *ServiceValidationResult) {
			validateUmaTwilioOptions(svc.Options, r, "twilio.sms")
		},
	},
	"twilio.whatsapp": {
		Key:         "twilio.whatsapp",
		Platform:    "whatsapp",
		RequiredEnv: []string{EnvTwilioAccountSID, EnvTwilioAuthToken},
		validate: func(svc *ServiceConfig, r *ServiceValidationResult) {
			validateUmaTwilioOptions(svc.Options, r, "twilio.whatsapp")
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(svc.Options["from"])), "whatsapp:") {
				r.Issues = append(r.Issues, "uma twilio.whatsapp option from is configured without the whatsapp: prefix")
			}
		},
	},
	"meta.whatsapp": {
		Key:         "meta.whatsapp",
		Platform:    "whatsapp",
		RequiredEnv: []string{EnvWhatsAppToken, EnvWhatsAppSecret},
		OptionalEnv: []string{EnvWhatsAppVerify},
		validate: func(svc *ServiceConfig, r *ServiceValidationResult) {
			if !looksNumericID(svc.Options["phone_number_id"]) {
				r.Issues = append(r.Issues, "uma meta.whatsapp requires numeric option phone_number_id")
			}
		},
	},
	UmaAdapterRelay: {
		Key:            UmaAdapterRelay,
		GenericWebhook: true,
		platform: func(o map[string]string) string {
			if strings.TrimSpace(o[OptionMedium]) == UmaMediumWhatsApp {
				return UmaMediumWhatsApp
			}
			return UmaMediumSMS
		},
		validate: validateUmaRelay,
	},
	UmaAdapterHTTPMail: {
		Key:            UmaAdapterHTTPMail,
		Platform:       umaPlatformEmail,
		ReceiveOnly:    true,
		GenericWebhook: true,
		validate:       validateUmaHTTPMail,
	},
}

// UmaAdapterCatalogue returns the supported uma adapter keys, sorted.
func UmaAdapterCatalogue() []string {
	out := make([]string, 0, len(umaAdapterSpecs))
	for key := range umaAdapterSpecs {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// UmaAdapterSpecFor returns the contract of a uma adapter key.
func UmaAdapterSpecFor(key string) (UmaAdapterSpec, bool) {
	spec, ok := umaAdapterSpecs[strings.TrimSpace(key)]
	return spec, ok
}

// UmaAdapterSpecForService returns the contract of a uma service's adapter;
// false for any other service or an unknown adapter.
func UmaAdapterSpecForService(service *ServiceConfig) (UmaAdapterSpec, bool) {
	if service == nil || service.Adapter != AdapterUma {
		return UmaAdapterSpec{}, false
	}
	return UmaAdapterSpecFor(service.Options[OptionUmaAdapter])
}

// validateUmaMessageService checks a message service whose adapter is uma.
// Generic message checks (default_action, receive and reply modes) have
// already run in validateMessageServiceConfig.
func validateUmaMessageService(service *ServiceConfig, result *ServiceValidationResult) {
	options := service.Options
	key := strings.TrimSpace(options[OptionUmaAdapter])
	if key == "" {
		result.Issues = append(result.Issues, "uma message service requires option "+OptionUmaAdapter+" (one of "+strings.Join(UmaAdapterCatalogue(), ", ")+")")
		return
	}
	spec, ok := UmaAdapterSpecFor(key)
	if !ok {
		result.Issues = append(result.Issues, fmt.Sprintf("unsupported uma adapter %q (one of %s)", key, strings.Join(UmaAdapterCatalogue(), ", ")))
		return
	}
	requireEnv(service.Env, result, spec.RequiredEnv...)
	if spec.validate != nil {
		spec.validate(service, result)
	}
	switch spec.PlatformFor(options) {
	case UmaMediumSMS, UmaMediumWhatsApp:
		if strings.TrimSpace(options["allowed_numbers"]) == "" {
			result.Warnings = append(result.Warnings, "uma service has no allowed numbers; any sender can route inbound messages")
		}
	}
	if key == "meta.whatsapp" && strings.TrimSpace(service.Env[EnvWhatsAppVerify]) == "" {
		result.Warnings = append(result.Warnings, "whatsapp verify token not set; Cloud webhook verification will fail")
	}
}

// validateUmaTwilioOptions checks what the uma Twilio adapters need: a sender
// number, and a public webhook URL without query or fragment. uma verifies
// the webhook signature over exactly that URL, so it is mandatory here, where
// the aps-native providers rebuild it from forwarded headers.
func validateUmaTwilioOptions(options map[string]string, result *ServiceValidationResult, key string) {
	if strings.TrimSpace(options["from"]) == "" {
		result.Issues = append(result.Issues, "uma "+key+" requires option from")
	}
	raw := strings.TrimSpace(options["webhook_url"])
	if raw == "" {
		result.Issues = append(result.Issues, "uma "+key+" requires option webhook_url (the public URL configured at Twilio, with no query or fragment)")
		return
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil:
		result.Issues = append(result.Issues, "uma "+key+" option webhook_url must be an absolute http(s) URL without credentials")
	case u.RawQuery != "" || u.Fragment != "" || strings.Contains(raw, "?") || strings.Contains(raw, "#"):
		result.Issues = append(result.Issues, "uma "+key+" option webhook_url must have no query or fragment; configure the base URL at Twilio")
	}
}

// describeUmaServiceRuntime reports a uma-bridged message service. Its
// provider webhook is received by the uma engine at the same route the
// native providers use, so operators keep the URL configured at the
// provider.
func describeUmaServiceRuntime(service *ServiceConfig) ServiceRuntimeInfo {
	provider := AdapterUma
	key := strings.TrimSpace(service.Options[OptionUmaAdapter])
	if key != "" {
		provider = AdapterUma + "/" + key
	}
	replyMode := "provider delivery"
	if value := strings.TrimSpace(service.Options["reply"]); value != "" {
		replyMode = value
	}
	routing := OptionDefaultAction
	executes := "normalized message execution handoff"
	if HasRouteTable(service) {
		routing = "sender route table"
		executes = "sender route table -> profile action"
	}
	route := ServiceWebhookPath(service)
	methods := "POST"
	if key == "meta.whatsapp" {
		// Cloud API subscription verification is a GET at the same URL.
		methods = "GET, POST"
	}
	spec, _ := UmaAdapterSpecFor(key)
	verifies := "the provider signature"
	if spec.GenericWebhook {
		auth, _ := UmaWebhookAuthFor(service.Options)
		verifies = "webhook auth: " + auth.Scheme
		if auth.Scheme == "" {
			verifies = "webhook auth: none configured, every request is refused"
		}
	}
	info := ServiceRuntimeInfo{
		Receives: "HTTP " + methods + " " + route + " (uma verifies " + verifies + ")",
		Executes: executes,
		Replies:  provider + " " + replyMode,
		Maturity: "ready",
		Routes:   []string{route},
		Metadata: ServiceRuntimeMetadata{
			Runtime:       "message-provider",
			Provider:      provider,
			Ingress:       "uma engine webhook receiver",
			Handoff:       "normalized message execution handoff",
			Delivery:      "uma reply, sent asynchronously by the engine worker",
			Retry:         "provider delivery retry policy max_attempts=3 base_delay=1s max_delay=30s",
			Routing:       routing,
			ErrorHooks:    []string{"ingress", "normalize", "route", "execute", "deliver", "retry"},
			ReceiveMode:   "webhook",
			DeliveryModes: []string{"text"},
		},
	}
	switch {
	case spec.ReceiveOnly:
		info.Replies = "disabled: " + provider + " receives only; an action's reply is refused, not sent"
		info.Metadata.Delivery = "disabled: receive only; a reply is refused and logged"
		info.Metadata.Retry = replyNone
		info.Metadata.ErrorHooks = []string{"ingress", "normalize", "route", "execute"}
		info.Metadata.DeliveryModes = nil
	case key == UmaAdapterRelay:
		info.Replies = provider + " " + replyMode + ", posted to the gateway's callback"
		info.Metadata.Delivery = "uma reply, posted to the gateway's callback by the engine worker"
	}
	return info
}

// UmaIgnoredOption is an option a uma service carries but the uma bridge never
// reads, or reads only to refuse what it asks for.
type UmaIgnoredOption struct {
	// Option is the service option key.
	Option string
	// Flag is the `service add` flag that sets it, without dashes; empty when
	// only `--option` reaches it.
	Flag string
	// Advice says why it has no effect and what to use instead.
	Advice string
	// Default reports that the option is unset and its default, Value, is
	// what has no effect.
	Default bool
	// Value is the default in effect when Default is set; empty otherwise,
	// since an operator's value is never repeated back.
	Value string
}

// umaIgnoredRule is one option the bridge does not read. applies, when set,
// narrows the rule to the values or uma adapters where that holds; a rule
// without it applies whenever the option is set to a non-empty value.
type umaIgnoredRule struct {
	option string
	flag   string
	// boolean marks an option whose value false is its default.
	boolean bool
	// fallback, when set, is the value an unset option takes; the rule then
	// judges it too.
	fallback string
	applies  func(adapterKey, value string) bool
	advice   func(adapterKey string) string
}

func fixedAdvice(text string) func(string) string { return func(string) string { return text } }

const (
	adviceTemplates   = "templates are not sent; a uma service replies with plain text"
	adviceGenericAuth = "uma verifies the provider's own signature and aps applies no generic webhook " +
		"auth to a uma route"
	adviceSlack    = "it is a Slack option; a uma service is SMS, WhatsApp or mail"
	adviceTelegram = "it is a Telegram option; a uma service is SMS, WhatsApp or mail"
	adviceTicket   = "it is a ticket option; a uma service is a message service"
	adviceRelay    = "only the relay.messages adapter reads it"
	adviceMail     = "only the http.mail adapter reads it"
	adviceMailOnly = "http.mail receives only; an action's reply is refused and logged, never sent " +
		"(set --reply none)"
)

// genericWebhookAdapter reports whether the adapter verifies the generic
// webhook auth options rather than a provider signature.
func genericWebhookAdapter(adapterKey string) bool {
	spec, ok := UmaAdapterSpecFor(adapterKey)
	return ok && spec.GenericWebhook
}

// notGenericWebhook narrows a generic webhook auth option to the adapters
// that verify a provider signature instead.
func notGenericWebhook(adapterKey, _ string) bool { return !genericWebhookAdapter(adapterKey) }

func adviceVerifyToken(adapterKey string) string {
	if adapterKey == "meta.whatsapp" {
		return "bind the token with --env " + EnvWhatsAppVerify + "=secret:NAME"
	}
	return "this uma adapter has no verify token"
}

func adviceSigningSecret(adapterKey string) string {
	switch {
	case adapterKey == "meta.whatsapp":
		return "bind the app secret with --env " + EnvWhatsAppSecret + "=secret:NAME"
	case genericWebhookAdapter(adapterKey):
		return "set --signature-secret-env NAME and bind it with --env NAME=secret:NAME"
	}
	return "bind the auth token with --env " + EnvTwilioAuthToken + "=secret:NAME " +
		"(Twilio signs with it)"
}

// adviceLiteralSecret is the advice for a secret written into the options.
func adviceLiteralSecret(flag string) func(string) string {
	return func(adapterKey string) string {
		if genericWebhookAdapter(adapterKey) {
			return "secrets are read from bindings only: set " + flag + " NAME and --env NAME=secret:NAME"
		}
		return adviceGenericAuth
	}
}

// genericAuthRule is a generic webhook auth option: read by relay.messages and
// http.mail, ignored by the adapters that check a provider signature.
func genericAuthRule(option, flag string, boolean bool) umaIgnoredRule {
	return umaIgnoredRule{
		option: option, flag: flag, boolean: boolean,
		applies: notGenericWebhook, advice: fixedAdvice(adviceGenericAuth),
	}
}

// onlyFor is a rule for an option a single adapter reads.
func onlyFor(option, adapterKey, advice string) umaIgnoredRule {
	return umaIgnoredRule{
		option:  option,
		applies: func(key, _ string) bool { return key != adapterKey },
		advice:  fixedAdvice(advice),
	}
}

// umaIgnoredRules is every option a uma service accepts and the bridge does
// not read, derived from what the bridge consumes: its channel configuration
// (internal/adapters/umabridge catalogue), the env bindings it resolves, and
// the routing, reply policy and allowlists aps applies to the normalized
// message. The native providers read most of these; the uma route skips
// their request validators and normalizers.
var umaIgnoredRules = []umaIgnoredRule{
	{option: "template_name", flag: "template-name", advice: fixedAdvice(adviceTemplates)},
	{option: "language_code", flag: "language-code", advice: fixedAdvice(adviceTemplates)},
	{option: "template_required", flag: "template-required", boolean: true, advice: fixedAdvice(adviceTemplates)},
	{option: "verify_token", flag: "verify-token", advice: adviceVerifyToken},
	{option: "verify_token_env", flag: "verify-token-env", advice: adviceVerifyToken},
	{option: "signing_secret_env", flag: "signing-secret-env", advice: adviceSigningSecret},
	{option: "app_secret", advice: adviceSigningSecret},
	{option: "app_secret_env", advice: adviceSigningSecret},
	{option: "signing_secret", advice: adviceSigningSecret},
	genericAuthRule(OptionAuthScheme, "auth-scheme", false),
	genericAuthRule(OptionAuthTokenEnv, "auth-token-env", false),
	genericAuthRule(OptionSignatureSecretEnv, "signature-secret-env", false),
	genericAuthRule(OptionAuthHeader, "", false),
	genericAuthRule(OptionTimestampHeader, "", false),
	genericAuthRule(OptionRequireTimestamp, "", true),
	genericAuthRule(OptionTimestampTolerance, "", false),
	genericAuthRule(OptionReplayIDHeader, "", false),
	genericAuthRule(OptionRequireReplayCheck, "", true),
	genericAuthRule(OptionAuthPayload, "", false),
	genericAuthRule(OptionAuthAllowUnsigned, "", true),
	genericAuthRule(OptionAuthMinSecretBytes, "", false),
	{option: OptionAuthToken, advice: adviceLiteralSecret("--auth-token-env")},
	{option: OptionSignatureSecret, advice: adviceLiteralSecret("--signature-secret-env")},
	onlyFor(OptionMedium, UmaAdapterRelay, adviceRelay),
	onlyFor(OptionCallbackURL, UmaAdapterRelay, adviceRelay),
	onlyFor(OptionCallbackHeader, UmaAdapterRelay, adviceRelay),
	onlyFor(OptionRecipients, UmaAdapterHTTPMail, adviceMail),
	onlyFor(OptionFormats, UmaAdapterHTTPMail, adviceMail),
	onlyFor(OptionRecipientHeader, UmaAdapterHTTPMail, adviceMail),
	onlyFor(OptionTrustedAuthservID, UmaAdapterHTTPMail, adviceMail),
	onlyFor(OptionAuthResultsField, UmaAdapterHTTPMail, adviceMail),
	{option: "webhook_secret_token", flag: "webhook-secret-token", advice: fixedAdvice(adviceTelegram)},
	{option: "webhook_secret_token_env", flag: "webhook-secret-token-env", advice: fixedAdvice(adviceTelegram)},
	{option: "bot_user_id", flag: "bot-user-id", advice: fixedAdvice(adviceSlack)},
	{option: "require_bot_mention", flag: "require-bot-mention", boolean: true, advice: fixedAdvice(adviceSlack)},
	{option: "dedup_ttl", flag: "dedup-ttl", advice: fixedAdvice(adviceSlack)},
	{option: "site", flag: "site", advice: fixedAdvice(adviceTicket)},
	{option: "project", flag: "project", advice: fixedAdvice(adviceTicket)},
	{option: "jql", flag: "jql", advice: fixedAdvice(adviceTicket)},
	{option: "workspace", flag: "linear-workspace", advice: fixedAdvice(adviceTicket)},
	{option: "team", flag: "team", advice: fixedAdvice(adviceTicket)},
	{option: "group", flag: "group", advice: fixedAdvice(adviceTicket)},
	{option: "events", flag: "events", advice: fixedAdvice(adviceTicket)},
	{option: "provider", flag: "provider", advice: fixedAdvice("pick the provider with --uma-adapter")},
	{
		option: "receive", flag: "receive",
		applies: func(_, value string) bool { return !strings.EqualFold(value, "webhook") },
		advice:  fixedAdvice("a uma service receives by webhook only"),
	},
	{
		option: "phone_number_id", flag: "phone-number-id",
		applies: func(key, _ string) bool { return key != "meta.whatsapp" },
		advice:  fixedAdvice("only the meta.whatsapp adapter reads it"),
	},
	{
		option: "from", flag: "from",
		applies: func(key, _ string) bool { return key == "meta.whatsapp" || key == UmaAdapterHTTPMail },
		advice: func(key string) string {
			if key == UmaAdapterHTTPMail {
				return "http.mail accepts mail for --option recipients"
			}
			return "meta.whatsapp sends from --phone-number-id"
		},
	},
	{
		option: "webhook_url", flag: "webhook-url",
		applies: func(key, _ string) bool { return key == "meta.whatsapp" || genericWebhookAdapter(key) },
		advice: func(key string) string {
			if key == "meta.whatsapp" {
				return "meta.whatsapp does not use it; the callback URL is set at Meta"
			}
			return key + " checks webhook auth, not a signed URL"
		},
	},
	{
		option: optionReply, flag: optionReply, fallback: "text",
		applies: func(key, value string) bool {
			return key == UmaAdapterHTTPMail && !strings.EqualFold(value, replyNone)
		},
		advice: fixedAdvice(adviceMailOnly),
	},
}

// UmaIgnoredOptions lists the options set on a uma service that the bridge
// ignores, in a stable order. An option that is unset, blank, or a boolean
// left false is not listed, unless its default has no effect either (an
// http.mail service's reply mode). It is empty for any other service.
func UmaIgnoredOptions(service *ServiceConfig) []UmaIgnoredOption {
	if service == nil || service.Adapter != AdapterUma {
		return nil
	}
	adapterKey := strings.TrimSpace(service.Options[OptionUmaAdapter])
	var out []UmaIgnoredOption
	for _, rule := range umaIgnoredRules {
		value := strings.TrimSpace(service.Options[rule.option])
		defaulted := value == "" && rule.fallback != ""
		if defaulted {
			value = rule.fallback
		}
		if value == "" || (rule.boolean && !truthyServiceOption(value)) {
			continue
		}
		if rule.applies != nil && !rule.applies(adapterKey, value) {
			continue
		}
		hit := UmaIgnoredOption{Option: rule.option, Flag: rule.flag, Advice: rule.advice(adapterKey), Default: defaulted}
		if defaulted {
			hit.Value = value
		}
		out = append(out, hit)
	}
	return out
}
