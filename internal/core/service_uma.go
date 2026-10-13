package core

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

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
	// validate checks the adapter's options.
	validate func(options map[string]string, result *ServiceValidationResult)
}

var umaAdapterSpecs = map[string]UmaAdapterSpec{
	"twilio.sms": {
		Key:         "twilio.sms",
		Platform:    "sms",
		RequiredEnv: []string{EnvTwilioAccountSID, EnvTwilioAuthToken},
		validate: func(o map[string]string, r *ServiceValidationResult) {
			validateUmaTwilioOptions(o, r, "twilio.sms")
		},
	},
	"twilio.whatsapp": {
		Key:         "twilio.whatsapp",
		Platform:    "whatsapp",
		RequiredEnv: []string{EnvTwilioAccountSID, EnvTwilioAuthToken},
		validate: func(o map[string]string, r *ServiceValidationResult) {
			validateUmaTwilioOptions(o, r, "twilio.whatsapp")
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(o["from"])), "whatsapp:") {
				r.Issues = append(r.Issues, "uma twilio.whatsapp option from is configured without the whatsapp: prefix")
			}
		},
	},
	"meta.whatsapp": {
		Key:         "meta.whatsapp",
		Platform:    "whatsapp",
		RequiredEnv: []string{EnvWhatsAppToken, EnvWhatsAppSecret},
		OptionalEnv: []string{EnvWhatsAppVerify},
		validate: func(o map[string]string, r *ServiceValidationResult) {
			if !looksNumericID(o["phone_number_id"]) {
				r.Issues = append(r.Issues, "uma meta.whatsapp requires numeric option phone_number_id")
			}
		},
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
		spec.validate(options, result)
	}
	if strings.TrimSpace(options["allowed_numbers"]) == "" && spec.Platform != "" {
		result.Warnings = append(result.Warnings, "uma service has no allowed numbers; any sender can route inbound messages")
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
	return ServiceRuntimeInfo{
		Receives: "HTTP " + methods + " " + route + " (uma verifies the provider signature)",
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
}

// UmaIgnoredOption is an option a uma service carries but the uma bridge never
// reads.
type UmaIgnoredOption struct {
	// Option is the service option key.
	Option string
	// Flag is the `service add` flag that sets it, without dashes; empty when
	// only `--option` reaches it.
	Flag string
	// Advice says why it has no effect and what to use instead.
	Advice string
}

// umaIgnoredRule is one option the bridge does not read. applies, when set,
// narrows the rule to the values or uma adapters where that holds; a rule
// without it applies whenever the option is set to a non-empty value.
type umaIgnoredRule struct {
	option string
	flag   string
	// boolean marks an option whose value false is its default.
	boolean bool
	applies func(adapterKey, value string) bool
	advice  func(adapterKey string) string
}

func fixedAdvice(text string) func(string) string { return func(string) string { return text } }

const (
	adviceTemplates   = "templates are not sent; a uma service replies with plain text"
	adviceGenericAuth = "uma verifies the provider's own signature and aps applies no generic webhook " +
		"auth to a uma route"
	adviceSlack    = "it is a Slack option; a uma service is SMS or WhatsApp"
	adviceTelegram = "it is a Telegram option; a uma service is SMS or WhatsApp"
	adviceTicket   = "it is a ticket option; a uma service is a message service"
)

func adviceVerifyToken(adapterKey string) string {
	if adapterKey == "meta.whatsapp" {
		return "bind the token with --env " + EnvWhatsAppVerify + "=secret:NAME"
	}
	return "this uma adapter has no verify token"
}

func adviceSigningSecret(adapterKey string) string {
	if adapterKey == "meta.whatsapp" {
		return "bind the app secret with --env " + EnvWhatsAppSecret + "=secret:NAME"
	}
	return "bind the auth token with --env " + EnvTwilioAuthToken + "=secret:NAME " +
		"(Twilio signs with it)"
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
	{option: "auth_scheme", flag: "auth-scheme", advice: fixedAdvice(adviceGenericAuth)},
	{option: "auth_token_env", flag: "auth-token-env", advice: fixedAdvice(adviceGenericAuth)},
	{option: "signature_secret_env", flag: "signature-secret-env", advice: fixedAdvice(adviceGenericAuth)},
	{option: "auth_token", advice: fixedAdvice(adviceGenericAuth)},
	{option: "auth_header", advice: fixedAdvice(adviceGenericAuth)},
	{option: "signature_secret", advice: fixedAdvice(adviceGenericAuth)},
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
		applies: func(key, _ string) bool { return key == "meta.whatsapp" },
		advice:  fixedAdvice("meta.whatsapp sends from --phone-number-id"),
	},
	{
		option: "webhook_url", flag: "webhook-url",
		applies: func(key, _ string) bool { return key == "meta.whatsapp" },
		advice:  fixedAdvice("meta.whatsapp does not use it; the callback URL is set at Meta"),
	},
}

// UmaIgnoredOptions lists the options set on a uma service that the bridge
// ignores, in a stable order. An option that is unset, blank, or a boolean
// left false is not listed. It is empty for any other service.
func UmaIgnoredOptions(service *ServiceConfig) []UmaIgnoredOption {
	if service == nil || service.Adapter != AdapterUma {
		return nil
	}
	adapterKey := strings.TrimSpace(service.Options[OptionUmaAdapter])
	var out []UmaIgnoredOption
	for _, rule := range umaIgnoredRules {
		value := strings.TrimSpace(service.Options[rule.option])
		if value == "" || (rule.boolean && !truthyServiceOption(value)) {
			continue
		}
		if rule.applies != nil && !rule.applies(adapterKey, value) {
			continue
		}
		out = append(out, UmaIgnoredOption{Option: rule.option, Flag: rule.flag, Advice: rule.advice(adapterKey)})
	}
	return out
}
