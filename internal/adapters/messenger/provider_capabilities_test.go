package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	coremessenger "hop.top/aps/internal/core/messenger"
)

// Capability probes: every ingress mode and delivery mode a provider
// declares in Metadata() must be backed by behaviour observable at the
// provider's transport boundary. A declaration with no probe, or whose
// probe fails, fails the test and names the provider and capability. A
// declaration that cannot be verified mechanically must be listed in
// capabilityExceptions with a reason, so new declarations are a conscious
// choice.

const (
	probeText    = "capability-probe-text"
	probeFileURL = "https://files.example.test/capability-probe.bin"
)

// deliveryObservation is what a probe saw leave the provider.
type deliveryObservation struct {
	// calls counts requests that reached the transport.
	calls int
	// contentType is the outbound request content type, when the transport
	// exposes one.
	contentType string
	// body is the JSON (or raw) form of the outbound request.
	body string
}

// capabilityProbe drives one provider against its fake transport.
type capabilityProbe struct {
	// external marks providers living outside this package; they cannot be
	// enumerated from the package sources but share the same contract.
	external bool
	provider func() (coremessenger.MessageProvider, func() deliveryObservation)
	// ingress maps each declared ingress mode to a native payload the
	// provider must normalize into a message carrying probeText.
	ingress map[coremessenger.IngressMode]coremessenger.NativeIngress
}

type capabilityKey struct {
	provider   string
	capability string
}

// capabilityExceptions lists declarations a probe cannot verify. Each entry
// needs a reason; the test fails when an exception is no longer declared.
var capabilityExceptions = map[capabilityKey]string{
	{"DiscordProvider", "ingress:stream"}: "pending removal of the declaration: no Gateway client exists.",
	{"DiscordProvider", "delivery:file"}:  "pending removal of the declaration: attachments are sent as URL embeds, not uploads.",
	{"WhatsAppProvider", "delivery:file"}: "declared per provider, honoured by the Twilio transport only " +
		"(first attachment as MediaUrl); the Cloud API transport sends text only. " +
		"The provider-level fake cannot tell the two apart.",
}

func probeJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "marshal error: " + err.Error()
	}
	return string(data)
}

// recordingSMSTransport records outbound SMS sends.
type recordingSMSTransport struct {
	sends []coremessenger.SMSDelivery
}

func (r *recordingSMSTransport) SendSMS(_ context.Context, d coremessenger.SMSDelivery) (*coremessenger.SMSDeliveryResult, error) {
	r.sends = append(r.sends, d)
	return &coremessenger.SMSDeliveryResult{MessageSID: "SM-out-1", Status: "queued", Provider: d.Provider}, nil
}

// uploadKeys are the request fields through which a provider references
// uploaded files. Link embeds and plain text do not count as file delivery.
var uploadKeys = map[string]bool{
	"attachments": true,
	"files":       true,
	"file":        true,
	"document":    true,
	"media":       true,
	"mediaurl":    true,
}

