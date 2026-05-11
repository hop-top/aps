package messenger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	WhatsAppSignatureHeader = "X-Hub-Signature-256"
	defaultWhatsAppBaseURL  = "https://graph.facebook.com/v19.0"
)

// WhatsAppAuthHook maps WhatsApp Cloud App Secret signatures onto the shared
// service validator. Verification challenges are handled separately because
// Meta sends them as GET query parameters, not signed POST bodies.
type WhatsAppAuthHook struct{}

func (WhatsAppAuthHook) AuthRequirements(service ServiceValidationConfig) AuthRequirements {
	opts := service.Options
	if len(opts) == 0 {
		return AuthRequirements{}
	}
	secret := firstConfigured(opts["app_secret"], opts["signature_secret"], serviceEnvLiteral(service.Env, "WHATSAPP_APP_SECRET"))
	secretEnv := firstConfigured(
		opts["app_secret_env"],
		opts["signature_secret_env"],
		opts["signing_secret_env"],
		serviceEnvSecretName(service.Env, "WHATSAPP_APP_SECRET"),
	)
	if secret == "" && secretEnv == "" {
		return AuthRequirements{}
	}
	return AuthRequirements{
		Scheme:             AuthSchemeHMACSHA256,
		Header:             WhatsAppSignatureHeader,
		SignatureSecret:    secret,
		SignatureSecretEnv: secretEnv,
	}
}

// WhatsAppVerificationChallenge validates a WhatsApp Cloud webhook verification
// query and returns the challenge body that Meta expects APS to echo.
func WhatsAppVerificationChallenge(service ServiceValidationConfig, query url.Values) (string, error) {
	if strings.TrimSpace(query.Get("hub.mode")) != "subscribe" {
		return "", ErrAuthFailed(service.ID, "invalid WhatsApp verification mode")
	}
	challenge := strings.TrimSpace(query.Get("hub.challenge"))
	if challenge == "" {
		return "", ErrAuthFailed(service.ID, "missing WhatsApp verification challenge")
	}
	expected := whatsappVerifyToken(service)
	if expected == "" {
		return "", ErrMissingSecret("WHATSAPP_VERIFY_TOKEN")
	}
	got := strings.TrimSpace(query.Get("hub.verify_token"))
	if got == "" || got != expected {
		return "", ErrAuthFailed(service.ID, "invalid WhatsApp verification token")
	}
	return challenge, nil
}

func whatsappVerifyToken(service ServiceValidationConfig) string {
	return firstConfigured(
		service.Options["verify_token"],
		service.Options["webhook_verify_token"],
		getenv(service.Options["verify_token_env"]),
		getenv(service.Options["webhook_verify_token_env"]),
		serviceEnvLiteral(service.Env, "WHATSAPP_VERIFY_TOKEN"),
		getenv(serviceEnvSecretName(service.Env, "WHATSAPP_VERIFY_TOKEN")),
		getenv("WHATSAPP_VERIFY_TOKEN"),
	)
}

type WhatsAppProvider struct {
	Config    WhatsAppProviderConfig
	Transport WhatsAppTransport
	Now       func() time.Time
}

type WhatsAppProviderConfig struct {
	ServiceID       string
	Provider        string
	AccessToken     string
	PhoneNumberID   string
	From            string
	AccountSID      string
	AuthToken       string
	BaseURL         string
	TemplateName    string
	LanguageCode    string
	RequireTemplate bool
}

type WhatsAppDelivery struct {
	Provider      string
	AccessToken   string
	PhoneNumberID string
	AccountSID    string
	AuthToken     string
	From          string
	To            string
	Body          string
	TemplateName  string
	LanguageCode  string
	Attachments   []Attachment
	Metadata      map[string]any
}

type WhatsAppDeliveryResult struct {
	MessageID string
	Status    string
	Provider  string
	Retriable bool
	Raw       map[string]any
}

type WhatsAppTransport interface {
	SendWhatsApp(context.Context, WhatsAppDelivery) (*WhatsAppDeliveryResult, error)
}

func NewWhatsAppProvider(config WhatsAppProviderConfig, transport WhatsAppTransport) *WhatsAppProvider {
	if config.Provider == "" {
		config.Provider = "whatsapp-cloud"
	}
	return &WhatsAppProvider{Config: config, Transport: transport, Now: func() time.Time { return time.Now().UTC() }}
}

