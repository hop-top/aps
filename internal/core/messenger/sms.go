package messenger

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	TwilioSignatureHeader = "X-Twilio-Signature"
	defaultTwilioBaseURL  = "https://api.twilio.com"
)

// TwilioWebhookValidator verifies Twilio-compatible SMS webhook signatures.
type TwilioWebhookValidator struct{}

func (TwilioWebhookValidator) AuthRequirements(ServiceValidationConfig) AuthRequirements {
	return AuthRequirements{}
}

func (TwilioWebhookValidator) ValidateProviderRequest(_ context.Context, input RequestValidationInput) error {
	token := twilioAuthToken(input.Service)
	if token == "" {
		return ErrMissingSecret("TWILIO_AUTH_TOKEN")
	}
	signature := strings.TrimSpace(input.Headers.Get(TwilioSignatureHeader))
	if signature == "" {
		return ErrAuthFailed(input.Service.ID, "missing Twilio signature")
	}
	requestURL := firstConfigured(input.Service.Options["webhook_url"], input.URL)
	if requestURL == "" {
		return ErrAuthFailed(input.Service.ID, "missing Twilio webhook URL")
	}
	form := input.Form
	if form == nil {
		form, _ = url.ParseQuery(string(input.Body))
	}
	if err := validateTwilioJSONBodySHA(requestURL, input.Headers, input.Body); err != nil {
		return ErrAuthFailed(input.Service.ID, err.Error())
	}
	expected := TwilioSignature(token, requestURL, form)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return ErrAuthFailed(input.Service.ID, "invalid Twilio signature")
	}
	return nil
}

func TwilioSignature(authToken, requestURL string, params url.Values) string {
	base := requestURL
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := append([]string(nil), params[key]...)
		sort.Strings(values)
		for _, value := range values {
			base += key + value
		}
	}
	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func validateTwilioJSONBodySHA(requestURL string, headers http.Header, body []byte) error {
	if !strings.Contains(strings.ToLower(headers.Get("Content-Type")), "application/json") {
		return nil
	}
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("invalid Twilio webhook URL")
	}
	expected := strings.TrimSpace(parsed.Query().Get("bodySHA256"))
	if expected == "" {
		return nil
	}
	sum := sha256.Sum256(body)
	if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(hex.EncodeToString(sum[:]))) {
		return fmt.Errorf("invalid Twilio body hash")
	}
	return nil
}

func twilioAuthToken(service ServiceValidationConfig) string {
	if service.Options == nil {
		service.Options = map[string]string{}
	}
	if service.Env == nil {
		service.Env = map[string]string{}
	}
	return firstConfigured(
		resolveConfiguredSecret(service.Options["twilio_auth_token"]),
		resolveConfiguredSecret(service.Options["auth_token"]),
		resolveConfiguredSecret(service.Options["signature_secret"]),
		getenv(service.Options["auth_token_env"]),
		getenv(service.Options["signature_secret_env"]),
		serviceEnvLiteral(service.Env, "TWILIO_AUTH_TOKEN"),
		getenv(serviceEnvSecretName(service.Env, "TWILIO_AUTH_TOKEN")),
		getenv("TWILIO_AUTH_TOKEN"),
	)
}

func resolveConfiguredSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "secret:") {
		return getenv(strings.TrimSpace(strings.TrimPrefix(value, "secret:")))
	}
	return value
}

// SMSProvider is a first-class SMS message provider with mockable delivery.
type SMSProvider struct {
	Config    SMSProviderConfig
	Transport SMSTransport
	Now       func() time.Time
}

type SMSProviderConfig struct {
	ServiceID   string
	Provider    string
	AccountSID  string
	AuthToken   string
	From        string
	CallbackURL string
}

type SMSDelivery struct {
	Provider   string
	AccountSID string
	AuthToken  string
	From       string
	To         string
	Body       string
	Metadata   map[string]any
}

type SMSDeliveryResult struct {
	MessageSID string
	Status     string
	Provider   string
	Retriable  bool
	Raw        map[string]any
}

type SMSTransport interface {
	SendSMS(context.Context, SMSDelivery) (*SMSDeliveryResult, error)
}

func NewSMSProvider(config SMSProviderConfig, transport SMSTransport) *SMSProvider {
	if config.Provider == "" {
		config.Provider = string(PlatformSMS)
	}
	return &SMSProvider{Config: config, Transport: transport, Now: func() time.Time { return time.Now().UTC() }}
}