func carriesFileUpload(obs deliveryObservation) bool {
	if strings.HasPrefix(strings.ToLower(obs.contentType), "multipart/") {
		return true
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(obs.body), &decoded); err != nil {
		return false
	}
	for key, value := range decoded {
		if !uploadKeys[strings.ToLower(key)] {
			continue
		}
		switch typed := value.(type) {
		case nil:
		case string:
			if typed != "" {
				return true
			}
		case []any:
			if len(typed) > 0 {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func capabilityProbes() map[string]capabilityProbe {
	webhook := func(body, contentType string) map[coremessenger.IngressMode]coremessenger.NativeIngress {
		ingress := coremessenger.NativeIngress{
			ServiceID: "capability-probe",
			Mode:      coremessenger.IngressModeWebhook,
			Body:      []byte(body),
		}
		if contentType != "" {
			ingress.Headers = map[string][]string{"Content-Type": {contentType}}
		}
		return map[coremessenger.IngressMode]coremessenger.NativeIngress{coremessenger.IngressModeWebhook: ingress}
	}
	form := "application/x-www-form-urlencoded"

	return map[string]capabilityProbe{
		"SlackProvider": {
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &fakeSlackTransport{}
				provider := NewSlackProvider(SlackProviderConfig{BotToken: "probe-token", Transport: transport})
				return provider, func() deliveryObservation {
					return deliveryObservation{calls: transport.calls, body: probeJSON(transport.last)}
				}
			},
			ingress: webhook(`{"type":"event_callback","event_id":"Ev1","event_time":1710000000,"event":{"type":"app_mention","user":"U1","channel":"C1","channel_type":"channel","text":"<@U0BOT> `+probeText+`","ts":"1710000000.000001"}}`, ""),
		},
		"TelegramProvider": {
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &captureTelegramTransport{}
				provider := NewTelegramProvider(TelegramProviderConfig{BotToken: "probe-token", Transport: transport})
				return provider, func() deliveryObservation {
					calls := 0
					if transport.token != "" {
						calls = 1
					}
					return deliveryObservation{calls: calls, body: probeJSON(transport.request)}
				}
			},
			ingress: webhook(`{"update_id":1,"message":{"message_id":1,"date":1773230400,"from":{"id":42,"first_name":"Ada"},"chat":{"id":7,"type":"private"},"text":"`+probeText+`"}}`, ""),
		},
		"TeamsProvider": {
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &fakeTeamsTransport{}
				provider := newTestTeamsProvider(transport)
				return provider, func() deliveryObservation {
					return deliveryObservation{calls: transport.calls, body: probeJSON(transport.gotActivity)}
				}
			},
			ingress: webhook(`{"type":"message","id":"1","serviceUrl":"https://smba.example.test/","from":{"id":"29:1","name":"Megan"},"conversation":{"conversationType":"personal","id":"a:1"},"recipient":{"id":"28:bot"},"text":"`+probeText+`"}`, ""),
		},
		"DiscordProvider": {
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &captureDiscordTransport{statusCode: http.StatusOK, body: `{"id":"2","channel_id":"1"}`}
				provider := NewDiscordProvider(DiscordProviderConfig{
					BotToken:   "probe-token",
					APIBaseURL: "https://discord.example.test/api/v10",
					Client:     transport,
				})
				return provider, func() deliveryObservation {
					calls := 0
					if transport.url != "" {
						calls = 1
					}
					return deliveryObservation{calls: calls, contentType: transport.contentType, body: transport.requestBody}
				}
			},
			ingress: webhook(`{"op":0,"t":"MESSAGE_CREATE","d":{"id":"1","guild_id":"3","channel_id":"2","content":"`+probeText+`","author":{"id":"4","username":"alice"}}}`, ""),
		},
		"SMSProvider": {
			external: true,
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &recordingSMSTransport{}
				provider := coremessenger.NewSMSProvider(coremessenger.SMSProviderConfig{From: "+15550000001"}, transport)
				return provider, func() deliveryObservation {
					return deliveryObservation{calls: len(transport.sends), body: probeJSON(transport.sends)}
				}
			},
			ingress: webhook(url.Values{"Body": {probeText}, "From": {"+15550000002"}, "To": {"+15550000001"}, "MessageSid": {"SM1"}}.Encode(), form),
		},
		"WhatsAppProvider": {
			external: true,
			provider: func() (coremessenger.MessageProvider, func() deliveryObservation) {
				transport := &recordingWhatsAppTransport{}
				provider := coremessenger.NewWhatsAppProvider(coremessenger.WhatsAppProviderConfig{
					Provider: "twilio", From: "whatsapp:+15550000001",
				}, transport)
				return provider, func() deliveryObservation {
					transport.mu.Lock()
					defer transport.mu.Unlock()
					return deliveryObservation{calls: len(transport.sends), body: probeJSON(transport.sends)}
				}
			},
			ingress: webhook(url.Values{"Body": {probeText}, "From": {"whatsapp:+15550000002"}, "To": {"whatsapp:+15550000001"}, "MessageSid": {"SM1"}}.Encode(), form),
		},
	}
}

