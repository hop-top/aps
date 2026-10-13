package umabridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma/adapter"
	"hop.top/uma/adapter/messaging/relay"
	"hop.top/uma/adapter/web/webhookauth"
	"hop.top/uma/model"
)

// relayFields are the JSON keys aps's native generic relay reads, so a
// gateway in the field keeps its payload. WaId is left out: it is the
// sender's number, and as a message id it would make every later message of
// that sender a duplicate under uma, whose event id binds sender and id.
//
//nolint:goconst // the native relay's names, listed once
var relayFields = relay.Fields{
	From:      []string{"From", "from"},
	To:        []string{"To", "to"},
	Body:      []string{"Body", "body", "text"},
	MessageID: []string{"MessageSid", "SmsSid", "message_id"},
	Media:     []string{"media"},
}

var relayEntry = entry{
	typeKey: model.TypeRelayMessages,
	factory: func(d factoryDeps) adapter.Factory {
		return relay.NewFactory(relay.Options{
			CallbackHosts: d.host.Relay.CallbackHosts,
			AllowHTTP:     d.host.Relay.AllowHTTP,
			Timeout:       d.opts.RESTTimeout,
			ReplayStore:   d.replay,
		})
	},
	purposes: func(options map[string]string) map[string]string {
		auth, _ := core.UmaWebhookAuthFor(options)
		return map[string]string{
			relay.PurposeWebhook:  auth.SecretBinding,
			relay.PurposeCallback: core.EnvRelayCallbackToken,
		}
	},
	channel: func(options map[string]string, _ func(string) (string, bool)) (model.Extension, string, error) {
		auth, issues := core.UmaWebhookAuthFor(options)
		if len(issues) > 0 {
			return model.Extension{}, "", fmt.Errorf("webhook auth: %s", strings.Join(issues, "; "))
		}
		cfg := relay.Config{
			Medium:  strings.TrimSpace(options[core.OptionMedium]),
			Auth:    relayAuth(auth),
			Inbound: relay.Inbound{Fields: relayFields, GenerateMissingID: true},
		}
		if url := strings.TrimSpace(options[core.OptionCallbackURL]); url != "" {
			cfg.Callback = &relay.Callback{URL: url}
			if h := strings.TrimSpace(options[core.OptionCallbackHeader]); h != "" {
				cfg.Callback.Auth, cfg.Callback.Header = relay.AuthHeader, h
			}
		}
		return cfg.Extension(), strings.TrimSpace(options["from"]), nil
	},
	address:        same,
	normalize:      relayMedia,
	noRedirect:     true,
	webhookPurpose: relay.PurposeWebhook,
}

// relayAuth is the relay's JSON form of the mapped webhook auth.
func relayAuth(a core.UmaWebhookAuth) relay.Auth {
	out := relay.Auth{
		Scheme: a.Scheme, Header: a.Header, Encoding: a.Encoding, Prefix: a.Prefix, PrefixOptional: a.PrefixOptional,
		Payload: a.Payload, KeyEncoding: a.KeyEncoding, MinSecretBytes: a.MinSecretBytes,
	}
	if a.TimestampHeader != "" {
		out.Timestamp = relay.AuthTimestamp{
			Header: a.TimestampHeader, Format: a.TimestampFormat,
			Tolerance: a.TimestampTolerance.String(), AllowUnsigned: a.TimestampUnsigned,
		}
	}
	if a.ReplayHeader != "" {
		out.Replay = relay.AuthReplay{
			Source: string(webhookauth.ReplayByHeader), Header: a.ReplayHeader, AllowUnsigned: a.ReplayUnsigned,
		}
	}
	return out
}

// relayMedia lists the media URLs a gateway sent as attachments, as the
// native generic relay passed MediaUrl through. The URLs are the gateway's
// choice: uma never fetched them and neither does the bridge.
func relayMedia(m *model.Message, msg *msgtypes.NormalizedMessage) {
	for _, part := range m.Content.Parts {
		if part.Kind != model.PartStructured || part.Structured == nil || part.Structured.Name != relay.PayloadMedia {
			continue
		}
		var media struct {
			Items []struct {
				URL         string `json:"url"`
				ContentType string `json:"content_type"`
			} `json:"items"`
		}
		if json.Unmarshal(part.Structured.Data, &media) != nil {
			continue
		}
		for _, item := range media.Items {
			msg.Attachments = append(msg.Attachments, msgtypes.Attachment{
				Type: attachmentType(item.ContentType), URL: item.URL, MimeType: item.ContentType,
			})
		}
	}
}