func (p *WhatsAppProvider) Metadata() ProviderRuntimeMetadata {
	return ProviderRuntimeMetadata{
		Provider:            string(PlatformWhatsApp),
		DisplayName:         "WhatsApp",
		IngressModes:        []IngressMode{IngressModeWebhook},
		DeliveryModes:       []DeliveryMode{DeliveryModeText, DeliveryModeFile},
		SupportsThreads:     true,
		SupportsAttachments: true,
	}
}

func (p *WhatsAppProvider) NormalizeIngress(_ context.Context, ingress NativeIngress) (*NormalizedMessage, error) {
	payload, err := ParseWhatsAppPayload(ingress.Headers, ingress.Body)
	if err != nil {
		return nil, ErrNormalizeFailed(string(PlatformWhatsApp), err)
	}
	msg, err := NormalizeWhatsAppPayload(payload, p.now())
	if err != nil {
		return nil, ErrNormalizeFailed(string(PlatformWhatsApp), err)
	}
	if msg.PlatformMetadata == nil {
		msg.PlatformMetadata = map[string]any{}
	}
	msg.PlatformMetadata["service_id"] = firstConfigured(ingress.ServiceID, p.Config.ServiceID)
	msg.PlatformMetadata["messenger_name"] = firstConfigured(ingress.ServiceID, p.Config.ServiceID)
	msg.PlatformMetadata["provider"] = firstConfigured(p.Config.Provider, ingress.Provider, "whatsapp-cloud")
	return msg, nil
}

func (p *WhatsAppProvider) DeliverMessage(ctx context.Context, delivery DeliveryRequest) (*DeliveryReceipt, error) {
	if err := delivery.Validate(); err != nil {
		return nil, err
	}
	provider := strings.ToLower(firstConfigured(metadataString(delivery.Metadata, "provider"), p.Config.Provider, "whatsapp-cloud"))
	templateName := firstConfigured(metadataString(delivery.Metadata, "template_name"), p.Config.TemplateName)
	languageCode := firstConfigured(metadataString(delivery.Metadata, "language_code"), p.Config.LanguageCode, "en_US")
	if p.Config.RequireTemplate && templateName == "" {
		return nil, fmt.Errorf("whatsapp template_name is required when template_required is enabled")
	}
	if templateName != "" && delivery.Text == "" && len(delivery.Attachments) > 0 {
		return nil, fmt.Errorf("whatsapp template delivery cannot use attachments")
	}

	transport := p.Transport
	if transport == nil {
		if provider == "twilio" {
			transport = NewTwilioWhatsAppTransport(p.Config.AccountSID, p.Config.AuthToken, nil)
		} else {
			transport = NewWhatsAppCloudTransport(p.Config.AccessToken, p.Config.BaseURL, nil)
		}
	}
	to := firstConfigured(metadataString(delivery.Metadata, "to"), metadataString(delivery.Metadata, "recipient"), delivery.ChannelID)
	phoneNumberID := firstConfigured(metadataString(delivery.Metadata, "phone_number_id"), p.Config.PhoneNumberID, p.Config.From)
	from := firstConfigured(metadataString(delivery.Metadata, "from"), p.Config.From, p.Config.PhoneNumberID)
	if to == "" {
		return nil, fmt.Errorf("whatsapp delivery recipient is required")
	}
	if provider == "twilio" && from == "" {
		return nil, fmt.Errorf("whatsapp Twilio delivery sender is required")
	}
	if provider != "twilio" && phoneNumberID == "" {
		return nil, fmt.Errorf("whatsapp phone_number_id is required")
	}

	result, err := transport.SendWhatsApp(ctx, WhatsAppDelivery{
		Provider:      provider,
		AccessToken:   p.Config.AccessToken,
		PhoneNumberID: phoneNumberID,
		AccountSID:    p.Config.AccountSID,
		AuthToken:     p.Config.AuthToken,
		From:          from,
		To:            to,
		Body:          delivery.Text,
		TemplateName:  templateName,
		LanguageCode:  languageCode,
		Attachments:   delivery.Attachments,
		Metadata:      delivery.Metadata,
	})
	if err != nil {
		return nil, err
	}
	return &DeliveryReceipt{
		Provider:     firstConfigured(result.Provider, delivery.Provider, p.Config.Provider, string(PlatformWhatsApp)),
		DeliveryID:   result.MessageID,
		Status:       firstConfigured(result.Status, "sent"),
		Retriable:    result.Retriable,
		DeliveredAt:  p.now(),
		ProviderData: result.Raw,
	}, nil
}