func (p *SMSProvider) Metadata() ProviderRuntimeMetadata {
	return ProviderRuntimeMetadata{
		Provider:        firstConfigured(p.Config.Provider, string(PlatformSMS)),
		DisplayName:     "SMS",
		IngressModes:    []IngressMode{IngressModeWebhook},
		DeliveryModes:   []DeliveryMode{DeliveryModeText},
		SupportsThreads: false,
	}
}

func (p *SMSProvider) NormalizeIngress(_ context.Context, ingress NativeIngress) (*NormalizedMessage, error) {
	payload, err := ParseSMSPayload(ingress.Headers, ingress.Body)
	if err != nil {
		return nil, ErrNormalizeFailed(string(PlatformSMS), err)
	}
	msg, err := NormalizeSMSPayload(payload, p.now())
	if err != nil {
		return nil, ErrNormalizeFailed(string(PlatformSMS), err)
	}
	if msg.PlatformMetadata == nil {
		msg.PlatformMetadata = map[string]any{}
	}
	msg.PlatformMetadata["service_id"] = firstConfigured(ingress.ServiceID, p.Config.ServiceID)
	msg.PlatformMetadata["provider"] = firstConfigured(p.Config.Provider, ingress.Provider, "twilio")
	return msg, nil
}

func (p *SMSProvider) DeliverMessage(ctx context.Context, delivery DeliveryRequest) (*DeliveryReceipt, error) {
	if err := delivery.Validate(); err != nil {
		return nil, err
	}
	transport := p.Transport
	if transport == nil {
		transport = NewTwilioSMSTransport(p.Config.AccountSID, p.Config.AuthToken, nil)
	}
	to := firstConfigured(metadataString(delivery.Metadata, "to"), delivery.ChannelID)
	from := firstConfigured(metadataString(delivery.Metadata, "from"), p.Config.From)
	if to == "" {
		return nil, fmt.Errorf("sms delivery recipient is required")
	}
	if from == "" {
		return nil, fmt.Errorf("sms delivery sender is required")
	}
	result, err := transport.SendSMS(ctx, SMSDelivery{
		Provider:   firstConfigured(delivery.Provider, p.Config.Provider, "twilio"),
		AccountSID: p.Config.AccountSID,
		AuthToken:  p.Config.AuthToken,
		From:       from,
		To:         to,
		Body:       delivery.Text,
		Metadata:   delivery.Metadata,
	})
	if err != nil {
		return nil, err
	}
	receipt := &DeliveryReceipt{
		Provider:     firstConfigured(result.Provider, delivery.Provider, p.Config.Provider, "twilio"),
		DeliveryID:   result.MessageSID,
		Status:       firstConfigured(result.Status, "sent"),
		Retriable:    result.Retriable,
		DeliveredAt:  p.now(),
		ProviderData: result.Raw,
	}
	return receipt, nil
}

func (p *SMSProvider) now() time.Time {
	if p != nil && p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

func ParseSMSPayload(headers map[string][]string, body []byte) (map[string]any, error) {
	contentType := ""
	if headers != nil {
		contentType = http.Header(headers).Get("Content-Type")
	}
	if strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("invalid sms form body: %w", err)
		}
		return valuesToMap(values), nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid sms JSON body: %w", err)
	}
	return payload, nil
}

func NormalizeSMSPayload(payload map[string]any, receivedAt time.Time) (*NormalizedMessage, error) {
	from := firstNonEmptyPayload(payload, "From", "from")
	to := firstNonEmptyPayload(payload, "To", "to")
	if from == "" {
		return nil, fmt.Errorf("missing from field in sms event")
	}
	if to == "" {
		return nil, fmt.Errorf("missing to field in sms event")
	}
	text := firstNonEmptyPayload(payload, "Body", "body", "text")
	attachments := smsAttachments(payload)
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return nil, fmt.Errorf("sms event requires Body, body, text, or media")
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	id := firstNonEmptyPayload(payload, "MessageSid", "SmsSid", "message_id")
	if id == "" {
		id = fmt.Sprintf("sms_%d", receivedAt.UnixNano())
	}
	return &NormalizedMessage{
		ID:          id,
		Timestamp:   receivedAt.UTC(),
		Platform:    string(PlatformSMS),
		Sender:      Sender{ID: from, Name: from, PlatformHandle: from, PlatformID: from},
		Channel:     Channel{ID: to, Name: to, Type: ChannelTypeDirect, PlatformID: to},
		Text:        text,
		Attachments: attachments,
		PlatformMetadata: map[string]any{
			"provider":    firstNonEmptyPayload(payload, "Provider", "provider"),
			"account_sid": firstNonEmptyPayload(payload, "AccountSid", "account_sid"),
			"from":        from,
			"to":          to,
		},
	}, nil
}

