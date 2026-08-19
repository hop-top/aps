package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SyntheticProbeIdentity reports which sender, channel, and workspace
// identities a synthetic webhook probe impersonates and which service
// option supplied each one. A source of "synthetic" means no option
// constrained that identity and the built-in placeholder was used.
type SyntheticProbeIdentity struct {
	Sender          string
	SenderSource    string
	Channel         string
	ChannelSource   string
	Workspace       string
	WorkspaceSource string
}

// Service option keys the probe reads to pick an identity the service
// already allows. They mirror the messenger validator's allowlist keys.
const (
	probeOptionAllowedNumbers    = "allowed_numbers"
	probeOptionAllowedChats      = "allowed_chats"
	probeOptionAllowedChannels   = "allowed_channels"
	probeOptionAllowedGuilds     = "allowed_guilds"
	probeOptionFrom              = "from"
	probeOptionPhoneNumberID     = "phone_number_id"
	probeOptionProvider          = "provider"
	probeOptionRequireBotMention = "require_bot_mention"
	probeOptionBotUserID         = "bot_user_id"
	probeIdentitySourceSynthetic = "synthetic"
	probeText                    = "aps service test"
	probeSenderName              = "APS"
	probeSenderHandle            = "aps"
	probeSlackEventMessage       = "message"
	probeWhatsAppTextType        = "text"
)

// Message adapter names the probe can synthesise payloads for.
const (
	probeAdapterTelegram = "telegram"
	probeAdapterSlack    = "slack"
	probeAdapterDiscord  = "discord"
	probeAdapterSMS      = "sms"
	probeAdapterWhatsApp = "whatsapp"
	probeAdapterEmail    = "email"
)

// SyntheticMessageWebhookPayload builds the adapter-shaped inbound payload
// that `aps service test --probe` POSTs at a service webhook.
//
// The payload impersonates the first value of each configured allowlist
// (allowed_numbers, allowed_chats, allowed_channels, allowed_guilds) and
// the service's own channel identity (sms/whatsapp `from`, WhatsApp
// `phone_number_id`) so a correctly configured service accepts the probe
// through its real validation path. No marker bypasses validation: the
// probe passes only because it presents an identity the operator already
// allowed. Services without allowlists fall back to fixed placeholder
// identities.
func SyntheticMessageWebhookPayload(adapter string, options map[string]string) ([]byte, SyntheticProbeIdentity, error) {
	adapter = strings.TrimSpace(strings.ToLower(adapter))
	var (
		identity SyntheticProbeIdentity
		payload  any
		err      error
	)
	switch adapter {
	case probeAdapterTelegram:
		payload, identity, err = syntheticTelegramProbe(options)
	case probeAdapterSlack:
		payload, identity = syntheticSlackProbe(options)
	case probeAdapterDiscord:
		payload, identity = syntheticDiscordProbe(options)
	case probeAdapterSMS:
		payload, identity = syntheticPhoneProbe(options)
	case probeAdapterEmail:
		payload, identity = syntheticEmailProbe(options)
	case probeAdapterWhatsApp:
		if strings.EqualFold(strings.TrimSpace(options[probeOptionProvider]), "twilio") {
			payload, identity = syntheticPhoneProbe(options)
		} else {
			payload, identity = syntheticWhatsAppCloudProbe(options)
		}
	default:
		return nil, identity, fmt.Errorf("no synthetic webhook payload for message adapter %q", adapter)
	}
	if err != nil {
		return nil, identity, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, identity, fmt.Errorf("encoding %s probe payload: %w", adapter, err)
	}
	return body, identity, nil
}

type telegramProbeUpdate struct {
	UpdateID int64                `json:"update_id"`
	Message  telegramProbeMessage `json:"message"`
}

type telegramProbeMessage struct {
	MessageID int64             `json:"message_id"`
	From      telegramProbeUser `json:"from"`
	Chat      telegramProbeChat `json:"chat"`
	Date      int64             `json:"date"`
	Text      string            `json:"text"`
}

type telegramProbeUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

type telegramProbeChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func syntheticTelegramProbe(options map[string]string) (telegramProbeUpdate, SyntheticProbeIdentity, error) {
	const senderID int64 = 1001
	chatID := int64(-1001234567890)
	identity := SyntheticProbeIdentity{
		Sender: strconv.FormatInt(senderID, 10), SenderSource: probeIdentitySourceSynthetic,
		Channel: strconv.FormatInt(chatID, 10), ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstCSVOption(options, probeOptionAllowedChats); allowed != "" {
		parsed, err := strconv.ParseInt(allowed, 10, 64)
		if err != nil {
			return telegramProbeUpdate{}, identity, fmt.Errorf("%s entry %q is not a numeric Telegram chat id", probeOptionAllowedChats, allowed)
		}
		chatID = parsed
		identity.Channel, identity.ChannelSource = allowed, probeOptionAllowedChats
	}
	chatType := "group"
	if chatID > 0 {
		chatType = "private"
	}
	return telegramProbeUpdate{
		UpdateID: 1000001,
		Message: telegramProbeMessage{
			MessageID: 1,
			From:      telegramProbeUser{ID: senderID, FirstName: probeSenderName},
			Chat:      telegramProbeChat{ID: chatID, Type: chatType},
			Date:      time.Now().Unix(),
			Text:      probeText,
		},
	}, identity, nil
}

type slackProbeEnvelope struct {
	Type   string          `json:"type"`
	TeamID string          `json:"team_id,omitempty"`
	Event  slackProbeEvent `json:"event"`
}

type slackProbeEvent struct {
	Type        string `json:"type"`
	ClientMsgID string `json:"client_msg_id"`
	User        string `json:"user"`
	Channel     string `json:"channel"`
	Text        string `json:"text"`
	TS          string `json:"ts"`
}

func syntheticSlackProbe(options map[string]string) (slackProbeEnvelope, SyntheticProbeIdentity) {
	identity := SyntheticProbeIdentity{
		Sender: "U012TEST", SenderSource: probeIdentitySourceSynthetic,
		Channel: "C012TEST", ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstCSVOption(options, probeOptionAllowedChannels); allowed != "" {
		identity.Channel, identity.ChannelSource = allowed, probeOptionAllowedChannels
	}
	event := slackProbeEvent{
		Type:        probeSlackEventMessage,
		ClientMsgID: "aps-service-test",
		User:        identity.Sender,
		Channel:     identity.Channel,
		Text:        probeText,
		TS:          fmt.Sprintf("%d.000000", time.Now().Unix()),
	}
	if truthyServiceOption(options[probeOptionRequireBotMention]) {
		// require_bot_mention accepts app_mention events outright and, when
		// bot_user_id is set, plain messages addressed to that user.
		event.Type = "app_mention"
		if botUserID := strings.TrimSpace(options[probeOptionBotUserID]); botUserID != "" {
			event.Text = "<@" + botUserID + "> " + probeText
		}
	}
	envelope := slackProbeEnvelope{Type: "event_callback", Event: event}
	if allowed := firstCSVOption(options, probeOptionAllowedGuilds); allowed != "" {
		identity.Workspace, identity.WorkspaceSource = allowed, probeOptionAllowedGuilds
		envelope.TeamID = allowed
	}
	return envelope, identity
}

type discordProbeMessage struct {
	ID        string             `json:"id"`
	ChannelID string             `json:"channel_id"`
	GuildID   string             `json:"guild_id,omitempty"`
	Content   string             `json:"content"`
	Author    discordProbeAuthor `json:"author"`
	Timestamp string             `json:"timestamp"`
}

type discordProbeAuthor struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func syntheticDiscordProbe(options map[string]string) (discordProbeMessage, SyntheticProbeIdentity) {
	identity := SyntheticProbeIdentity{
		Sender: "987654321098765432", SenderSource: probeIdentitySourceSynthetic,
		Channel: "123456789012345678", ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstCSVOption(options, probeOptionAllowedChannels); allowed != "" {
		identity.Channel, identity.ChannelSource = allowed, probeOptionAllowedChannels
	}
	message := discordProbeMessage{
		ID:        "aps-service-test",
		ChannelID: identity.Channel,
		Content:   probeText,
		Author:    discordProbeAuthor{ID: identity.Sender, Username: probeSenderHandle},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if allowed := firstCSVOption(options, probeOptionAllowedGuilds); allowed != "" {
		identity.Workspace, identity.WorkspaceSource = allowed, probeOptionAllowedGuilds
		message.GuildID = allowed
	}
	return message, identity
}

// phoneProbeMessage is the Twilio-shaped inbound shared by sms and
// Twilio-backed whatsapp: the sender is the remote number and the channel
// is the service's own number.
type phoneProbeMessage struct {
	MessageSid string `json:"MessageSid"`
	From       string `json:"From"`
	To         string `json:"To"`
	Body       string `json:"Body"`
}

func syntheticPhoneProbe(options map[string]string) (phoneProbeMessage, SyntheticProbeIdentity) {
	identity := SyntheticProbeIdentity{
		Sender: "+15550100001", SenderSource: probeIdentitySourceSynthetic,
		Channel: "+15550100002", ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstCSVOption(options, probeOptionAllowedNumbers); allowed != "" {
		identity.Sender, identity.SenderSource = allowed, probeOptionAllowedNumbers
	}
	if from := strings.TrimSpace(options[probeOptionFrom]); from != "" {
		identity.Channel, identity.ChannelSource = from, probeOptionFrom
	}
	return phoneProbeMessage{
		MessageSid: "SMAPS000000000000000000000000000000",
		From:       identity.Sender,
		To:         identity.Channel,
		Body:       probeText,
	}, identity
}

type whatsAppCloudProbe struct {
	Object string               `json:"object"`
	Entry  []whatsAppProbeEntry `json:"entry"`
}

type whatsAppProbeEntry struct {
	ID      string                `json:"id"`
	Changes []whatsAppProbeChange `json:"changes"`
}

type whatsAppProbeChange struct {
	Field string             `json:"field"`
	Value whatsAppProbeValue `json:"value"`
}

type whatsAppProbeValue struct {
	MessagingProduct string                  `json:"messaging_product"`
	Metadata         whatsAppProbeMetadata   `json:"metadata"`
	Contacts         []whatsAppProbeContact  `json:"contacts"`
	Messages         []whatsAppProbeTextData `json:"messages"`
}

type whatsAppProbeMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type whatsAppProbeContact struct {
	Profile whatsAppProbeProfile `json:"profile"`
	WaID    string               `json:"wa_id"`
}

type whatsAppProbeProfile struct {
	Name string `json:"name"`
}

type whatsAppProbeTextData struct {
	From      string            `json:"from"`
	ID        string            `json:"id"`
	Timestamp int64             `json:"timestamp"`
	Type      string            `json:"type"`
	Text      whatsAppProbeText `json:"text"`
}

type whatsAppProbeText struct {
	Body string `json:"body"`
}

func syntheticWhatsAppCloudProbe(options map[string]string) (whatsAppCloudProbe, SyntheticProbeIdentity) {
	identity := SyntheticProbeIdentity{
		Sender: "15550100001", SenderSource: probeIdentitySourceSynthetic,
		Channel: "123456789012345", ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstCSVOption(options, probeOptionAllowedNumbers); allowed != "" {
		identity.Sender, identity.SenderSource = allowed, probeOptionAllowedNumbers
	}
	if phoneNumberID := strings.TrimSpace(options[probeOptionPhoneNumberID]); phoneNumberID != "" {
		identity.Channel, identity.ChannelSource = phoneNumberID, probeOptionPhoneNumberID
	}
	displayNumber := strings.TrimSpace(options[probeOptionFrom])
	if displayNumber == "" {
		displayNumber = "+15550100002"
	}
	return whatsAppCloudProbe{
		Object: "whatsapp_business_account",
		Entry: []whatsAppProbeEntry{{
			ID: "123456789000000",
			Changes: []whatsAppProbeChange{{
				Field: "messages",
				Value: whatsAppProbeValue{
					MessagingProduct: probeAdapterWhatsApp,
					Metadata:         whatsAppProbeMetadata{DisplayPhoneNumber: displayNumber, PhoneNumberID: identity.Channel},
					Contacts:         []whatsAppProbeContact{{Profile: whatsAppProbeProfile{Name: probeSenderName}, WaID: identity.Sender}},
					Messages: []whatsAppProbeTextData{{
						From:      identity.Sender,
						ID:        "wamid.APS000000000000000000000000000001",
						Timestamp: time.Now().Unix(),
						Type:      probeWhatsAppTextType,
						Text:      whatsAppProbeText{Body: probeText},
					}},
				},
			}},
		}},
	}, identity
}

type emailProbeMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// syntheticEmailProbe impersonates the first literal (non-glob) entry of
// allowed_senders so an address-allowlisted email service accepts the
// probe; glob-only allowlists fall back to the synthetic sender, which a
// `*@domain` pattern cannot match, so the 403 stays honest.
func syntheticEmailProbe(options map[string]string) (emailProbeMessage, SyntheticProbeIdentity) {
	identity := SyntheticProbeIdentity{
		Sender: "aps@example.com", SenderSource: probeIdentitySourceSynthetic,
		Channel: "inbox@example.com", ChannelSource: probeIdentitySourceSynthetic,
	}
	if allowed := firstLiteralCSVOption(options, OptionAllowedSenders); allowed != "" {
		identity.Sender, identity.SenderSource = allowed, OptionAllowedSenders
	}
	if from := strings.TrimSpace(options[probeOptionFrom]); from != "" {
		identity.Channel, identity.ChannelSource = from, probeOptionFrom
	}
	return emailProbeMessage{
		From:    identity.Sender,
		To:      identity.Channel,
		Subject: probeText,
		Body:    probeText,
	}, identity
}

// firstLiteralCSVOption is firstCSVOption restricted to entries without
// glob metacharacters, for allowlists that accept patterns.
func firstLiteralCSVOption(options map[string]string, key string) string {
	if options == nil {
		return ""
	}
	for _, part := range strings.Split(options[key], ",") {
		if part = strings.TrimSpace(part); part != "" && !strings.Contains(part, "*") {
			return part
		}
	}
	return ""
}

// firstCSVOption returns the first non-empty comma-separated entry of a
// service option, mirroring how the messenger validator reads allowlists.
func firstCSVOption(options map[string]string, key string) string {
	if options == nil {
		return ""
	}
	for _, part := range strings.Split(options[key], ",") {
		if part = strings.TrimSpace(part); part != "" {
			return part
		}
	}
	return ""
}
