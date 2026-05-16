package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"hop.top/aps/internal/logging"
	kitalias "hop.top/kit/go/console/alias"
)

// ServiceConfig is the persisted profile-facing service definition.
type ServiceConfig struct {
	ID           string            `yaml:"id"`
	Type         string            `yaml:"type"`
	Adapter      string            `yaml:"adapter,omitempty"`
	Profile      string            `yaml:"profile"`
	Description  string            `yaml:"description,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
	Labels       map[string]string `yaml:"labels,omitempty"`
	Options      map[string]string `yaml:"options,omitempty"`
	Delivery     *ServiceDelivery  `yaml:"delivery,omitempty"`
	LastInbound  *ServiceEventMeta `yaml:"last_inbound,omitempty"`
	LastOutbound *ServiceEventMeta `yaml:"last_outbound,omitempty"`
}

// ServiceRuntimeInfo describes reachable behavior for a persisted service.
type ServiceRuntimeInfo struct {
	Receives string
	Executes string
	Replies  string
	Maturity string
	Routes   []string
	Metadata ServiceRuntimeMetadata
}

// ServiceRuntimeMetadata captures contract-level service runtime details that
// are too structured for the display-oriented receives/executes/replies fields.
type ServiceRuntimeMetadata struct {
	Runtime       string
	Provider      string
	Ingress       string
	Handoff       string
	Delivery      string
	Retry         string
	ErrorHooks    []string
	ReceiveMode   string
	DeliveryModes []string
}

// ServiceDelivery describes operator-facing delivery health for a service.
type ServiceDelivery struct {
	Health      string                   `yaml:"health,omitempty"`
	Status      string                   `yaml:"status,omitempty"`
	LastError   string                   `yaml:"last_error,omitempty"`
	UpdatedAt   time.Time                `yaml:"updated_at,omitempty"`
	RetryPolicy *ServiceRetryPolicy      `yaml:"retry_policy,omitempty"`
	Attempts    []ServiceDeliveryAttempt `yaml:"attempts,omitempty"`
}

// ServiceRetryPolicy describes the operator-visible retry policy that applied
// to the most recent provider delivery attempt set.
type ServiceRetryPolicy struct {
	MaxAttempts int    `yaml:"max_attempts,omitempty"`
	BaseDelay   string `yaml:"base_delay,omitempty"`
	MaxDelay    string `yaml:"max_delay,omitempty"`
}

// ServiceDeliveryAttempt stores provider delivery observability without
// payloads or unredacted provider error bodies.
type ServiceDeliveryAttempt struct {
	At            time.Time `yaml:"at,omitempty"`
	Provider      string    `yaml:"provider,omitempty"`
	MessageID     string    `yaml:"message_id,omitempty"`
	ChannelID     string    `yaml:"channel_id,omitempty"`
	Attempt       int       `yaml:"attempt,omitempty"`
	MaxAttempts   int       `yaml:"max_attempts,omitempty"`
	Status        string    `yaml:"status,omitempty"`
	DeliveryID    string    `yaml:"delivery_id,omitempty"`
	Retriable     bool      `yaml:"retriable,omitempty"`
	Delay         string    `yaml:"delay,omitempty"`
	RedactedError string    `yaml:"redacted_error,omitempty"`
}

// ServiceEventMeta stores compact last-event metadata without retaining bodies.
type ServiceEventMeta struct {
	At          time.Time                `yaml:"at,omitempty"`
	Direction   string                   `yaml:"direction,omitempty"`
	MessageID   string                   `yaml:"message_id,omitempty"`
	Platform    string                   `yaml:"platform,omitempty"`
	ChannelID   string                   `yaml:"channel_id,omitempty"`
	SenderID    string                   `yaml:"sender_id,omitempty"`
	Status      string                   `yaml:"status,omitempty"`
	Detail      string                   `yaml:"detail,omitempty"`
	DeliveryID  string                   `yaml:"delivery_id,omitempty"`
	Attempt     int                      `yaml:"attempt,omitempty"`
	MaxAttempts int                      `yaml:"max_attempts,omitempty"`
	Retriable   bool                     `yaml:"retriable,omitempty"`
	RetryDelay  string                   `yaml:"retry_delay,omitempty"`
	Attempts    []ServiceDeliveryAttempt `yaml:"attempts,omitempty"`
	RetryPolicy *ServiceRetryPolicy      `yaml:"retry_policy,omitempty"`
}

// ServiceValidationResult is a static config validation report.
type ServiceValidationResult struct {
	Valid    bool
	Issues   []string
	Warnings []string
}

// ResolvedServiceType records how user-facing service type input resolved.
type ResolvedServiceType struct {
	InputType string
	Type      string
	Adapter   string
	Aliased   bool
}

var serviceTypeAliases = map[string]string{
	"api":      "api agent-protocol",
	"webhook":  "webhook generic",
	"a2a":      "a2a jsonrpc",
	"events":   "events bus",
	"mobile":   "mobile aps",
	"slack":    "message slack",
	"telegram": "message telegram",
	"discord":  "message discord",
	"sms":      "message sms",
	"whatsapp": "message whatsapp",
	"email":    "ticket email",
	"github":   "ticket github",
	"gitlab":   "ticket gitlab",
	"jira":     "ticket jira",
	"linear":   "ticket linear",
}

var canonicalServiceTypes = map[string]bool{
	"api":     true,
	"webhook": true,
	"a2a":     true,
	"client":  true,
	"message": true,
	"ticket":  true,
	"events":  true,
	"mobile":  true,
	"voice":   true,
}

var defaultServiceAdapters = map[string]string{
	"api":     "agent-protocol",
	"webhook": "generic",
	"a2a":     "jsonrpc",
	"events":  "bus",
	"mobile":  "aps",
}

// NewServiceAliasStore returns the kit alias store used for service type
// expansion. Values are encoded as "canonical-type adapter".
func NewServiceAliasStore() *kitalias.Store {
	store := kitalias.NewStore("")
	for name, target := range serviceTypeAliases {
		_ = store.Set(name, target)
	}
	return store
}

// ResolveServiceType expands a canonical service type or adapter alias into
// canonical type/adapter fields.
func ResolveServiceType(typeInput, adapterInput string) (ResolvedServiceType, error) {
	typeInput = strings.TrimSpace(strings.ToLower(typeInput))
	adapterInput = strings.TrimSpace(strings.ToLower(adapterInput))
	if typeInput == "" {
		return ResolvedServiceType{}, fmt.Errorf("service type is required")
	}

	store := NewServiceAliasStore()
	expanded := store.Expand([]string{typeInput})
	if len(expanded) == 2 && expanded[0] != typeInput {
		if adapterInput != "" && adapterInput != expanded[1] {
			return ResolvedServiceType{}, fmt.Errorf(
				"service type alias %q resolves adapter %q, cannot also use adapter %q",
				typeInput, expanded[1], adapterInput,
			)
		}
		return ResolvedServiceType{
			InputType: typeInput,
			Type:      expanded[0],
			Adapter:   expanded[1],
			Aliased:   true,
		}, nil
	}

	if !canonicalServiceTypes[typeInput] {
		return ResolvedServiceType{}, fmt.Errorf("unknown service type or alias %q", typeInput)
	}

	adapter := adapterInput
	if adapter == "" {
		adapter = defaultServiceAdapters[typeInput]
	}
	if adapter == "" {
		return ResolvedServiceType{}, fmt.Errorf("service type %q requires --adapter", typeInput)
	}

	return ResolvedServiceType{
		InputType: typeInput,
		Type:      typeInput,
		Adapter:   adapter,
	}, nil
}

func ServiceTypeAliases() map[string]string {
	out := make(map[string]string, len(serviceTypeAliases))
	for k, v := range serviceTypeAliases {
		out[k] = v
	}
	return out
}

func SortedServiceTypeInputs() []string {
	values := make([]string, 0, len(canonicalServiceTypes)+len(serviceTypeAliases))
	for value := range canonicalServiceTypes {
		values = append(values, value)
	}
	for value := range serviceTypeAliases {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func GetServicesDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "services"), nil
}

func GetServicePath(id string) (string, error) {
	safeID, err := normalizeServiceID(id)
	if err != nil {
		return "", err
	}
	dir, err := GetServicesDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, safeID+".yaml")
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("service id %q resolves outside services directory", id)
	}
	return path, nil
}

func normalizeServiceID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("service id cannot be empty")
	}
	if filepath.IsAbs(id) || id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("service id %q must not contain path components", id)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("service id %q contains invalid character %q", id, r)
	}
	return id, nil
}

func SaveService(service *ServiceConfig) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}
	id, err := normalizeServiceID(service.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(service.Type) == "" {
		return fmt.Errorf("service type cannot be empty")
	}
	if strings.TrimSpace(service.Profile) == "" {
		return fmt.Errorf("service profile cannot be empty")
	}
	service.ID = id

	path, err := GetServicePath(service.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to create services directory: %w", err)
	}
	data, err := yaml.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write service: %w", err)
	}
	return nil
}

func LoadService(id string) (*ServiceConfig, error) {
	path, err := GetServicePath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service %s: %w", id, err)
	}
	var service ServiceConfig
	if err := yaml.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("failed to parse service %s: %w", id, err)
	}
	if service.ID != id {
		return nil, fmt.Errorf("service ID mismatch: path=%s, content=%s", id, service.ID)
	}
	return &service, nil
}

func ServiceWebhookPath(service *ServiceConfig) string {
	if service == nil {
		return ""
	}
	switch service.Type {
	case "message":
		return "/services/" + service.ID + "/webhook"
	case "ticket":
		return "/services/" + service.ID + "/ticket/" + service.Adapter
	case "webhook":
		return "/webhook"
	default:
		return ""
	}
}

func ServiceWebhookURL(service *ServiceConfig, baseURL string) (string, error) {
	path := ServiceWebhookPath(service)
	if path == "" {
		return "", fmt.Errorf("service %q does not expose an HTTP webhook route", serviceID(service))
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	return strings.TrimRight(baseURL, "/") + path, nil
}

func ValidateServiceConfig(service *ServiceConfig) ServiceValidationResult {
	result := ServiceValidationResult{Valid: true}
	if service == nil {
		result.Issues = append(result.Issues, "service config is required")
		result.Valid = false
		return result
	}
	if strings.TrimSpace(service.ID) == "" {
		result.Issues = append(result.Issues, "service id is required")
	}
	if strings.TrimSpace(service.Type) == "" {
		result.Issues = append(result.Issues, "service type is required")
	}
	if strings.TrimSpace(service.Profile) == "" {
		result.Issues = append(result.Issues, "service profile is required")
	}
	if service.Type == "message" {
		validateMessageServiceConfig(service, &result)
	}
	result.Valid = len(result.Issues) == 0
	return result
}

func RecordServiceInboundEvent(id string, event ServiceEventMeta) error {
	service, err := LoadService(id)
	if err != nil {
		return err
	}
	event.Direction = "inbound"
	if event.Status == "" {
		event.Status = "received"
	}
	recordedEvent := normalizeServiceEvent(event)
	service.LastInbound = &recordedEvent
	serviceDelivery(service).Health = deliveryHealth(service)
	service.Delivery.UpdatedAt = recordedEvent.At
	return SaveService(service)
}

func RecordServiceOutboundEvent(id string, event ServiceEventMeta) error {
	service, err := LoadService(id)
	if err != nil {
		return err
	}
	event.Direction = "outbound"
	if event.Status == "" {
		event.Status = "unknown"
	}
	recordedEvent := normalizeServiceEvent(event)
	service.LastOutbound = &recordedEvent
	delivery := serviceDelivery(service)
	delivery.UpdatedAt = recordedEvent.At
	applyOutboundDeliveryState(delivery, recordedEvent)
	return SaveService(service)
}

func DescribeServiceRuntime(service *ServiceConfig) ServiceRuntimeInfo {
	if service == nil {
		return ServiceRuntimeInfo{Maturity: "planned"}
	}

	switch service.Type {
	case "api":
		return ServiceRuntimeInfo{
			Receives: "Agent Protocol HTTP requests",
			Executes: "profile action",
			Replies:  "JSON run/thread/store responses or SSE output stream",
			Maturity: "ready",
			Routes:   []string{"/health", "/v1/runs", "/v1/threads", "/v1/agents", "/v1/store", "/v1/skills"},
		}
	case "webhook":
		return ServiceRuntimeInfo{
			Receives: "HTTP POST /webhook with X-APS-Event",
			Executes: "mapped profile action",
			Replies:  "status JSON, not action stdout",
			Maturity: "status-only",
			Routes:   []string{"/webhook"},
		}
	case "a2a":
		return ServiceRuntimeInfo{
			Receives: "A2A JSON-RPC task messages",
			Executes: "placeholder text processing",
			Replies:  "A2A task response",
			Maturity: "placeholder",
			Routes:   []string{"aps a2a server --profile " + service.Profile},
		}
	case "client":
		if service.Adapter == "acp" {
			return ServiceRuntimeInfo{
				Receives: "stdio JSON-RPC",
				Executes: "ACP session, filesystem, terminal, and skill methods",
				Replies:  "JSON-RPC responses",
				Maturity: "ready",
				Routes:   []string{"aps acp server " + service.Profile},
			}
		}
	case "message":
		route := "/services/" + service.ID + "/webhook"
		provider := service.Adapter
		receiveMode := "webhook"
		replyMode := "provider delivery"
		if service.Options != nil {
			if value := strings.TrimSpace(service.Options["provider"]); value != "" {
				provider = value
			}
			if value := strings.TrimSpace(service.Options["receive"]); value != "" {
				receiveMode = value
			}
			if value := strings.TrimSpace(service.Options["reply"]); value != "" {
				replyMode = value
			}
		}
		return ServiceRuntimeInfo{
			Receives: "HTTP POST " + route,
			Executes: "normalized message execution handoff",
			Replies:  provider + " " + replyMode,
			Maturity: "ready",
			Routes:   []string{route},
			Metadata: ServiceRuntimeMetadata{
				Runtime:     "message-provider",
				Provider:    provider,
				Ingress:     "native provider " + receiveMode + " ingress",
				Handoff:     "normalized message execution handoff",
				Delivery:    "provider delivery interface",
				Retry:       "provider delivery retry policy max_attempts=3 base_delay=1s max_delay=30s",
				ErrorHooks:  []string{"ingress", "normalize", "route", "execute", "deliver", "retry"},
				ReceiveMode: receiveMode,
				DeliveryModes: []string{
					"text",
					"reaction",
					"file",
				},
			},
		}
	case "ticket":
		return describeTicketServiceRuntime(service)
	case "events":
		return ServiceRuntimeInfo{
			Receives: "bus topics",
			Executes: "none",
			Replies:  "JSONL to stdout",
			Maturity: "observe-only",
			Routes:   []string{"aps listen --profile " + service.Profile},
		}
	case "mobile":
		return ServiceRuntimeInfo{
			Receives: "pairing requests and WebSocket command messages",
			Executes: "pairing/token flow; command execution placeholder",
			Replies:  "pairing responses and placeholder command acknowledgements",
			Maturity: "placeholder",
			Routes:   []string{"aps adapter pair --profile " + service.Profile},
		}
	case "voice":
		return ServiceRuntimeInfo{
			Receives: "component voice adapters only; no service route mounted",
			Executes: "backend process lifecycle and session registration only",
			Replies:  "component-level audio/text frames",
			Maturity: "component",
		}
	}

	return ServiceRuntimeInfo{
		Receives: "not mounted by aps service runtime",
		Executes: "not verified",
		Replies:  "not verified",
		Maturity: "planned",
	}
}

func validateMessageServiceConfig(service *ServiceConfig, result *ServiceValidationResult) {
	adapter := strings.TrimSpace(strings.ToLower(service.Adapter))
	if adapter == "" {
		result.Issues = append(result.Issues, "message service requires an adapter")
		return
	}
	if !knownMessageAdapters[adapter] {
		result.Issues = append(result.Issues, fmt.Sprintf("unsupported message adapter %q", service.Adapter))
		return
	}
	options := service.Options
	env := service.Env
	if strings.TrimSpace(options["default_action"]) == "" {
		result.Issues = append(result.Issues, "message service requires option default_action to dispatch inbound messages")
	}
	validateMessageReceiveMode(options["receive"], result)
	validateMessageReplyMode(options["reply"], result)
	validateMessageExecutionMode(options["execution"], result)
	switch adapter {
	case "telegram":
		requireEnv(env, result, "TELEGRAM_BOT_TOKEN")
		validateTelegramWebhookSecret(options, result)
	case "slack":
		requireEnv(env, result, "SLACK_BOT_TOKEN", "SLACK_SIGNING_SECRET")
		if truthyServiceOption(options["require_bot_mention"]) && strings.TrimSpace(options["bot_user_id"]) == "" {
			result.Warnings = append(result.Warnings, "Slack require_bot_mention without bot_user_id only accepts app_mention events")
		}
	case "discord":
		requireEnv(env, result, "DISCORD_BOT_TOKEN")
		if strings.EqualFold(strings.TrimSpace(options["receive"]), "interaction") {
			requireEnv(env, result, "DISCORD_PUBLIC_KEY")
		}
	case "sms":
		provider := strings.TrimSpace(strings.ToLower(options["provider"]))
		if provider == "" {
			result.Issues = append(result.Issues, "sms message service requires option provider")
		} else if provider != "twilio" && provider != "generic" {
			result.Issues = append(result.Issues, fmt.Sprintf("unsupported sms provider %q", options["provider"]))
		}
		if strings.TrimSpace(options["from"]) == "" {
			result.Issues = append(result.Issues, "sms message service requires option from")
		}
		if strings.TrimSpace(options["allowed_numbers"]) == "" {
			result.Warnings = append(result.Warnings, "sms service has no allowed numbers; any sender can route inbound messages")
		}
		if provider == "twilio" || provider == "" {
			requireEnv(env, result, "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN")
		}
	case "whatsapp":
		provider := strings.TrimSpace(strings.ToLower(options["provider"]))
		if provider == "" {
			result.Issues = append(result.Issues, "whatsapp message service requires option provider")
		} else if provider != "whatsapp-cloud" && provider != "twilio" && provider != "generic" {
			result.Issues = append(result.Issues, fmt.Sprintf("unsupported whatsapp provider %q", options["provider"]))
		}
		if provider == "whatsapp-cloud" || provider == "" {
			if strings.TrimSpace(options["phone_number_id"]) == "" {
				result.Issues = append(result.Issues, "whatsapp-cloud message service requires option phone_number_id")
			}
			if !looksNumericID(options["phone_number_id"]) {
				result.Issues = append(result.Issues, "whatsapp phone_number_id must be numeric")
			}
			requireEnv(env, result, "WHATSAPP_ACCESS_TOKEN")
			if strings.TrimSpace(options["verify_token"]) == "" && strings.TrimSpace(options["verify_token_env"]) == "" && strings.TrimSpace(env["WHATSAPP_VERIFY_TOKEN"]) == "" {
				result.Warnings = append(result.Warnings, "whatsapp verify token not set; Cloud webhook verification will fail")
			}
			if strings.TrimSpace(options["app_secret"]) == "" && strings.TrimSpace(options["app_secret_env"]) == "" && strings.TrimSpace(options["signing_secret_env"]) == "" && strings.TrimSpace(env["WHATSAPP_APP_SECRET"]) == "" {
				result.Warnings = append(result.Warnings, "whatsapp app secret not set; Cloud webhook signatures will not be validated")
			}
		}
		if provider == "twilio" {
			if strings.TrimSpace(options["from"]) == "" {
				result.Issues = append(result.Issues, "twilio whatsapp message service requires option from")
			}
			requireEnv(env, result, "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN")
		}
		if strings.TrimSpace(options["allowed_numbers"]) == "" {
			result.Warnings = append(result.Warnings, "whatsapp service has no allowed numbers; any sender can route inbound messages")
		}
		if truthyServiceOption(options["template_required"]) && strings.TrimSpace(options["template_name"]) == "" {
			result.Issues = append(result.Issues, "whatsapp template_required requires option template_name")
		}
	}
}

func validateTelegramWebhookSecret(options map[string]string, result *ServiceValidationResult) {
	token := strings.TrimSpace(options["webhook_secret_token"])
	tokenEnv := strings.TrimSpace(options["webhook_secret_token_env"])
	if token == "" && tokenEnv == "" {
		result.Warnings = append(result.Warnings, "telegram webhook secret token not set; Telegram requests will not be secret-token validated")
		return
	}
	if len(token) > 256 {
		result.Issues = append(result.Issues, "telegram webhook secret token must be 256 characters or fewer")
	}
	if strings.ContainsAny(token, "\r\n") {
		result.Issues = append(result.Issues, "telegram webhook secret token must not contain newlines")
	}
}

var knownMessageAdapters = map[string]bool{
	"telegram": true,
	"slack":    true,
	"discord":  true,
	"sms":      true,
	"whatsapp": true,
}

func validateMessageReceiveMode(value string, result *ServiceValidationResult) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		result.Warnings = append(result.Warnings, "message receive mode not set; assuming webhook")
		return
	}
	switch value {
	case "webhook", "polling":
	default:
		result.Issues = append(result.Issues, "message receive mode must be webhook or polling")
	}
}

func validateMessageReplyMode(value string, result *ServiceValidationResult) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		result.Warnings = append(result.Warnings, "reply mode not set; assuming text")
		return
	}
	switch value {
	case "text", "auto", "none":
		if value == "none" {
			result.Warnings = append(result.Warnings, "reply mode none disables outbound delivery")
		}
	default:
		result.Issues = append(result.Issues, "reply mode must be text, auto, or none")
	}
}

func validateMessageExecutionMode(value string, result *ServiceValidationResult) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return
	}
	switch value {
	case "action", "chat":
	default:
		result.Issues = append(result.Issues, "execution mode must be action or chat")
	}
}

func requireEnv(env map[string]string, result *ServiceValidationResult, keys ...string) {
	for _, key := range keys {
		if strings.TrimSpace(env[key]) == "" {
			result.Issues = append(result.Issues, "missing env binding "+key)
		}
	}
}

func truthyServiceOption(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func looksNumericID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func serviceDelivery(service *ServiceConfig) *ServiceDelivery {
	if service.Delivery == nil {
		service.Delivery = &ServiceDelivery{}
	}
	return service.Delivery
}

func deliveryHealth(service *ServiceConfig) string {
	if service.Delivery != nil && service.Delivery.Health != "" {
		return service.Delivery.Health
	}
	if service.LastInbound != nil || service.LastOutbound != nil {
		return "receiving"
	}
	return "unknown"
}

func normalizeServiceEvent(event ServiceEventMeta) ServiceEventMeta {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	} else {
		event.At = event.At.UTC()
	}
	event.Detail = logging.Apply(event.Detail)
	for i := range event.Attempts {
		event.Attempts[i] = normalizeServiceDeliveryAttempt(event.Attempts[i], event.At)
	}
	return event
}

func applyOutboundDeliveryState(delivery *ServiceDelivery, event ServiceEventMeta) {
	if event.RetryPolicy != nil {
		delivery.RetryPolicy = event.RetryPolicy
	}
	if len(event.Attempts) > 0 {
		delivery.Attempts = append([]ServiceDeliveryAttempt(nil), event.Attempts...)
		last := event.Attempts[len(event.Attempts)-1]
		delivery.Status = last.Status
		if !last.At.IsZero() {
			delivery.UpdatedAt = last.At
		}
		if last.RedactedError != "" {
			delivery.LastError = last.RedactedError
		}
		setDeliveryHealth(delivery, last.Status)
		return
	}
	if event.Attempt > 0 {
		attempt := normalizeServiceDeliveryAttempt(ServiceDeliveryAttempt{
			At:            event.At,
			Provider:      event.Platform,
			Attempt:       event.Attempt,
			MaxAttempts:   event.MaxAttempts,
			Status:        event.Status,
			DeliveryID:    event.DeliveryID,
			Retriable:     event.Retriable,
			Delay:         event.RetryDelay,
			RedactedError: event.Detail,
		}, event.At)
		delivery.Attempts = []ServiceDeliveryAttempt{attempt}
	}
	delivery.Status = event.Status
	if event.Detail != "" {
		delivery.LastError = event.Detail
	}
	setDeliveryHealth(delivery, event.Status)
}

func normalizeServiceDeliveryAttempt(attempt ServiceDeliveryAttempt, fallbackAt time.Time) ServiceDeliveryAttempt {
	if attempt.At.IsZero() {
		attempt.At = fallbackAt
	} else {
		attempt.At = attempt.At.UTC()
	}
	attempt.RedactedError = logging.Apply(attempt.RedactedError)
	return attempt
}

func setDeliveryHealth(delivery *ServiceDelivery, status string) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "success", "accepted", "completed", "executed", "sent", "delivered":
		delivery.Health = "healthy"
		delivery.LastError = ""
	case "retry_scheduled", "retrying":
		delivery.Health = "degraded"
	case "dead_letter", "failed", "failed_delivery", "error":
		delivery.Health = "failed"
	default:
		if status == "" {
			delivery.Health = "unknown"
		} else {
			delivery.Health = "degraded"
		}
	}
}

func serviceID(service *ServiceConfig) string {
	if service == nil {
		return ""
	}
	return service.ID
}

func SyntheticMessageWebhookPayload(adapter string) ([]byte, error) {
	switch strings.TrimSpace(strings.ToLower(adapter)) {
	case "telegram":
		return json.Marshal(map[string]any{
			"update_id": 1000001,
			"message": map[string]any{
				"message_id": 1,
				"from":       map[string]any{"id": 1001, "first_name": "APS"},
				"chat":       map[string]any{"id": -1001234567890, "type": "group"},
				"date":       time.Now().Unix(),
				"text":       "aps service test",
			},
		})
	case "slack":
		return json.Marshal(map[string]any{
			"event": map[string]any{
				"client_msg_id": "aps-service-test",
				"user":          "U012TEST",
				"channel":       "C012TEST",
				"text":          "aps service test",
				"ts":            fmt.Sprintf("%d.000000", time.Now().Unix()),
			},
		})
	case "discord":
		return json.Marshal(map[string]any{
			"id":         "aps-service-test",
			"channel_id": "123456789012345678",
			"content":    "aps service test",
			"author":     map[string]any{"id": "987654321098765432", "username": "aps"},
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	case "sms":
		return json.Marshal(map[string]any{
			"MessageSid": "SMAPS000000000000000000000000000000",
			"From":       "+15550100001",
			"To":         "+15550100002",
			"Body":       "aps service test",
		})
	case "whatsapp":
		return json.Marshal(map[string]any{
			"object": "whatsapp_business_account",
			"entry": []any{
				map[string]any{
					"id": "123456789000000",
					"changes": []any{
						map[string]any{
							"field": "messages",
							"value": map[string]any{
								"messaging_product": "whatsapp",
								"metadata": map[string]any{
									"display_phone_number": "+15550100002",
									"phone_number_id":      "123456789012345",
								},
								"contacts": []any{
									map[string]any{
										"profile": map[string]any{"name": "APS"},
										"wa_id":   "15550100001",
									},
								},
								"messages": []any{
									map[string]any{
										"from":      "15550100001",
										"id":        "wamid.APS000000000000000000000000000001",
										"timestamp": time.Now().Unix(),
										"type":      "text",
										"text":      map[string]any{"body": "aps service test"},
									},
								},
							},
						},
					},
				},
			},
		})
	default:
		return nil, fmt.Errorf("no synthetic webhook payload for message adapter %q", adapter)
	}
}

func describeTicketServiceRuntime(service *ServiceConfig) ServiceRuntimeInfo {
	route := "/services/" + service.ID + "/ticket/" + service.Adapter
	info := ServiceRuntimeInfo{
		Receives: "ticket events",
		Executes: "routed profile action with normalized ticket payload",
		Replies:  "status metadata",
		Maturity: "component",
		Routes:   []string{route},
	}
	switch service.Adapter {
	case "jira":
		info.Receives = "Jira issue/comment events"
		info.Replies = "Jira comment body or status metadata"
	case "linear":
		info.Receives = "Linear issue/comment events"
		info.Replies = "Linear comment body or status metadata"
	case "gitlab":
		info.Receives = "GitLab issue/MR/note events"
		info.Replies = "GitLab note body or status metadata"
	}
	return info
}
