package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infracloudio/msbotbuilder-go/connector/auth"
	"github.com/infracloudio/msbotbuilder-go/connector/client"
	"github.com/infracloudio/msbotbuilder-go/schema"

	coremessenger "hop.top/aps/internal/core/messenger"
)

const teamsLegacyTokenURL = "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token" //nolint:gosec // G101: OAuth token endpoint URL, not a credential

// Delivery metadata keys carried from the inbound Teams activity to outbound
// connector delivery.
const (
	teamsMetaServiceURL     = "teams_service_url"
	teamsMetaConversationID = "teams_conversation_id"
	teamsMetaActivityID     = "teams_activity_id"
	teamsMetaRecipientID    = "teams_recipient_id"
)

// deliveryStatusSuccess is the receipt status every provider reports for a
// delivered message.
const deliveryStatusSuccess = "success"

// TeamsProviderConfig configures a TeamsProvider.
type TeamsProviderConfig struct {
	AppID       string
	AppPassword string
	// TenantID selects the single-tenant token endpoint for outbound
	// delivery. Empty falls back to the legacy multi-tenant endpoint, which
	// only works for bot registrations created before multi-tenant creation
	// was blocked.
	TenantID   string
	TokenURL   string
	Transport  TeamsTransport
	Normalizer *Normalizer
	Now        func() time.Time
}

// TeamsProvider normalizes inbound Microsoft Teams activities and delivers
// outbound replies through the Bot Framework connector service.
type TeamsProvider struct {
	appID        string
	appPassword  string
	transport    TeamsTransport
	transportErr error
	normalizer   *Normalizer
	now          func() time.Time
}

var (
	_ coremessenger.MessageProvider  = (*TeamsProvider)(nil)
	_ coremessenger.ProviderDelivery = (*TeamsProvider)(nil)
)

