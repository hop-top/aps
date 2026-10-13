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