func (p *WhatsAppProvider) now() time.Time {
	if p != nil && p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

func ParseWhatsAppPayload(headers map[string][]string, body []byte) (map[string]any, error) {
	contentType := ""
	if headers != nil {
		contentType = http.Header(headers).Get("Content-Type")
	}
	if strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("invalid whatsapp form body: %w", err)
		}
		return valuesToMap(values), nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid whatsapp JSON body: %w", err)
	}
	return payload, nil
}

func NormalizeWhatsAppPayload(payload map[string]any, receivedAt time.Time) (*NormalizedMessage, error) {
	if entry, ok := firstPayloadMap(payload, "entry"); ok {
		return normalizeWhatsAppCloudPayload(payload, entry, receivedAt)
	}
	return normalizeTwilioWhatsAppPayload(payload, receivedAt)
}

func normalizeTwilioWhatsAppPayload(payload map[string]any, receivedAt time.Time) (*NormalizedMessage, error) {
	from := firstNonEmptyPayload(payload, "From", "from")
	to := firstNonEmptyPayload(payload, "To", "to")
	if from == "" {
		return nil, fmt.Errorf("missing from field in whatsapp event")
	}
	if to == "" {
		return nil, fmt.Errorf("missing to field in whatsapp event")
	}
	text := firstNonEmptyPayload(payload, "Body", "body", "text")
	attachments := smsAttachments(payload)
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return nil, fmt.Errorf("whatsapp event requires Body, body, text, or media")
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	id := firstNonEmptyPayload(payload, "MessageSid", "SmsSid", "WaId", "message_id")
	if id == "" {
		id = fmt.Sprintf("whatsapp_%d", receivedAt.UnixNano())
	}
	return &NormalizedMessage{
		ID:          id,
		Timestamp:   receivedAt.UTC(),
		Platform:    string(PlatformWhatsApp),
		Sender:      Sender{ID: from, Name: from, PlatformHandle: from, PlatformID: from},
		Channel:     Channel{ID: to, Name: to, Type: ChannelTypeDirect, PlatformID: to},
		Text:        text,
		Attachments: attachments,
		PlatformMetadata: map[string]any{
			"provider":    "twilio",
			"account_sid": firstNonEmptyPayload(payload, "AccountSid", "account_sid"),
			"from":        from,
			"to":          to,
		},
	}, nil
}