// NewTeamsProvider builds a TeamsProvider, defaulting the normalizer, clock,
// and SDK connector transport when the config leaves them unset.
func NewTeamsProvider(config TeamsProviderConfig) *TeamsProvider {
	normalizer := config.Normalizer
	if normalizer == nil {
		normalizer = NewNormalizer()
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	provider := &TeamsProvider{
		appID:       strings.TrimSpace(config.AppID),
		appPassword: strings.TrimSpace(config.AppPassword),
		transport:   config.Transport,
		normalizer:  normalizer,
		now:         now,
	}
	if provider.transport == nil {
		transport, err := NewTeamsSDKTransport(provider.appID, provider.appPassword, teamsTokenURL(config.TenantID, config.TokenURL))
		if err != nil {
			provider.transportErr = err
		} else {
			provider.transport = transport
		}
	}
	return provider
}

// Metadata describes the Teams provider's runtime capabilities.
func (p *TeamsProvider) Metadata() coremessenger.ProviderRuntimeMetadata {
	return coremessenger.ProviderRuntimeMetadata{
		Provider:            string(coremessenger.PlatformTeams),
		DisplayName:         "Microsoft Teams",
		IngressModes:        []coremessenger.IngressMode{coremessenger.IngressModeWebhook},
		DeliveryModes:       []coremessenger.DeliveryMode{coremessenger.DeliveryModeText},
		SupportsThreads:     true,
		SupportsAttachments: true,
		SupportsReactions:   false,
	}
}

// NormalizeIngress converts an inbound Bot Framework activity into the
// unified NormalizedMessage format.
func (p *TeamsProvider) NormalizeIngress(ctx context.Context, ingress coremessenger.NativeIngress) (*coremessenger.NormalizedMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("teams ingress aborted: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(ingress.Body, &raw); err != nil {
		return nil, fmt.Errorf("invalid Bot Framework activity JSON: %w", err)
	}
	msg, err := p.normalizer.Normalize(string(coremessenger.PlatformTeams), raw)
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

// DeliverMessage posts an outbound reply to the Bot Framework connector
// service identified by the delivery metadata.
func (p *TeamsProvider) DeliverMessage(ctx context.Context, delivery coremessenger.DeliveryRequest) (*coremessenger.DeliveryReceipt, error) {
	if p.appID == "" {
		return nil, coremessenger.ErrMissingSecret("TEAMS_APP_ID") //nolint:wrapcheck // typed MessengerError; handler matches on code
	}
	if p.appPassword == "" {
		return nil, coremessenger.ErrMissingSecret("TEAMS_APP_PASSWORD") //nolint:wrapcheck // typed MessengerError; handler matches on code
	}
	if err := delivery.Validate(); err != nil {
		return nil, fmt.Errorf("invalid teams delivery request: %w", err)
	}
	if p.transport == nil {
		return nil, fmt.Errorf("teams connector transport unavailable: %w", p.transportErr)
	}

	serviceURL := strings.TrimRight(metadataString(delivery.Metadata, teamsMetaServiceURL), "/")
	if serviceURL == "" {
		return nil, fmt.Errorf("teams delivery requires teams_service_url metadata from the inbound activity")
	}
	conversationID := metadataString(delivery.Metadata, teamsMetaConversationID)
	if conversationID == "" {
		conversationID = delivery.ChannelID
	}
	replyToID := metadataString(delivery.Metadata, teamsMetaActivityID)

	// replyToActivity when the inbound activity is known, else
	// sendToConversation. IDs are used verbatim, matching connector
	// service expectations.
	rawURL := fmt.Sprintf("%s/v3/conversations/%s/activities", serviceURL, conversationID)
	if replyToID != "" {
		rawURL += "/" + replyToID
	}
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Teams connector URL: %w", err)
	}

	activity := schema.Activity{
		Type:         schema.Message,
		Text:         delivery.Text,
		Conversation: schema.ConversationAccount{ID: conversationID},
		ReplyToID:    replyToID,
	}
	if botID := metadataString(delivery.Metadata, teamsMetaRecipientID); botID != "" {
		activity.From = schema.ChannelAccount{ID: botID}
	}

	if err := p.transport.PostActivity(ctx, *target, activity); err != nil {
		return nil, fmt.Errorf("teams connector post failed: %w", err)
	}

	return &coremessenger.DeliveryReceipt{
		Provider:    string(coremessenger.PlatformTeams),
		Status:      deliveryStatusSuccess,
		DeliveredAt: p.now().UTC(),
		ProviderData: map[string]any{
			teamsConversationField: conversationID,
			"service_url":          serviceURL,
			"reply_to":             replyToID,
		},
	}, nil
}

// teamsTokenURL picks the OAuth token endpoint for outbound connector calls:
// explicit override, tenant-specific endpoint, then the legacy multi-tenant
// endpoint.
func teamsTokenURL(tenantID, override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	if tenant := strings.TrimSpace(tenantID); tenant != "" {
		return "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token"
	}
	return teamsLegacyTokenURL
}

// TeamsTransport posts an outbound activity to the Bot Framework connector
// service.
type TeamsTransport interface {
	PostActivity(ctx context.Context, target url.URL, activity schema.Activity) error
}

// TeamsSDKTransport delivers through the Bot Framework SDK connector client,
// which acquires and caches the client-credentials token for the configured
// endpoint.
type TeamsSDKTransport struct {
	client client.Client
}

// NewTeamsSDKTransport builds an SDK connector transport with default HTTP
// clients.
func NewTeamsSDKTransport(appID, appPassword, tokenURL string) (*TeamsSDKTransport, error) {
	return NewTeamsSDKTransportWithConfig(TeamsSDKTransportConfig{
		AppID:       appID,
		AppPassword: appPassword,
		TokenURL:    tokenURL,
	})
}

// TeamsSDKTransportConfig configures the SDK connector transport. AuthClient
// and ReplyClient default to fresh http.Clients; tests inject recording
// clients through them.
type TeamsSDKTransportConfig struct {
	AppID       string
	AppPassword string
	TokenURL    string
	AuthClient  *http.Client
	ReplyClient *http.Client
}

// NewTeamsSDKTransportWithConfig builds an SDK connector transport from the
// full config, honoring injected auth and reply HTTP clients.
func NewTeamsSDKTransportWithConfig(cfg TeamsSDKTransportConfig) (*TeamsSDKTransport, error) {
	config, err := client.NewClientConfig(auth.SimpleCredentialProvider{
		AppID:    cfg.AppID,
		Password: cfg.AppPassword,
	}, cfg.TokenURL)
	if err != nil {
		return nil, fmt.Errorf("teams connector config: %w", err)
	}
	if cfg.AuthClient != nil {
		config.AuthClient = cfg.AuthClient
	}
	if cfg.ReplyClient != nil {
		config.ReplyClient = cfg.ReplyClient
	}
	connector, err := client.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("teams connector client: %w", err)
	}
	return &TeamsSDKTransport{client: connector}, nil
}

// PostActivity posts the activity to the connector service through the SDK
// client.
func (t *TeamsSDKTransport) PostActivity(ctx context.Context, target url.URL, activity schema.Activity) error {
	if err := t.client.Post(ctx, target, activity); err != nil {
		return fmt.Errorf("bot framework connector: %w", err)
	}
	return nil
}
