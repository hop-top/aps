package umabridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma/adapter"
	"hop.top/uma/adapter/messaging/meta/whatsapp"
	"hop.top/uma/adapter/messaging/twilio/sms"
	twhatsapp "hop.top/uma/adapter/messaging/twilio/whatsapp"
	"hop.top/uma/model"
)

// Options are host-side settings of the provider factories. They are test
// seams: an operator cannot set them from a service, so a channel cannot
// redirect credentials to another origin.
type Options struct {
	// TwilioBaseURL overrides the Twilio REST origin.
	TwilioBaseURL string
	// GraphBaseURL overrides the Graph API origin.
	GraphBaseURL string
	// RESTTimeout bounds one provider REST call; zero takes the adapter's.
	RESTTimeout time.Duration
}

// entry is one uma adapter the bridge can host: how to build its factory,
// its channel configuration and destination from a service, and which env
// binding backs each credential purpose.
type entry struct {
	typeKey model.TypeKey
	// factory builds the adapter factory.
	factory func(Options) adapter.Factory
	// purposes maps a uma credential purpose to an env binding name.
	purposes map[string]string
	// channel builds the channel configuration and the inbox destination
	// (the address inbound messages arrive at) from the service options and
	// resolved bindings.
	channel func(options map[string]string, binding func(string) (string, bool)) (model.Extension, string, error)
	// address renders a provider address as aps carries it in sender and
	// channel ids.
	address func(string) string
	// thread is the thread a received message belongs to, when the provider
	// scopes replies to the message they quote; nil for a provider that does
	// not, or a message that quotes nothing.
	thread func(*model.Message) *msgtypes.Thread
}

var catalogue = map[string]entry{
	"twilio.sms": {
		typeKey: model.TypeTwilioSms,
		factory: func(o Options) adapter.Factory {
			return sms.NewFactory(sms.Options{BaseURL: o.TwilioBaseURL, Timeout: o.RESTTimeout})
		},
		purposes: map[string]string{
			sms.PurposeREST:    core.EnvTwilioAuthToken,
			sms.PurposeWebhook: core.EnvTwilioAuthToken,
		},
		channel: func(options map[string]string, binding func(string) (string, bool)) (model.Extension, string, error) {
			sid, err := requireBinding(binding, core.EnvTwilioAccountSID)
			if err != nil {
				return model.Extension{}, "", err
			}
			cfg := sms.Config{
				AccountSID:          sid,
				From:                options["from"],
				MessagingServiceSID: options["messaging_service_sid"],
				WebhookURL:          options["webhook_url"],
			}
			return cfg.Extension(), cfg.From, nil
		},
		address: func(s string) string { return s },
	},
	"twilio.whatsapp": {
		typeKey: model.TypeTwilioWhatsapp,
		factory: func(o Options) adapter.Factory {
			return twhatsapp.NewFactory(twhatsapp.Options{BaseURL: o.TwilioBaseURL, Timeout: o.RESTTimeout})
		},
		purposes: map[string]string{
			twhatsapp.PurposeREST:    core.EnvTwilioAuthToken,
			twhatsapp.PurposeWebhook: core.EnvTwilioAuthToken,
		},
		channel: func(options map[string]string, binding func(string) (string, bool)) (model.Extension, string, error) {
			sid, err := requireBinding(binding, core.EnvTwilioAccountSID)
			if err != nil {
				return model.Extension{}, "", err
			}
			cfg := twhatsapp.Config{
				AccountSID:          sid,
				From:                options["from"],
				MessagingServiceSID: options["messaging_service_sid"],
				WebhookURL:          options["webhook_url"],
			}
			return cfg.Extension(), cfg.From, nil
		},
		// aps carries Twilio WhatsApp addresses with their whatsapp: prefix
		// in sender and channel ids; uma strips it.
		address: func(s string) string {
			if s == "" || strings.HasPrefix(s, "whatsapp:") {
				return s
			}
			return "whatsapp:" + s
		},
	},
	"meta.whatsapp": {
		typeKey: model.TypeMetaWhatsapp,
		factory: func(o Options) adapter.Factory {
			return whatsapp.NewFactory(whatsapp.Options{BaseURL: o.GraphBaseURL})
		},
		purposes: map[string]string{
			whatsapp.PurposeGraph:       core.EnvWhatsAppToken,
			whatsapp.PurposeAppSecret:   core.EnvWhatsAppSecret,
			whatsapp.PurposeVerifyToken: core.EnvWhatsAppVerify,
		},
		channel: func(options map[string]string, _ func(string) (string, bool)) (model.Extension, string, error) {
			cfg := whatsapp.Config{PhoneNumberID: options["phone_number_id"]}
			return cfg.Extension(), cfg.PhoneNumberID, nil
		},
		address: func(s string) string { return s },
		// The native Cloud provider scopes a reply to the message it quotes
		// (context.id); uma keeps that context in the inbound payload
		// extension of the message.
		thread: func(m *model.Message) *msgtypes.Thread {
			for _, ext := range m.Metadata {
				if ext.Name != whatsapp.InboundSchema {
					continue
				}
				var inbound struct {
					Context struct {
						ID string `json:"id"`
					} `json:"context"`
				}
				if json.Unmarshal(ext.Data, &inbound) == nil && inbound.Context.ID != "" {
					return &msgtypes.Thread{ID: inbound.Context.ID, Type: msgtypes.ThreadTypeReply}
				}
			}
			return nil
		},
	},
}

func requireBinding(binding func(string) (string, bool), name string) (string, error) {
	v, ok := binding(name)
	if !ok {
		return "", fmt.Errorf("env binding %s is not available", name)
	}
	return v, nil
}

func lookupEntry(key string) (entry, error) {
	e, ok := catalogue[strings.TrimSpace(key)]
	if !ok {
		return entry{}, fmt.Errorf("unsupported uma adapter %q (one of %s)", key, strings.Join(core.UmaAdapterCatalogue(), ", "))
	}
	return e, nil
}