type TwilioSMSTransport struct {
	AccountSID string
	AuthToken  string
	BaseURL    string
	Client     *http.Client
}

func NewTwilioSMSTransport(accountSID, authToken string, client *http.Client) *TwilioSMSTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &TwilioSMSTransport{
		AccountSID: accountSID,
		AuthToken:  authToken,
		BaseURL:    defaultTwilioBaseURL,
		Client:     client,
	}
}

func (t *TwilioSMSTransport) SendSMS(ctx context.Context, delivery SMSDelivery) (*SMSDeliveryResult, error) {
	accountSID := firstConfigured(delivery.AccountSID, t.AccountSID)
	authToken := firstConfigured(delivery.AuthToken, t.AuthToken)
	if accountSID == "" {
		return nil, fmt.Errorf("TWILIO_ACCOUNT_SID is required")
	}
	if authToken == "" {
		return nil, ErrMissingSecret("TWILIO_AUTH_TOKEN")
	}
	if delivery.From == "" {
		return nil, fmt.Errorf("sms delivery sender is required")
	}
	if delivery.To == "" {
		return nil, fmt.Errorf("sms delivery recipient is required")
	}
	if strings.TrimSpace(delivery.Body) == "" {
		return nil, fmt.Errorf("sms delivery body is required")
	}
	baseURL := strings.TrimRight(firstConfigured(t.BaseURL, defaultTwilioBaseURL), "/")
	endpoint := baseURL + "/2010-04-01/Accounts/" + url.PathEscape(accountSID) + "/Messages.json"
	form := url.Values{}
	form.Set("From", delivery.From)
	form.Set("To", delivery.To)
	form.Set("Body", delivery.Body)
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
		msg := firstNonEmptyString(mapString(parsed, "message"), strings.TrimSpace(string(respBody)), resp.Status)
		return nil, &SMSDeliveryError{StatusCode: resp.StatusCode, Message: msg, Retriable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	return &SMSDeliveryResult{
		MessageSID: mapString(parsed, "sid"),
		Status:     firstNonEmptyString(mapString(parsed, "status"), "sent"),
		Provider:   "twilio",
		Raw:        parsed,
	}, nil
}

type SMSDeliveryError struct {
	StatusCode int
	Message    string
	Retriable  bool
}

func (e *SMSDeliveryError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("sms delivery failed: HTTP %d: %s", e.StatusCode, e.Message)
}

func valuesToMap(values url.Values) map[string]any {
	out := make(map[string]any, len(values))
	for key, vals := range values {
		if len(vals) == 1 {
			out[key] = vals[0]
			continue
		}
		copied := append([]string(nil), vals...)
		out[key] = copied
	}
	return out
}

func firstNonEmptyPayload(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value := payloadString(payload, key)
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []string:
		if len(typed) > 0 {
			return typed[0]
		}
	case []any:
		if len(typed) > 0 {
			return fmt.Sprint(typed[0])
		}
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
	return ""
}

func smsAttachments(payload map[string]any) []Attachment {
	count, _ := strconv.Atoi(firstNonEmptyPayload(payload, "NumMedia", "num_media"))
	attachments := make([]Attachment, 0, count)
	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)
		mediaURL := firstNonEmptyPayload(payload, "MediaUrl"+idx, "media_url_"+idx)
		if mediaURL == "" {
			continue
		}
		mimeType := firstNonEmptyPayload(payload, "MediaContentType"+idx, "media_content_type_"+idx)
		attachments = append(attachments, Attachment{Type: smsMediaType(mimeType), URL: mediaURL, MimeType: mimeType})
	}
	return attachments
}

func smsMediaType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	default:
		return "file"
	}
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func encodeSMSForm(values map[string]string) []byte {
	form := url.Values{}
	for key, value := range values {
		form.Set(key, value)
	}
	return []byte(form.Encode())
}

func readSMSForm(body []byte) url.Values {
	values, _ := url.ParseQuery(string(bytes.TrimSpace(body)))
	return values
}
