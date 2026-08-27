package messenger

import "sort"

// PlatformMeta is the single source of truth for messenger platform
// presentation: the display name and the descriptive columns the docs
// render. Capability facts (ingress/delivery modes, thread, attachment,
// and reaction support) stay in each provider's Metadata() and are not
// duplicated here.
type PlatformMeta struct {
	Platform       MessengerPlatform // enum value from this package
	Display        string            // platform name docs render
	Alias          string            // service alias for `aps service add --type`; empty when the platform has no message alias
	ChannelControl string            // channel/chat/number gating flags
	ChannelID      string            // expected channel ID format; feeds ChannelIDFormat
	AuthSource     string            // where the operator obtains the token or secret (docs: typical token source)
	Ingress        string            // inbound payload summary
	Normalize      string            // provider events the normalizer accepts
	Reply          string            // outbound payload shape the denormalizer emits
	Signature      string            // ingress signature/auth validation
	Support        string            // current route support through `aps serve`
	Maturity       string            // service maturity summary
	Notes          string            // extra operator notes
}

// supportJSONWebhookRoute is the shared current-support column for
// webhook-mounted message adapters.
const supportJSONWebhookRoute = "JSON webhook route through `aps serve`"

var platformMeta = []PlatformMeta{
	{
		Platform:       PlatformTelegram,
		Display:        "Telegram",
		Alias:          string(PlatformTelegram),
		ChannelControl: "`--allowed-chat`",
		ChannelID:      "Numeric chat ID (e.g., -1001234567890)",
		AuthSource:     "BotFather",
		Ingress:        "Bot API update JSON",
		Normalize:      "Bot API `message` and `edited_message` JSON",
		Reply:          "`sendMessage` JSON",
		Signature:      "Telegram secret-token header",
		Support:        supportJSONWebhookRoute,
		Maturity:       "Ready when mounted with `aps serve`",
		Notes:          "Chat IDs are numeric; groups usually start with `-100`.",
	},
	{
		Platform:       PlatformSlack,
		Display:        "Slack",
		Alias:          string(PlatformSlack),
		ChannelControl: "`--allowed-channel`",
		ChannelID:      "Alphanumeric channel ID (e.g., C01ABC2DEF)",
		AuthSource:     "Slack API dashboard",
		Ingress:        "Events API JSON",
		Normalize:      "Events API event envelope JSON",
		Reply:          "text response JSON",
		Signature:      "Slack signing secret",
		Support:        supportJSONWebhookRoute,
		Maturity:       "Ready when mounted with `aps serve`; app verification is external",
		Notes:          "Slack URL verification and app provisioning are outside `aps service add`.",
	},
	{
		Platform:       PlatformTeams,
		Display:        "Teams",
		Alias:          string(PlatformTeams),
		ChannelControl: "`--allowed-channel`",
		ChannelID:      "Bot Framework conversation ID (e.g., 19:abc123@thread.tacv2)",
		AuthSource:     "Azure Bot registration (App ID + client secret)",
		Ingress:        "Bot Framework Activity JSON",
		Normalize:      "Bot Framework `message` activity JSON",
		Reply:          "message activity JSON",
		Signature:      "Microsoft-signed Bot Framework JWT (`Authorization` header)",
		Support:        supportJSONWebhookRoute,
		Maturity:       "Ready when mounted with `aps serve`; Azure Bot registration is external",
	},
	{
		Platform:       PlatformDiscord,
		Display:        "Discord",
		Alias:          string(PlatformDiscord),
		ChannelControl: "`--allowed-channel`, `--allowed-guild`",
		ChannelID:      "Numeric channel ID (e.g., 1234567890123456789)",
		AuthSource:     "Discord Developer Portal",
		Ingress:        "Message JSON or relay",
		Normalize:      "message-create style JSON",
		Reply:          "content response JSON",
		Signature:      "Interactions Ed25519 only",
		Support:        supportJSONWebhookRoute,
		Maturity:       "Ready when mounted with `aps serve`; Gateway client is external",
		Notes:          "Discord Gateway/bot runtime is not created by `aps service add`.",
	},
	{
		Platform:  PlatformGitHub,
		Display:   "GitHub",
		ChannelID: "org/repo (e.g., myorg/myrepo)",
		Notes:     "The `github` alias resolves to a ticket service, not a message service.",
	},
	{
		Platform:       PlatformEmail,
		Display:        "Email",
		ChannelControl: "`--allowed-sender`",
		ChannelID:      "Mailbox name or email address (e.g., inbox, work@co.com)",
		AuthSource:     "your email bridge (IMAP poller, MTA hook)",
		Ingress:        "Bridge-posted `{from,to,subject,body}` JSON",
		Signature:      "Generic webhook auth (`auth_scheme` bearer/token/hmac-sha256/ed25519)",
		Support:        "JSON relay route through `aps serve`",
		Notes:          "The `email` alias resolves to the ticket adapter; the email message adapter has no alias and is addressed as `--type message --adapter email`.",
	},
	{
		Platform:       PlatformSMS,
		Display:        "SMS",
		Alias:          string(PlatformSMS),
		ChannelControl: "`--allowed-number`",
		ChannelID:      "Phone number receiving SMS (e.g., +15551234567)",
		AuthSource:     "SMS provider such as Twilio",
		Ingress:        "Twilio/generic phone JSON or form",
		Normalize:      "Twilio-style or generic phone fields in JSON/form",
		Reply:          "text response metadata",
		Signature:      "Twilio signature when provider is `twilio`",
		Support:        "JSON relay route through `aps serve`",
		Maturity:       "Ready for Twilio or JSON relays",
		Notes:          "Twilio signatures require exact public `--webhook-url`.",
	},
	{
		Platform:       PlatformWhatsApp,
		Display:        "WhatsApp",
		Alias:          string(PlatformWhatsApp),
		ChannelControl: "`--allowed-number`, `--phone-number-id`",
		ChannelID:      "WhatsApp phone number ID or receiving number (e.g., 123456789012345)",
		AuthSource:     "WhatsApp Cloud API or Twilio",
		Ingress:        "Cloud API JSON or Twilio-style form/JSON",
		Normalize:      "Cloud API JSON or Twilio-style WhatsApp JSON/form",
		Reply:          "text/template response metadata",
		Signature:      "Cloud `X-Hub-Signature-256`; Twilio signature when provider is `twilio`",
		Support:        "JSON webhook/relay route through `aps serve`",
		Maturity:       "Ready for Cloud API and Twilio-compatible relays",
		Notes:          "Use `--phone-number-id` for Cloud API channel IDs.",
	},
}

// AllPlatformMeta returns every platform metadata row, sorted by platform.
func AllPlatformMeta() []PlatformMeta {
	out := make([]PlatformMeta, len(platformMeta))
	copy(out, platformMeta)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Platform < out[j].Platform
	})
	return out
}

// PlatformMetaFor returns the metadata row for a platform.
func PlatformMetaFor(platform MessengerPlatform) (PlatformMeta, bool) {
	for _, row := range platformMeta {
		if row.Platform == platform {
			return row, true
		}
	}
	return PlatformMeta{}, false
}

// PlatformDisplay returns the display name for a platform, empty when
// unknown.
func PlatformDisplay(platform MessengerPlatform) string {
	row, ok := PlatformMetaFor(platform)
	if !ok {
		return ""
	}
	return row.Display
}

// ChannelIDFormat documents the expected channel ID format per platform,
// derived from the platform metadata table.
var ChannelIDFormat = func() map[MessengerPlatform]string {
	out := make(map[MessengerPlatform]string, len(platformMeta))
	for _, row := range platformMeta {
		out[row.Platform] = row.ChannelID
	}
	return out
}()