func TestProviderDeclaredCapabilitiesAreImplemented(t *testing.T) {
	probes := capabilityProbes()

	// Every provider declared in this package must have a probe: a new
	// provider fails here until it opts in.
	seen := map[string]bool{}
	for _, name := range metadataReceiverTypes(t) {
		seen[name] = true
		if _, ok := probes[name]; !ok {
			t.Errorf("provider %s declares capabilities but has no probe in capabilityProbes", name)
		}
	}
	for name, probe := range probes {
		if !probe.external && !seen[name] {
			t.Errorf("stale probe %s: no Metadata() method found in package sources", name)
		}
	}

	usedExceptions := map[capabilityKey]bool{}
	for name, probe := range probes {
		t.Run(name, func(t *testing.T) {
			provider, observe := probe.provider()
			meta := provider.Metadata()
			deliverer, canDeliver := provider.(coremessenger.ProviderDelivery)

			for _, mode := range meta.IngressModes {
				key := capabilityKey{name, "ingress:" + string(mode)}
				if reason, ok := capabilityExceptions[key]; ok {
					usedExceptions[key] = true
					t.Logf("acknowledged exception %s: %s", key.capability, reason)
					continue
				}
				ingress, ok := probe.ingress[mode]
				if !ok {
					t.Errorf("%s declares ingress mode %q but no probe backs it", name, mode)
					continue
				}
				ingress.Provider = meta.Provider
				msg, err := provider.NormalizeIngress(context.Background(), ingress)
				if err != nil {
					t.Errorf("%s declares ingress mode %q but its probe payload fails to normalize: %v", name, mode, err)
					continue
				}
				if !strings.Contains(msg.Text, probeText) {
					t.Errorf("%s declares ingress mode %q but normalized text = %q, want %q", name, mode, msg.Text, probeText)
				}
			}

			for _, mode := range meta.DeliveryModes {
				key := capabilityKey{name, "delivery:" + string(mode)}
				if reason, ok := capabilityExceptions[key]; ok {
					usedExceptions[key] = true
					t.Logf("acknowledged exception %s: %s", key.capability, reason)
					continue
				}
				if !canDeliver {
					t.Errorf("%s declares delivery mode %q but does not implement ProviderDelivery", name, mode)
					continue
				}
				request := coremessenger.DeliveryRequest{
					Provider:  meta.Provider,
					ServiceID: "capability-probe",
					ChannelID: "C1",
					Text:      probeText,
					Metadata: map[string]any{
						"teams_service_url": "https://smba.example.test",
						"to":                "+15550000002",
					},
				}
				switch mode {
				case coremessenger.DeliveryModeText:
				case coremessenger.DeliveryModeFile:
					request.Attachments = []coremessenger.Attachment{{Type: "file", URL: probeFileURL, MimeType: "application/octet-stream"}}
				case coremessenger.DeliveryModeReaction:
					t.Errorf("%s declares delivery mode %q but no probe backs it", name, mode)
					continue
				default:
					t.Errorf("%s declares delivery mode %q but no probe backs it", name, mode)
					continue
				}
				if _, err := deliverer.DeliverMessage(context.Background(), request); err != nil {
					t.Errorf("%s declares delivery mode %q but its probe delivery failed: %v", name, mode, err)
					continue
				}
				obs := observe()
				if obs.calls == 0 {
					t.Errorf("%s declares delivery mode %q but nothing reached its transport", name, mode)
					continue
				}
				if !strings.Contains(obs.body, probeText) {
					t.Errorf("%s declares delivery mode %q but the request lost the message text: %s", name, mode, obs.body)
				}
				if mode == coremessenger.DeliveryModeFile && !carriesFileUpload(obs) {
					t.Errorf("%s declares delivery mode %q but the request carries no file upload (text or link only): %s", name, mode, obs.body)
				}
			}
		})
	}

	for key, reason := range capabilityExceptions {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("exception %v has no reason", key)
		}
		if !usedExceptions[key] {
			t.Errorf("stale exception %v: capability is no longer declared", key)
		}
	}
}
