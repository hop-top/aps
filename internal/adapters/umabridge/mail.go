package umabridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma/adapter"
	"hop.top/uma/adapter/mail/httpmail"
	"hop.top/uma/adapter/web/webhookauth"
	"hop.top/uma/model"
)

// mailDestination is the one inbox destination of an http.mail service:
// every recipient it lists maps to it, so a message to two of them reaches
// the action once.
const mailDestination = "mail"

var httpMailEntry = entry{
	typeKey: model.TypeHttpMail,
	factory: func(d factoryDeps) adapter.Factory {
		return httpmail.NewFactory(httpmail.Options{ReplayStore: d.replay})
	},
	purposes: func(options map[string]string) map[string]string {
		auth, _ := core.UmaWebhookAuthFor(options)
		return map[string]string{httpmail.PurposeWebhook: auth.SecretBinding}
	},
	channel: func(options map[string]string, _ func(string) (string, bool)) (model.Extension, string, error) {
		auth, issues := core.UmaWebhookAuthFor(options)
		if len(issues) > 0 {
			return model.Extension{}, "", fmt.Errorf("webhook auth: %s", strings.Join(issues, "; "))
		}
		cfg := httpmail.Config{
			Auth:            mailAuth(auth),
			RecipientHeader: strings.TrimSpace(options[core.OptionRecipientHeader]),
			Evidence: httpmail.Evidence{
				TrustedAuthservID:          strings.TrimSpace(options[core.OptionTrustedAuthservID]),
				AuthenticationResultsField: strings.TrimSpace(options[core.OptionAuthResultsField]),
			},
		}
		if strings.TrimSpace(options[core.OptionFormats]) != "" {
			cfg.Formats = core.UmaMailFormats(options)
		}
		for _, r := range core.UmaMailRecipients(options) {
			cfg.Recipients = append(cfg.Recipients, httpmail.Recipient{Address: r, Destination: mailDestination})
		}
		return cfg.Extension(), mailDestination, nil
	},
	address:        same,
	normalize:      mailMessage,
	webhookPurpose: httpmail.PurposeWebhook,
}

// mailAuth is http.mail's JSON form of the mapped webhook auth: durations in
// whole seconds.
func mailAuth(a core.UmaWebhookAuth) httpmail.Auth {
	out := httpmail.Auth{
		Scheme: a.Scheme, Header: a.Header, Encoding: a.Encoding, Prefix: a.Prefix, PrefixOptional: a.PrefixOptional,
		Payload: a.Payload, KeyEncoding: a.KeyEncoding, MinSecretBytes: a.MinSecretBytes,
	}
	if a.TimestampHeader != "" {
		out.Timestamp = &httpmail.AuthTimestamp{
			Header: a.TimestampHeader, Format: a.TimestampFormat,
			ToleranceSeconds: max(1, int64(a.TimestampTolerance/time.Second)), AllowUnsigned: a.TimestampUnsigned,
		}
	}
	if a.ReplayHeader != "" {
		out.Replay = &httpmail.AuthReplay{
			Source: string(webhookauth.ReplayByHeader), Header: a.ReplayHeader, AllowUnsigned: a.ReplayUnsigned,
		}
	}
	return out
}

// mailRecord is the part of http.mail's message metadata the bridge reads.
type mailRecord struct {
	Recipients []string `json:"recipients"`
	Headers    struct {
		Subject string `json:"subject"`
	} `json:"headers"`
}

// mailParts is http.mail's description of a message's attachments.
type mailParts struct {
	Attachments []struct {
		Filename  string `json:"filename"`
		MediaType string `json:"media_type"`
		Size      int64  `json:"size"`
		Staged    bool   `json:"staged"`
		Kind      string `json:"kind"`
		URL       string `json:"url"`
	} `json:"attachments"`
}

// mailMessage keys an http.mail message as the native email relay did: the
// recipient as channel and the subject as a topic thread, so a converted
// service keeps its sessions. The id is uma's message ref, never the
// Message-ID, which the sender chooses and anyone can repeat. An attachment
// described with only a URL is passed through, as the native relay passed
// it, never fetched; staged bytes are materialized by the provider.
func mailMessage(m *model.Message, msg *msgtypes.NormalizedMessage) {
	msg.ID = string(m.Ref.ID)
	var rec mailRecord
	for _, ext := range m.Metadata {
		if ext.Name == httpmail.MessageSchema {
			_ = json.Unmarshal(ext.Data, &rec)
		}
	}
	if len(rec.Recipients) > 0 {
		to := rec.Recipients[0]
		msg.Channel = msgtypes.Channel{ID: to, Name: to, Type: msgtypes.ChannelTypeDirect, PlatformID: to}
	}
	if subject := strings.TrimSpace(rec.Headers.Subject); subject != "" {
		msg.Thread = &msgtypes.Thread{ID: subject, Type: msgtypes.ThreadTypeTopic}
		msg.PlatformMetadata["subject"] = subject
	}
	msg.PlatformMetadata["sender_assurance"] = m.Author.Assurance
	for _, part := range m.Content.Parts {
		if part.Kind != model.PartStructured || part.Structured == nil || part.Structured.Name != httpmail.PartsSchema {
			continue
		}
		var parts mailParts
		if json.Unmarshal(part.Structured.Data, &parts) != nil {
			continue
		}
		for _, a := range parts.Attachments {
			if a.Staged || a.URL == "" {
				continue
			}
			kind := a.Kind
			if kind == "" {
				kind = attachmentType(a.MediaType)
			}
			msg.Attachments = append(msg.Attachments, msgtypes.Attachment{
				Type: kind, URL: a.URL, MimeType: a.MediaType, SizeBytes: a.Size,
			})
		}
	}
}
