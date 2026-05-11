package messenger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	coremessenger "hop.top/aps/internal/core/messenger"
)

type SlackProviderConfig struct {
	BotToken   string
	BaseURL    string
	Transport  SlackTransport
	Normalizer *Normalizer
	Now        func() time.Time
}

type SlackProvider struct {
	botToken   string
	transport  SlackTransport
	normalizer *Normalizer
	now        func() time.Time
}

var _ coremessenger.MessageProvider = (*SlackProvider)(nil)
var _ coremessenger.ProviderDelivery = (*SlackProvider)(nil)

func NewSlackProvider(config SlackProviderConfig) *SlackProvider {
	transport := config.Transport
	if transport == nil {
		transport = &SlackHTTPTransport{
			BaseURL:    config.BaseURL,
			HTTPClient: http.DefaultClient,
		}
	}
	normalizer := config.Normalizer
	if normalizer == nil {
		normalizer = NewNormalizer()
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &SlackProvider{
		botToken:   strings.TrimSpace(config.BotToken),
		transport:  transport,
		normalizer: normalizer,
		now:        now,
	}
}

func (p *SlackProvider) Metadata() coremessenger.ProviderRuntimeMetadata {
	return coremessenger.ProviderRuntimeMetadata{
		Provider:            string(coremessenger.PlatformSlack),
		DisplayName:         "Slack",
		IngressModes:        []coremessenger.IngressMode{coremessenger.IngressModeWebhook},
		DeliveryModes:       []coremessenger.DeliveryMode{coremessenger.DeliveryModeText, coremessenger.DeliveryModeFile},
		SupportsThreads:     true,
		SupportsAttachments: true,
		SupportsReactions:   false,
	}
}

func (p *SlackProvider) NormalizeIngress(ctx context.Context, ingress coremessenger.NativeIngress) (*coremessenger.NormalizedMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(ingress.Body, &raw); err != nil {
		return nil, fmt.Errorf("invalid Slack Events API JSON: %w", err)
	}
	msg, err := p.normalizer.Normalize(string(coremessenger.PlatformSlack), raw)
	if err != nil {
		return nil, err
	}
	if msg.PlatformMetadata == nil {
		msg.PlatformMetadata = map[string]any{}
	}
	if ingress.ServiceID != "" {
		msg.PlatformMetadata["service_id"] = ingress.ServiceID
		msg.PlatformMetadata["messenger_name"] = ingress.ServiceID
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = p.now().UTC()
	}
	return msg, nil
}

func (p *SlackProvider) DeliverMessage(ctx context.Context, delivery coremessenger.DeliveryRequest) (*coremessenger.DeliveryReceipt, error) {
	if strings.TrimSpace(p.botToken) == "" {
		return nil, coremessenger.ErrMissingSecret("SLACK_BOT_TOKEN")
	}
	if err := delivery.Validate(); err != nil {
		return nil, err
	}
	req := SlackPostMessageRequest{
		Channel: delivery.ChannelID,
		Text:    delivery.Text,
	}
	if delivery.ThreadID != "" {
		req.ThreadTS = delivery.ThreadID
	}
	if value := metadataString(delivery.Metadata, "slack_thread_ts"); value != "" {
		req.ThreadTS = value
	}
	if value := metadataString(delivery.Metadata, "slack_channel_id"); value != "" {
		req.Channel = value
	}

	resp, err := p.transport.PostMessage(ctx, p.botToken, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("Slack chat.postMessage returned nil response")
	}
	if !resp.OK {
		description := strings.TrimSpace(resp.Error)
		if description == "" {
			description = "Slack chat.postMessage failed"
		}
		return nil, errors.New(description)
	}

	return &coremessenger.DeliveryReceipt{
		Provider:    string(coremessenger.PlatformSlack),
		DeliveryID:  resp.TS,
		Status:      "success",
		DeliveredAt: p.now().UTC(),
		ProviderData: map[string]any{
			"channel":   resp.Channel,
			"thread_ts": req.ThreadTS,
		},
	}, nil
}

type SlackTransport interface {
	PostMessage(ctx context.Context, botToken string, req SlackPostMessageRequest) (*SlackPostMessageResponse, error)
}

type SlackHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type SlackHTTPTransport struct {
	BaseURL    string
	HTTPClient SlackHTTPDoer
}

func (t *SlackHTTPTransport) PostMessage(ctx context.Context, botToken string, payload SlackPostMessageRequest) (*SlackPostMessageResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(t.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://slack.com/api"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(botToken))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "aps-slack-message-service")
	client := t.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var apiResp SlackPostMessageResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("Slack chat.postMessage response decode failed: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if apiResp.Error != "" {
			return &apiResp, nil
		}
		return nil, fmt.Errorf("Slack chat.postMessage returned HTTP %d", resp.StatusCode)
	}
	return &apiResp, nil
}

type SlackPostMessageRequest struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

type SlackPostMessageResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Channel string `json:"channel,omitempty"`
	TS      string `json:"ts,omitempty"`
}