func normalizeWhatsAppCloudPayload(raw, entry map[string]any, receivedAt time.Time) (*NormalizedMessage, error) {
	change, ok := firstPayloadMap(entry, "changes")
	if !ok {
		return nil, fmt.Errorf("missing changes in whatsapp event")
	}
	value, ok := payloadMap(change, "value")
	if !ok {
		return nil, fmt.Errorf("missing value in whatsapp change")
	}
	message, ok := firstPayloadMap(value, "messages")
	if !ok {
		return nil, fmt.Errorf("missing messages in whatsapp value")
	}
	from := payloadString(message, "from")
	if from == "" {
		return nil, fmt.Errorf("missing from in whatsapp message")
	}
	metadata, _ := payloadMap(value, "metadata")
	channelID := firstConfigured(payloadString(metadata, "phone_number_id"), payloadString(metadata, "display_phone_number"))
	if channelID == "" {
		return nil, fmt.Errorf("missing phone_number_id in whatsapp metadata")
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	if ts := parseWhatsAppUnix(payloadString(message, "timestamp")); !ts.IsZero() {
		receivedAt = ts
	}

	text := ""
	if textMap, ok := payloadMap(message, "text"); ok {
		text = payloadString(textMap, "body")
	}
	attachments := whatsappCloudAttachments(message)
	if text == "" && len(attachments) > 0 {
		if media, ok := payloadMap(message, payloadString(message, "type")); ok {
			text = firstConfigured(payloadString(media, "caption"), payloadString(message, "type"))
		}
	}
	contact, _ := firstPayloadMap(value, "contacts")
	profile, _ := payloadMap(contact, "profile")

	msg := &NormalizedMessage{
		ID:        firstConfigured(payloadString(message, "id"), fmt.Sprintf("whatsapp_%d", receivedAt.UnixNano())),
		Timestamp: receivedAt.UTC(),
		Platform:  string(PlatformWhatsApp),
		Sender: Sender{
			ID:             from,
			Name:           payloadString(profile, "name"),
			PlatformHandle: from,
			PlatformID:     from,
		},
		Channel: Channel{
			ID:         channelID,
			Name:       payloadString(metadata, "display_phone_number"),
			Type:       ChannelTypeDirect,
			PlatformID: channelID,
		},
		Text:             text,
		Attachments:      attachments,
		PlatformMetadata: raw,
	}
	if contextMap, ok := payloadMap(message, "context"); ok {
		if contextID := payloadString(contextMap, "id"); contextID != "" {
			msg.Thread = &Thread{ID: contextID, Type: ThreadTypeReply}
		}
	}
	return msg, nil
}

type WhatsAppCloudTransport struct {
	AccessToken string
	BaseURL     string
	Client      *http.Client
}

func NewWhatsAppCloudTransport(accessToken, baseURL string, client *http.Client) *WhatsAppCloudTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &WhatsAppCloudTransport{AccessToken: accessToken, BaseURL: baseURL, Client: client}
}

