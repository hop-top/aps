package messenger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
)

const defaultDiscordAPIBaseURL = "https://discord.com/api/v10"

type DiscordHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type DiscordProviderConfig struct {
	BotToken    string
	BotTokenEnv string
	APIBaseURL  string
	Client      DiscordHTTPDoer
	Normalizer  *Normalizer
}

type DiscordProvider struct {
	botToken   string
	tokenEnv   string
	apiBaseURL string
	client     DiscordHTTPDoer
	normalizer *Normalizer
}

var _ msgtypes.MessageProvider = (*DiscordProvider)(nil)
var _ msgtypes.ProviderDelivery = (*DiscordProvider)(nil)

func NewDiscordProvider(config DiscordProviderConfig) *DiscordProvider {
	apiBaseURL := strings.TrimRight(strings.TrimSpace(config.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultDiscordAPIBaseURL
	}
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	normalizer := config.Normalizer
	if normalizer == nil {
		normalizer = NewNormalizer()
	}
	return &DiscordProvider{
		botToken:   strings.TrimSpace(config.BotToken),
		tokenEnv:   strings.TrimSpace(config.BotTokenEnv),
		apiBaseURL: apiBaseURL,
		client:     client,
		normalizer: normalizer,
	}
}

func (p *DiscordProvider) Metadata() msgtypes.ProviderRuntimeMetadata {
	return msgtypes.ProviderRuntimeMetadata{
		Provider:            string(msgtypes.PlatformDiscord),
		DisplayName:         "Discord",
		IngressModes:        []msgtypes.IngressMode{msgtypes.IngressModeWebhook, msgtypes.IngressModeStream},
		DeliveryModes:       []msgtypes.DeliveryMode{msgtypes.DeliveryModeText, msgtypes.DeliveryModeFile},
		SupportsThreads:     true,
		SupportsAttachments: true,
		SupportsReactions:   false,
	}
}

func (p *DiscordProvider) NormalizeIngress(ctx context.Context, ingress msgtypes.NativeIngress) (*msgtypes.NormalizedMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(ingress.Body, &raw); err != nil {
		return nil, fmt.Errorf("invalid Discord JSON body: %w", err)
	}
	msg, err := p.normalizer.Normalize(string(msgtypes.PlatformDiscord), raw)
	if err != nil {
		return nil, err
	}
	if msg.PlatformMetadata == nil {
		msg.PlatformMetadata = map[string]any{}
	}
	if ingress.ServiceID != "" {
		msg.PlatformMetadata["service_id"] = ingress.ServiceID
	}
	return msg, nil
}

func (p *DiscordProvider) DeliverMessage(ctx context.Context, delivery msgtypes.DeliveryRequest) (*msgtypes.DeliveryReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := delivery.Validate(); err != nil {
		return nil, err
	}
	token := p.authorizationToken()
	if token == "" {
		return nil, msgtypes.ErrMissingSecret("DISCORD_BOT_TOKEN")
	}

	channelID := discordDeliveryChannelID(delivery)
	payload := discordDeliveryPayload(delivery)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode Discord message payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBaseURL+"/channels/"+channelID+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", discordAuthorizationHeader(token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aps-discord-message-service")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &msgtypes.DeliveryReceipt{
			Provider:    string(msgtypes.PlatformDiscord),
			Status:      "failed",
			Retriable:   resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
			DeliveredAt: time.Now().UTC(),
			ProviderData: map[string]any{
				"status_code": resp.StatusCode,
				"body":        string(respBody),
			},
		}, fmt.Errorf("Discord API returned HTTP %d", resp.StatusCode)
	}

	var decoded map[string]any
	_ = json.Unmarshal(respBody, &decoded)
	return &msgtypes.DeliveryReceipt{
		Provider:    string(msgtypes.PlatformDiscord),
		DeliveryID:  getString(decoded, "id"),
		Status:      "success",
		DeliveredAt: time.Now().UTC(),
		ProviderData: map[string]any{
			"channel_id": getString(decoded, "channel_id"),
		},
	}, nil
}

func (p *DiscordProvider) authorizationToken() string {
	return firstNonEmpty(p.botToken, os.Getenv(p.tokenEnv))
}

func discordDeliveryChannelID(delivery msgtypes.DeliveryRequest) string {
	if channelID := discordMetadataString(delivery.Metadata, "discord_target_channel_id"); channelID != "" {
		return channelID
	}
	if delivery.ThreadID != "" && strings.EqualFold(discordMetadataString(delivery.Metadata, "thread_type"), msgtypes.ThreadTypeTopic) {
		return delivery.ThreadID
	}
	return strings.TrimSpace(delivery.ChannelID)
}

func discordDeliveryPayload(delivery msgtypes.DeliveryRequest) map[string]any {
	payload := map[string]any{
		"content": strings.TrimSpace(delivery.Text),
		"allowed_mentions": map[string]any{
			"parse": []string{},
		},
	}
	if replyID := discordReplyMessageID(delivery); replyID != "" {
		payload["message_reference"] = map[string]any{
			"message_id":         replyID,
			"channel_id":         delivery.ChannelID,
			"fail_if_not_exists": false,
		}
	}
	if embeds := discordAttachmentEmbeds(delivery.Attachments); len(embeds) > 0 {
		payload["embeds"] = embeds
	}
	return payload
}

func discordReplyMessageID(delivery msgtypes.DeliveryRequest) string {
	if replyID := discordMetadataString(delivery.Metadata, "discord_reply_to_message_id"); replyID != "" {
		return replyID
	}
	if replyID := discordMetadataString(delivery.Metadata, "reply_to_message_id"); replyID != "" {
		return replyID
	}
	if delivery.ThreadID != "" && !strings.EqualFold(discordMetadataString(delivery.Metadata, "thread_type"), msgtypes.ThreadTypeTopic) {
		return delivery.ThreadID
	}
	return ""
}

func discordAttachmentEmbeds(attachments []msgtypes.Attachment) []map[string]any {
	embeds := make([]map[string]any, 0, len(attachments))
	for _, attachment := range attachments {
		if strings.TrimSpace(attachment.URL) == "" {
			continue
		}
		embed := map[string]any{"url": attachment.URL}
		if attachment.Type == "image" {
			embed["image"] = map[string]any{"url": attachment.URL}
		}
		if attachment.MimeType != "" {
			embed["description"] = attachment.MimeType
		}
		embeds = append(embeds, embed)
		if len(embeds) == 10 {
			break
		}
	}
	return embeds
}

func discordAuthorizationHeader(token string) string {
	if strings.HasPrefix(strings.TrimSpace(token), "Bot ") {
		return strings.TrimSpace(token)
	}
	return "Bot " + strings.TrimSpace(token)
}

func discordMetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