func (t *WhatsAppCloudTransport) SendWhatsApp(ctx context.Context, delivery WhatsAppDelivery) (*WhatsAppDeliveryResult, error) {
	token := firstConfigured(delivery.AccessToken, t.AccessToken)
	if token == "" {
		return nil, ErrMissingSecret("WHATSAPP_ACCESS_TOKEN")
	}
	if delivery.PhoneNumberID == "" {
		return nil, fmt.Errorf("whatsapp phone_number_id is required")
	}
	if delivery.To == "" {
		return nil, fmt.Errorf("whatsapp delivery recipient is required")
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                stripWhatsAppAddressPrefix(delivery.To),
	}
	if delivery.TemplateName != "" {
		payload["type"] = "template"
		payload["template"] = map[string]any{
			"name": delivery.TemplateName,
			"language": map[string]any{
				"code": firstConfigured(delivery.LanguageCode, "en_US"),
			},
		}
	} else {
		if strings.TrimSpace(delivery.Body) == "" {
			return nil, fmt.Errorf("whatsapp text body is required")
		}
		payload["type"] = "text"
		payload["text"] = map[string]any{
			"preview_url": false,
			"body":        delivery.Body,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(firstConfigured(t.BaseURL, defaultWhatsAppBaseURL), "/")
	endpoint := baseURL + "/" + url.PathEscape(delivery.PhoneNumberID) + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aps-whatsapp-message-service")
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	parsed := map[string]any{}
	_ = json.Unmarshal(respBody, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := firstConfigured(whatsappAPIErrorMessage(parsed), strings.TrimSpace(string(respBody)), resp.Status)
		return nil, &WhatsAppDeliveryError{StatusCode: resp.StatusCode, Message: msg, Retriable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	return &WhatsAppDeliveryResult{
		MessageID: firstWhatsAppMessageID(parsed),
		Status:    "sent",
		Provider:  "whatsapp-cloud",
		Raw:       parsed,
	}, nil
}

type TwilioWhatsAppTransport struct {
	AccountSID string
	AuthToken  string
	BaseURL    string
	Client     *http.Client
}

func NewTwilioWhatsAppTransport(accountSID, authToken string, client *http.Client) *TwilioWhatsAppTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &TwilioWhatsAppTransport{AccountSID: accountSID, AuthToken: authToken, BaseURL: defaultTwilioBaseURL, Client: client}
}

func (t *TwilioWhatsAppTransport) SendWhatsApp(ctx context.Context, delivery WhatsAppDelivery) (*WhatsAppDeliveryResult, error) {
	accountSID := firstConfigured(delivery.AccountSID, t.AccountSID)
	authToken := firstConfigured(delivery.AuthToken, t.AuthToken)
	if accountSID == "" {
		return nil, fmt.Errorf("TWILIO_ACCOUNT_SID is required")
	}
	if authToken == "" {
		return nil, ErrMissingSecret("TWILIO_AUTH_TOKEN")
	}
	if delivery.From == "" {
		return nil, fmt.Errorf("whatsapp Twilio delivery sender is required")
	}
	if delivery.To == "" {
		return nil, fmt.Errorf("whatsapp delivery recipient is required")
	}
	if strings.TrimSpace(delivery.Body) == "" && metadataString(delivery.Metadata, "content_sid") == "" && len(delivery.Attachments) == 0 {
		return nil, fmt.Errorf("whatsapp delivery body is required")
	}
	baseURL := strings.TrimRight(firstConfigured(t.BaseURL, defaultTwilioBaseURL), "/")
	endpoint := baseURL + "/2010-04-01/Accounts/" + url.PathEscape(accountSID) + "/Messages.json"
	form := url.Values{}
	form.Set("From", ensureWhatsAppAddressPrefix(delivery.From))
	form.Set("To", ensureWhatsAppAddressPrefix(delivery.To))
	if contentSID := metadataString(delivery.Metadata, "content_sid"); contentSID != "" {
		form.Set("ContentSid", contentSID)
	} else {
		form.Set("Body", delivery.Body)
	}
	if len(delivery.Attachments) > 0 && delivery.Attachments[0].URL != "" {
		form.Set("MediaUrl", delivery.Attachments[0].URL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(accountSID, authToken)
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	parsed := map[string]any{}
	_ = json.Unmarshal(respBody, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := firstConfigured(mapString(parsed, "message"), strings.TrimSpace(string(respBody)), resp.Status)
		return nil, &WhatsAppDeliveryError{StatusCode: resp.StatusCode, Message: msg, Retriable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	return &WhatsAppDeliveryResult{
		MessageID: mapString(parsed, "sid"),
		Status:    firstConfigured(mapString(parsed, "status"), "sent"),
		Provider:  "twilio",
		Raw:       parsed,
	}, nil
}

type WhatsAppDeliveryError struct {
	StatusCode int
	Message    string
	Retriable  bool
}

func (e *WhatsAppDeliveryError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("whatsapp delivery failed: HTTP %d: %s", e.StatusCode, e.Message)
}

func firstPayloadMap(payload map[string]any, key string) (map[string]any, bool) {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil, false
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case []any:
		for _, item := range typed {
			if itemMap, ok := item.(map[string]any); ok {
				return itemMap, true
			}
		}
	}
	return nil, false
}

func payloadMap(payload map[string]any, key string) (map[string]any, bool) {
	if payload == nil {
		return nil, false
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func whatsappCloudAttachments(message map[string]any) []Attachment {
	messageType := payloadString(message, "type")
	media, ok := payloadMap(message, messageType)
	if !ok {
		return nil
	}
	switch messageType {
	case "audio", "document", "image", "sticker", "video":
	default:
		return nil
	}
	attachmentType := messageType
	if attachmentType == "document" || attachmentType == "sticker" {
		attachmentType = "file"
	}
	return []Attachment{{
		Type:     attachmentType,
		URL:      payloadString(media, "id"),
		MimeType: payloadString(media, "mime_type"),
	}}
}

func parseWhatsAppUnix(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	sec, err := parseUnixSeconds(value)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

func firstWhatsAppMessageID(parsed map[string]any) string {
	messages, ok := parsed["messages"].([]any)
	if !ok || len(messages) == 0 {
		return ""
	}
	first, ok := messages[0].(map[string]any)
	if !ok {
		return ""
	}
	return payloadString(first, "id")
}

func whatsappAPIErrorMessage(parsed map[string]any) string {
	errMap, ok := payloadMap(parsed, "error")
	if !ok {
		return ""
	}
	return firstConfigured(payloadString(errMap, "message"), payloadString(errMap, "error_data"))
}

func stripWhatsAppAddressPrefix(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "whatsapp:")
}

func ensureWhatsAppAddressPrefix(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "whatsapp:") {
		return value
	}
	return "whatsapp:" + value
}
