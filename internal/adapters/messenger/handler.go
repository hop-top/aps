package messenger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

// VoiceHandler handles voice messages (audio attachments) from messenger platforms.
// If set on Handler, audio messages bypass the action router and go to the voice pipeline.
type VoiceHandler interface {
	HandleVoiceMessage(ctx context.Context, msg *msgtypes.NormalizedMessage) error
}

// MessageLogger defines the logging interface for messenger message events.
// This decouples the handler from the concrete WorkspaceMessageLogger
// implementation in the core layer, which may still be under construction.
type MessageLogger interface {
	LogMessageReceived(msg *msgtypes.NormalizedMessage) error
	LogActionExecuted(msgID, actionName string, status string, durationMS int64) error
}

// Handler handles incoming messenger webhook events over HTTP. It normalizes
// platform-specific payloads, routes them to the appropriate profile action,
// and returns a platform-appropriate response.
type Handler struct {
	router            *MessageRouter
	normalizer        *Normalizer
	logger            MessageLogger
	voiceHandler      VoiceHandler
	validator         *msgtypes.ServiceValidator
	chatRunner        ChatTurnRunner
	telegramTransport TelegramTransport
	slackTransport    SlackTransport
	whatsappTransport msgtypes.WhatsAppTransport
}

// NewHandler creates a Handler with the given router, normalizer, and optional
// logger. The logger may be nil, in which case message logging is skipped.
// Additional functional options may be provided (e.g. WithVoiceHandler).
func NewHandler(router *MessageRouter, normalizer *Normalizer, logger MessageLogger, opts ...func(*Handler)) *Handler {
	validator := msgtypes.NewServiceValidator()
	validator.Hooks[string(msgtypes.PlatformTelegram)] = msgtypes.TelegramAuthHook{}
	validator.Hooks[string(msgtypes.PlatformSlack)] = msgtypes.SlackAuthHook{}
	validator.Hooks[string(msgtypes.PlatformWhatsApp)] = msgtypes.WhatsAppAuthHook{}
	validator.Hooks["whatsapp-cloud"] = msgtypes.WhatsAppAuthHook{}
	h := &Handler{router: router, normalizer: normalizer, logger: logger, validator: validator}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithVoiceHandler attaches a voice pipeline handler to the messenger Handler.
func WithVoiceHandler(vh VoiceHandler) func(*Handler) {
	return func(h *Handler) { h.voiceHandler = vh }
}

func WithServiceValidator(v *msgtypes.ServiceValidator) func(*Handler) {
	return func(h *Handler) {
		if v != nil {
			if v.Hooks == nil {
				v.Hooks = msgtypes.ProviderAuthHooks{}
			}
			if _, ok := v.Hooks[string(msgtypes.PlatformTelegram)]; !ok {
				v.Hooks[string(msgtypes.PlatformTelegram)] = msgtypes.TelegramAuthHook{}
			}
			if _, ok := v.Hooks[string(msgtypes.PlatformSlack)]; !ok {
				v.Hooks[string(msgtypes.PlatformSlack)] = msgtypes.SlackAuthHook{}
			}
			if _, ok := v.Hooks[string(msgtypes.PlatformWhatsApp)]; !ok {
				v.Hooks[string(msgtypes.PlatformWhatsApp)] = msgtypes.WhatsAppAuthHook{}
			}
			if _, ok := v.Hooks["whatsapp-cloud"]; !ok {
				v.Hooks["whatsapp-cloud"] = msgtypes.WhatsAppAuthHook{}
			}
		}
		h.validator = v
	}
}

func WithTelegramTransport(t TelegramTransport) func(*Handler) {
	return func(h *Handler) { h.telegramTransport = t }
}

func WithSlackTransport(t SlackTransport) func(*Handler) {
	return func(h *Handler) { h.slackTransport = t }
}

func WithWhatsAppTransport(t msgtypes.WhatsAppTransport) func(*Handler) {
	return func(h *Handler) { h.whatsappTransport = t }
}

func WithChatTurnRunner(runner ChatTurnRunner) func(*Handler) {
	return func(h *Handler) { h.chatRunner = runner }
}

// ServeHTTP handles incoming webhook POST requests. The URL path is expected
// to end with /messengers/{platform}/webhook. It extracts the platform from
// the path, validates the request, and dispatches to handleWebhook.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST requests are accepted")
		return
	}

	// Extract platform from URL path. Expected pattern:
	//   /messengers/{platform}/webhook
	// We look for the segment immediately before "/webhook".
	platform := extractPlatform(r.URL.Path)
	if platform == "" {
		writeError(w, http.StatusNotFound, "unable to determine platform from URL path; expected /messengers/{platform}/webhook")
		return
	}

	h.handleWebhook(w, r, platform)
}

func (h *Handler) ServeServiceWebhook(w http.ResponseWriter, r *http.Request, serviceID, adapter string) {
	if r.Method == http.MethodGet && adapter == string(msgtypes.PlatformWhatsApp) {
		h.handleWhatsAppVerification(w, r, serviceID, adapter)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST requests are accepted")
		return
	}
	if serviceID == "" || adapter == "" {
		writeError(w, http.StatusInternalServerError, "message service is not configured")
		return
	}
	service, err := core.LoadService(serviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q not found", serviceID))
		return
	}
	if service.Type != "message" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has type %q, not message", serviceID, service.Type))
		return
	}
	if strings.TrimSpace(service.Adapter) != adapter {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has adapter %q, not %q", serviceID, service.Adapter, adapter))
		return
	}
	h.handleWebhookForMessenger(w, r, adapter, serviceID, service)
}

// handleWebhook processes a single webhook event for the given platform.
func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request, platform string) {
	h.handleWebhookForMessenger(w, r, platform, "", nil)
}

func (h *Handler) handleWebhookForMessenger(w http.ResponseWriter, r *http.Request, platform, messengerName string, service *core.ServiceConfig) {
	// Decode the raw JSON payload.
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	if service != nil {
		validationConfig := serviceValidationConfig(service)
		form := formValuesForValidation(r.Header, rawBody)
		if err := h.serviceValidator().ValidateRequest(r.Context(), msgtypes.RequestValidationInput{
			Service:    validationConfig,
			Method:     r.Method,
			URL:        publicRequestURL(r, service),
			Headers:    r.Header,
			Body:       rawBody,
			Form:       form,
			RemoteAddr: r.RemoteAddr,
		}); err != nil {
			writeValidationError(w, err)
			return
		}
	}
	if service != nil && platform == string(msgtypes.PlatformTelegram) {
		h.handleTelegramServiceWebhook(w, r, rawBody, messengerName, service)
		return
	}
	if service != nil && platform == string(msgtypes.PlatformSlack) && serviceProvider(service, string(msgtypes.PlatformSlack)) == string(msgtypes.PlatformSlack) {
		h.handleSlackServiceWebhook(w, r, rawBody, messengerName, service)
		return
	}
	if service != nil && platform == string(msgtypes.PlatformWhatsApp) {
		h.handleWhatsAppServiceWebhook(w, r, rawBody, messengerName, service)
		return
	}

	body, err := decodeWebhookBody(platform, r.Header, rawBody)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msgtypes.MessengerPlatform(platform) == msgtypes.PlatformSlack {
		if handleSlackAcknowledgement(w, body, h.serviceValidator(), messengerName, service) {
			return
		}
	}

	// Normalize the platform-specific event into a NormalizedMessage.
	msg, err := h.normalizer.Normalize(platform, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("normalization failed: %v", err))
		return
	}
	if messengerName != "" {
		if msg.PlatformMetadata == nil {
			msg.PlatformMetadata = map[string]any{}
		}
		msg.PlatformMetadata["messenger_name"] = messengerName
		_ = core.RecordServiceInboundEvent(messengerName, core.ServiceEventMeta{
			MessageID: msg.ID,
			Platform:  platform,
			ChannelID: msg.Channel.ID,
			SenderID:  msg.Sender.ID,
			Status:    "received",
		})
	}
	if service != nil {
		if err := h.serviceValidator().ValidateMessage(serviceValidationConfig(service), msg); err != nil {
			writeValidationError(w, err)
			return
		}
	}

	// Log the received message if a logger is available.
	if h.logger != nil {
		// Log errors are not fatal to the processing pipeline.
		_ = h.logger.LogMessageReceived(msg)
	}

	// If this message contains audio and a voice handler is registered, delegate to it.
	if h.voiceHandler != nil && hasAudioAttachment(msg) {
		if err := h.voiceHandler.HandleVoiceMessage(r.Context(), msg); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("voice handling failed: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "accepted",
			"message_id": msg.ID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Route and execute the action.
	result, err := h.router.HandleMessage(r.Context(), msg)
	if err != nil {
		// Log the failure if possible.
		if h.logger != nil {
			_ = h.logger.LogActionExecuted(msg.ID, "", "failed", 0)
		}
		if messengerName != "" {
			_ = core.RecordServiceOutboundEvent(messengerName, core.ServiceEventMeta{
				MessageID: msg.ID,
				Platform:  platform,
				ChannelID: msg.Channel.ID,
				SenderID:  msg.Sender.ID,
				Status:    "failed",
				Detail:    err.Error(),
			})
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("message handling failed: %v", err))
		return
	}

	// Log the action execution result.
	if h.logger != nil {
		durationMS := result.ExecutionTime.Milliseconds()
		actionName := ""
		// Try to extract action name from the output for logging.
		if result.Status == "success" {
			actionName = extractActionFromOutput(result.Output)
		}
		_ = h.logger.LogActionExecuted(msg.ID, actionName, result.Status, durationMS)
	}

	// Build the response payload. If the action was successful, we
	// denormalize the result back to platform format. Otherwise we return
	// a generic status response.
	var response map[string]any
	if result.Status == "success" {
		response, err = h.normalizer.Denormalize(platform, result)
		if err != nil {
			// Denormalization failure is not fatal; fall back to generic.
			response = map[string]any{
				"status":  result.Status,
				"message": result.Output,
			}
		}
	} else {
		response = map[string]any{
			"status":  result.Status,
			"message": result.Output,
		}
	}

	response["message_id"] = msg.ID
	response["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	if messengerName != "" {
		_ = core.RecordServiceOutboundEvent(messengerName, core.ServiceEventMeta{
			MessageID: msg.ID,
			Platform:  platform,
			ChannelID: msg.Channel.ID,
			SenderID:  msg.Sender.ID,
			Status:    result.Status,
			Detail:    result.Output,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleTelegramServiceWebhook(w http.ResponseWriter, r *http.Request, rawBody []byte, serviceID string, service *core.ServiceConfig) {
	provider := NewTelegramProvider(TelegramProviderConfig{
		BotToken:  resolveServiceEnv(service, "TELEGRAM_BOT_TOKEN"),
		Transport: h.telegramTransport,
	})
	validatingProvider := &serviceValidatingProvider{
		base:      provider,
		validator: h.serviceValidator(),
		service:   serviceValidationConfig(service),
	}
	var deliveryAttempts []msgtypes.DeliveryAttempt
	runtime, err := msgtypes.NewRuntime(
		validatingProvider,
		h.router,
		&serviceRuntimeExecutor{router: h.router, service: service, chatRunner: h.chatRunner},
		runtimeOptionsWithDeliveryAttempts(serviceID, &deliveryAttempts),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("telegram runtime failed: %v", err))
		return
	}

	result, err := runtime.HandleIngress(r.Context(), msgtypes.NativeIngress{
		ServiceID:  serviceID,
		Provider:   string(msgtypes.PlatformTelegram),
		Mode:       msgtypes.IngressModeWebhook,
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header,
		Query:      r.URL.Query(),
		Body:       rawBody,
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		_ = recordServiceDeliveryAttempts(serviceID, deliveryAttempts, nil)
		writeRuntimeError(w, err)
		return
	}
	if result != nil && result.Message != nil {
		_ = core.RecordServiceInboundEvent(serviceID, core.ServiceEventMeta{
			MessageID: result.Message.ID,
			Platform:  string(msgtypes.PlatformTelegram),
			ChannelID: result.Message.Channel.ID,
			SenderID:  result.Message.Sender.ID,
			Status:    "received",
		})
		if result.Delivery != nil {
			_ = recordServiceDeliveryAttempts(serviceID, result.DeliveryAttempts, result.Message)
			if len(result.DeliveryAttempts) == 0 {
				_ = core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
					MessageID: result.Message.ID,
					Platform:  string(msgtypes.PlatformTelegram),
					ChannelID: result.Message.Channel.ID,
					SenderID:  result.Message.Sender.ID,
					Status:    result.Delivery.Status,
				})
			}
		} else if result.Result != nil {
			_ = recordServiceExecutionEvent(serviceID, result.Message, result.Result)
		}
	}

	response := map[string]any{
		"status":    "accepted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if result != nil {
		if result.Message != nil {
			response["message_id"] = result.Message.ID
		}
		response["route"] = result.Route.TargetAction()
		if result.Result != nil {
			response["status"] = result.Result.Status
		}
		if result.Delivery != nil {
			response["delivery"] = result.Delivery
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleSlackServiceWebhook(w http.ResponseWriter, r *http.Request, rawBody []byte, serviceID string, service *core.ServiceConfig) {
	body, err := decodeWebhookBody(string(msgtypes.PlatformSlack), r.Header, rawBody)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if handleSlackAcknowledgement(w, body, h.serviceValidator(), serviceID, service) {
		return
	}

	provider := NewSlackProvider(SlackProviderConfig{
		BotToken:   resolveServiceEnv(service, "SLACK_BOT_TOKEN"),
		BaseURL:    serviceOption(service, "slack_api_base_url"),
		Transport:  h.slackTransport,
		Normalizer: h.normalizer,
	})
	validatingProvider := &serviceValidatingProvider{
		base:      provider,
		validator: h.serviceValidator(),
		service:   serviceValidationConfig(service),
	}
	var deliveryAttempts []msgtypes.DeliveryAttempt
	runtime, err := msgtypes.NewRuntime(
		validatingProvider,
		h.router,
		&serviceRuntimeExecutor{router: h.router, service: service, chatRunner: h.chatRunner},
		runtimeOptionsWithDeliveryAttempts(serviceID, &deliveryAttempts),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("slack runtime failed: %v", err))
		return
	}

	result, err := runtime.HandleIngress(r.Context(), msgtypes.NativeIngress{
		ServiceID:  serviceID,
		Provider:   string(msgtypes.PlatformSlack),
		Mode:       msgtypes.IngressModeWebhook,
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header,
		Query:      r.URL.Query(),
		Body:       rawBody,
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		_ = recordServiceDeliveryAttempts(serviceID, deliveryAttempts, nil)
		writeProviderRuntimeError(w, string(msgtypes.PlatformSlack), err)
		return
	}
	if result != nil && result.Message != nil {
		_ = core.RecordServiceInboundEvent(serviceID, core.ServiceEventMeta{
			MessageID: result.Message.ID,
			Platform:  string(msgtypes.PlatformSlack),
			ChannelID: result.Message.Channel.ID,
			SenderID:  result.Message.Sender.ID,
			Status:    "received",
		})
		if result.Delivery != nil {
			_ = recordServiceDeliveryAttempts(serviceID, result.DeliveryAttempts, result.Message)
			if len(result.DeliveryAttempts) == 0 {
				_ = core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
					MessageID: result.Message.ID,
					Platform:  string(msgtypes.PlatformSlack),
					ChannelID: result.Message.Channel.ID,
					SenderID:  result.Message.Sender.ID,
					Status:    result.Delivery.Status,
				})
			}
		} else if result.Result != nil {
			_ = recordServiceExecutionEvent(serviceID, result.Message, result.Result)
		}
	}

	response := map[string]any{
		"status":    "accepted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if result != nil {
		if result.Message != nil {
			response["message_id"] = result.Message.ID
		}
		response["route"] = result.Route.TargetAction()
		if result.Result != nil {
			response["status"] = result.Result.Status
		}
		if result.Delivery != nil {
			response["delivery"] = result.Delivery
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleWhatsAppVerification(w http.ResponseWriter, r *http.Request, serviceID, adapter string) {
	service, err := core.LoadService(serviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q not found", serviceID))
		return
	}
	if service.Type != "message" || strings.TrimSpace(service.Adapter) != adapter {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q is not a WhatsApp message service", serviceID))
		return
	}
	challenge, err := msgtypes.WhatsAppVerificationChallenge(serviceValidationConfig(service), r.URL.Query())
	if err != nil {
		writeValidationError(w, err)
		return
	}
	writeText(w, http.StatusOK, challenge)
}

func (h *Handler) handleWhatsAppServiceWebhook(w http.ResponseWriter, r *http.Request, rawBody []byte, serviceID string, service *core.ServiceConfig) {
	provider := msgtypes.NewWhatsAppProvider(msgtypes.WhatsAppProviderConfig{
		ServiceID:       serviceID,
		Provider:        serviceProvider(service, "whatsapp-cloud"),
		AccessToken:     resolveServiceEnv(service, "WHATSAPP_ACCESS_TOKEN"),
		PhoneNumberID:   serviceOption(service, "phone_number_id"),
		From:            serviceOption(service, "from"),
		AccountSID:      resolveServiceEnv(service, "TWILIO_ACCOUNT_SID"),
		AuthToken:       resolveServiceEnv(service, "TWILIO_AUTH_TOKEN"),
		BaseURL:         serviceOption(service, "whatsapp_api_base_url"),
		TemplateName:    serviceOption(service, "template_name"),
		LanguageCode:    serviceOption(service, "language_code"),
		RequireTemplate: truthyHandlerOption(serviceOption(service, "template_required")),
	}, h.whatsappTransport)
	validatingProvider := &serviceValidatingProvider{
		base:      provider,
		validator: h.serviceValidator(),
		service:   serviceValidationConfig(service),
	}
	var deliveryAttempts []msgtypes.DeliveryAttempt
	runtime, err := msgtypes.NewRuntime(
		validatingProvider,
		h.router,
		&serviceRuntimeExecutor{router: h.router, service: service, chatRunner: h.chatRunner},
		runtimeOptionsWithDeliveryAttempts(serviceID, &deliveryAttempts),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("whatsapp runtime failed: %v", err))
		return
	}

	result, err := runtime.HandleIngress(r.Context(), msgtypes.NativeIngress{
		ServiceID:  serviceID,
		Provider:   string(msgtypes.PlatformWhatsApp),
		Mode:       msgtypes.IngressModeWebhook,
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header,
		Query:      r.URL.Query(),
		Body:       rawBody,
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		_ = recordServiceDeliveryAttempts(serviceID, deliveryAttempts, nil)
		writeProviderRuntimeError(w, string(msgtypes.PlatformWhatsApp), err)
		return
	}
	if result != nil && result.Message != nil {
		_ = core.RecordServiceInboundEvent(serviceID, core.ServiceEventMeta{
			MessageID: result.Message.ID,
			Platform:  string(msgtypes.PlatformWhatsApp),
			ChannelID: result.Message.Channel.ID,
			SenderID:  result.Message.Sender.ID,
			Status:    "received",
		})
		if result.Delivery != nil {
			_ = recordServiceDeliveryAttempts(serviceID, result.DeliveryAttempts, result.Message)
			if len(result.DeliveryAttempts) == 0 {
				_ = core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
					MessageID: result.Message.ID,
					Platform:  string(msgtypes.PlatformWhatsApp),
					ChannelID: result.Message.Channel.ID,
					SenderID:  result.Message.Sender.ID,
					Status:    result.Delivery.Status,
				})
			}
		} else if result.Result != nil {
			_ = recordServiceExecutionEvent(serviceID, result.Message, result.Result)
		}
	}

	response := map[string]any{
		"status":    "accepted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if result != nil {
		if result.Message != nil {
			response["message_id"] = result.Message.ID
		}
		response["route"] = result.Route.TargetAction()
		if result.Result != nil {
			response["status"] = result.Result.Status
		}
		if result.Delivery != nil {
			response["delivery"] = result.Delivery
		}
	}
	writeJSON(w, http.StatusOK, response)
}

type serviceValidatingProvider struct {
	base      msgtypes.MessageProvider
	validator *msgtypes.ServiceValidator
	service   msgtypes.ServiceValidationConfig
}

func (p *serviceValidatingProvider) Metadata() msgtypes.ProviderRuntimeMetadata {
	return p.base.Metadata()
}

func (p *serviceValidatingProvider) NormalizeIngress(ctx context.Context, ingress msgtypes.NativeIngress) (*msgtypes.NormalizedMessage, error) {
	msg, err := p.base.NormalizeIngress(ctx, ingress)
	if err != nil {
		return nil, err
	}
	if p.validator != nil {
		if err := p.validator.ValidateMessage(p.service, msg); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (p *serviceValidatingProvider) DeliverMessage(ctx context.Context, delivery msgtypes.DeliveryRequest) (*msgtypes.DeliveryReceipt, error) {
	deliverer, ok := p.base.(msgtypes.ProviderDelivery)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement delivery", p.base.Metadata().Provider)
	}
	return deliverer.DeliverMessage(ctx, delivery)
}

type serviceRuntimeExecutor struct {
	router     *MessageRouter
	service    *core.ServiceConfig
	chatRunner ChatTurnRunner
}

func (e *serviceRuntimeExecutor) ExecuteMessage(ctx context.Context, handoff msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	if serviceExecutionMode(e.service) == "chat" {
		return NewChatMessageExecutor(e.chatRunner, e.service).ExecuteMessage(ctx, handoff)
	}
	actionResult, err := e.router.ExecuteAction(ctx, handoff.ProfileID, handoff.ActionName, handoff.Message)
	if err != nil {
		return nil, err
	}
	result := &msgtypes.ExecutionResult{
		Status: actionResult.Status,
		Output: actionResult.Output,
	}
	if actionResult.Status != "success" || replyMode(e.service) == "none" || strings.TrimSpace(actionResult.Output) == "" {
		return result, nil
	}
	result.Reply = &msgtypes.DeliveryRequest{
		Text:     actionResult.Output,
		Metadata: replyMetadata(handoff.Message, e.service),
	}
	return result, nil
}

func runtimeOptionsWithDeliveryAttempts(serviceID string, attempts *[]msgtypes.DeliveryAttempt) msgtypes.RuntimeOptions {
	policy := msgtypes.DefaultRetryPolicy()
	return msgtypes.RuntimeOptions{
		ServiceID:   serviceID,
		RetryPolicy: policy,
		Hooks: msgtypes.RuntimeHooks{
			OnDeliveryAttempt: func(_ context.Context, attempt msgtypes.DeliveryAttempt) error {
				if attempts != nil {
					*attempts = append(*attempts, attempt)
				}
				return nil
			},
		},
	}
}

func recordServiceDeliveryAttempts(serviceID string, attempts []msgtypes.DeliveryAttempt, msg *msgtypes.NormalizedMessage) error {
	if len(attempts) == 0 {
		return nil
	}
	last := attempts[len(attempts)-1]
	at := last.FinishedAt
	if at.IsZero() {
		at = last.StartedAt
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	messageID := last.MessageID
	channelID := last.ChannelID
	senderID := ""
	if msg != nil {
		if messageID == "" {
			messageID = msg.ID
		}
		if channelID == "" {
			channelID = msg.Channel.ID
		}
		senderID = msg.Sender.ID
	}
	return core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
		At:          at,
		MessageID:   messageID,
		Platform:    last.Provider,
		ChannelID:   channelID,
		SenderID:    senderID,
		Status:      last.Status,
		Detail:      last.RedactedError,
		DeliveryID:  last.DeliveryID,
		Attempts:    serviceDeliveryAttempts(attempts),
		RetryPolicy: serviceRetryPolicy(msgtypes.DefaultRetryPolicy()),
	})
}

func recordServiceExecutionEvent(serviceID string, msg *msgtypes.NormalizedMessage, result *msgtypes.ExecutionResult) error {
	if msg == nil || result == nil {
		return nil
	}
	status := "executed"
	detail := "execution completed without provider delivery"
	switch strings.TrimSpace(strings.ToLower(result.Status)) {
	case "success", "completed":
	case "failed", "error", "timeout":
		status = "failed"
		detail = "execution failed before provider delivery"
	default:
		if strings.TrimSpace(result.Status) != "" {
			status = result.Status
		}
	}
	return core.RecordServiceOutboundEvent(serviceID, core.ServiceEventMeta{
		MessageID: msg.ID,
		Platform:  msg.Platform,
		ChannelID: msg.Channel.ID,
		SenderID:  msg.Sender.ID,
		Status:    status,
		Detail:    detail,
	})
}

func serviceDeliveryAttempts(attempts []msgtypes.DeliveryAttempt) []core.ServiceDeliveryAttempt {
	out := make([]core.ServiceDeliveryAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		at := attempt.FinishedAt
		if at.IsZero() {
			at = attempt.StartedAt
		}
		delay := ""
		if attempt.Delay > 0 {
			delay = attempt.Delay.String()
		}
		out = append(out, core.ServiceDeliveryAttempt{
			At:            at,
			Provider:      attempt.Provider,
			MessageID:     attempt.MessageID,
			ChannelID:     attempt.ChannelID,
			Attempt:       attempt.Attempt,
			MaxAttempts:   attempt.MaxAttempts,
			Status:        attempt.Status,
			DeliveryID:    attempt.DeliveryID,
			Retriable:     attempt.Retriable,
			Delay:         delay,
			RedactedError: attempt.RedactedError,
		})
	}
	return out
}

func serviceRetryPolicy(policy msgtypes.RetryPolicy) *core.ServiceRetryPolicy {
	return &core.ServiceRetryPolicy{
		MaxAttempts: policy.MaxAttempts,
		BaseDelay:   policy.BaseDelay.String(),
		MaxDelay:    policy.MaxDelay.String(),
	}
}

func replyMetadata(msg *msgtypes.NormalizedMessage, service *core.ServiceConfig) map[string]any {
	metadata := map[string]any{}
	if msg == nil {
		return metadata
	}
	switch msgtypes.MessengerPlatform(msg.Platform) {
	case msgtypes.PlatformTelegram:
		if msg.PlatformMetadata == nil {
			return metadata
		}
		if id, ok := msg.PlatformMetadata["telegram_message_id"]; ok {
			metadata["reply_to_message_id"] = id
		}
		if id, ok := msg.PlatformMetadata["telegram_message_thread_id"]; ok {
			metadata["message_thread_id"] = id
		}
	case msgtypes.PlatformWhatsApp:
		metadata["to"] = msg.Sender.ID
		metadata["phone_number_id"] = msg.Channel.ID
		metadata["provider"] = serviceProvider(service, "whatsapp-cloud")
		if templateName := serviceOption(service, "template_name"); templateName != "" {
			metadata["template_name"] = templateName
		}
		if languageCode := serviceOption(service, "language_code"); languageCode != "" {
			metadata["language_code"] = languageCode
		}
	}
	return metadata
}

func replyMode(service *core.ServiceConfig) string {
	if service == nil || service.Options == nil {
		return "text"
	}
	value := strings.TrimSpace(strings.ToLower(service.Options["reply"]))
	if value == "" || value == "auto" {
		return "text"
	}
	return value
}

func serviceExecutionMode(service *core.ServiceConfig) string {
	if service == nil || service.Options == nil {
		return "action"
	}
	value := strings.TrimSpace(strings.ToLower(service.Options["execution"]))
	if value == "" {
		return "action"
	}
	return value
}

func truthyHandlerOption(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func serviceOption(service *core.ServiceConfig, key string) string {
	if service == nil || service.Options == nil {
		return ""
	}
	return strings.TrimSpace(service.Options[key])
}

func serviceProvider(service *core.ServiceConfig, fallback string) string {
	if provider := serviceOption(service, "provider"); provider != "" {
		return provider
	}
	return fallback
}

func resolveServiceEnv(service *core.ServiceConfig, key string) string {
	if service != nil && service.Env != nil {
		value := strings.TrimSpace(service.Env[key])
		if strings.HasPrefix(value, "secret:") {
			if envValue := os.Getenv(strings.TrimSpace(strings.TrimPrefix(value, "secret:"))); envValue != "" {
				return envValue
			}
		}
		if value != "" && !strings.HasPrefix(value, "secret:") {
			return value
		}
	}
	return os.Getenv(key)
}

func writeRuntimeError(w http.ResponseWriter, err error) {
	writeProviderRuntimeError(w, string(msgtypes.PlatformTelegram), err)
}

func writeProviderRuntimeError(w http.ResponseWriter, provider string, err error) {
	var runtimeErr *msgtypes.RuntimeError
	if !errors.As(err, &runtimeErr) {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s runtime failed: %v", provider, err))
		return
	}
	switch runtimeErr.Stage {
	case msgtypes.RuntimeStageNormalize:
		if msgtypes.IsSenderNotAllowed(runtimeErr.Err) {
			writeValidationError(w, runtimeErr.Err)
			return
		}
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s normalization failed: %v", provider, runtimeErr.Err))
	case msgtypes.RuntimeStageRoute:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s route failed: %v", provider, runtimeErr.Err))
	case msgtypes.RuntimeStageExecute:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s execution failed: %v", provider, runtimeErr.Err))
	case msgtypes.RuntimeStageDeliver:
		writeError(w, http.StatusBadGateway, fmt.Sprintf("%s delivery failed: %v", provider, runtimeErr.Err))
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s runtime failed: %v", provider, runtimeErr.Err))
	}
}

// extractPlatform parses the platform name from a URL path. It expects the
// path to contain /messengers/{platform}/webhook and returns the platform
// segment. Returns empty string if the pattern is not found.
func extractPlatform(path string) string {
	// Normalize trailing slashes.
	path = strings.TrimRight(path, "/")

	segments := strings.Split(path, "/")
	// We need at least 3 segments for: .../{messengers}/{platform}/{webhook}
	if len(segments) < 3 {
		return ""
	}

	// Walk backwards: last segment should be "webhook", the one before is
	// the platform, and the one before that should be "messengers".
	last := segments[len(segments)-1]
	platform := segments[len(segments)-2]
	prefix := segments[len(segments)-3]

	if last != "webhook" || prefix != "messengers" {
		return ""
	}

	if platform == "" {
		return ""
	}

	return platform
}

// extractActionFromOutput attempts to extract the action name from the
// placeholder ExecuteAction output format for logging purposes.
func extractActionFromOutput(output string) string {
	// The placeholder output format is:
	//   action "X" dispatched to profile "Y" (...)
	const prefix = `action "`
	before, rest, ok := strings.Cut(output, prefix)
	if !ok {
		return ""
	}
	_ = before
	action, _, ok := strings.Cut(rest, `"`)
	if !ok {
		return ""
	}
	return action
}

// writeJSON serializes data as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body, err := json.Marshal(data)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}
	_, _ = w.Write(logging.ApplyBytes(body))
}

// hasAudioAttachment reports whether the message contains at least one audio attachment.
func hasAudioAttachment(msg *msgtypes.NormalizedMessage) bool {
	for _, att := range msg.Attachments {
		if att.Type == "audio" {
			return true
		}
	}
	return false
}

// writeError writes a JSON-formatted error response with the given HTTP status code.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body, err := json.Marshal(map[string]any{
		"error":     http.StatusText(status),
		"code":      status,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": http.StatusText(status), "code": status})
		return
	}
	_, _ = w.Write(logging.ApplyBytes(body))
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (h *Handler) serviceValidator() *msgtypes.ServiceValidator {
	if h.validator != nil {
		return h.validator
	}
	return msgtypes.NewServiceValidator()
}

func writeValidationError(w http.ResponseWriter, err error) {
	switch {
	case msgtypes.IsAuthFailed(err):
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("message provider validation failed: %v", err))
	case msgtypes.IsSenderNotAllowed(err):
		writeError(w, http.StatusForbidden, fmt.Sprintf("message sender not allowed: %v", err))
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("message provider validation failed: %v", err))
	}
}

func serviceValidationConfig(service *core.ServiceConfig) msgtypes.ServiceValidationConfig {
	if service == nil {
		return msgtypes.ServiceValidationConfig{}
	}
	return msgtypes.ServiceValidationConfig{
		ID:      service.ID,
		Adapter: service.Adapter,
		Env:     service.Env,
		Options: service.Options,
	}
}

func handleSlackAcknowledgement(w http.ResponseWriter, body map[string]any, validator *msgtypes.ServiceValidator, serviceID string, service *core.ServiceConfig) bool {
	if getString(body, "type") == "url_verification" {
		challenge := getString(body, "challenge")
		if challenge == "" {
			writeError(w, http.StatusBadRequest, "Slack URL verification payload is missing challenge")
			return true
		}
		writeText(w, http.StatusOK, challenge)
		return true
	}

	eventID := getString(body, "event_id")
	if serviceID != "" && eventID != "" && validator.MarkReplay(serviceID, "slack:event:"+eventID, slackDedupTTL(service)) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "duplicate",
			"event_id":   eventID,
			"message_id": eventID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
		return true
	}

	event := getMap(body, "event")
	if event == nil {
		return false
	}
	eventType := getString(event, "type")
	if eventType != "" && eventType != "message" && eventType != "app_mention" {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "ignored",
			"reason":    "unsupported_slack_event",
			"event_id":  eventID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return true
	}
	if getString(event, "bot_id") != "" || getString(event, "subtype") == "bot_message" {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "ignored",
			"reason":    "slack_bot_message",
			"event_id":  eventID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return true
	}
	return false
}

func slackDedupTTL(service *core.ServiceConfig) time.Duration {
	if service == nil || service.Options == nil {
		return 24 * time.Hour
	}
	value := strings.TrimSpace(service.Options["dedup_ttl"])
	if value == "" {
		return 24 * time.Hour
	}
	ttl, err := time.ParseDuration(value)
	if err != nil || ttl <= 0 {
		return 24 * time.Hour
	}
	return ttl
}

func decodeWebhookBody(platform string, headers http.Header, rawBody []byte) (map[string]any, error) {
	contentType := strings.ToLower(headers.Get("Content-Type"))
	switch msgtypes.MessengerPlatform(platform) {
	case msgtypes.PlatformSMS, msgtypes.PlatformWhatsApp:
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			values, err := url.ParseQuery(string(rawBody))
			if err != nil {
				return nil, fmt.Errorf("invalid %s form body: %v", platform, err)
			}
			return formToMap(values), nil
		}
	}
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %v", err)
	}
	return body, nil
}

func formValuesForValidation(headers http.Header, rawBody []byte) url.Values {
	if !strings.Contains(strings.ToLower(headers.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return nil
	}
	values, err := url.ParseQuery(string(rawBody))
	if err != nil {
		return nil
	}
	return values
}

func formToMap(values url.Values) map[string]any {
	out := make(map[string]any, len(values))
	for key, vals := range values {
		if len(vals) == 1 {
			out[key] = vals[0]
			continue
		}
		out[key] = append([]string(nil), vals...)
	}
	return out
}

func publicRequestURL(r *http.Request, service *core.ServiceConfig) string {
	if service != nil && service.Options != nil {
		if configured := strings.TrimSpace(service.Options["webhook_url"]); configured != "" {
			return configured
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return scheme + "://" + host + r.URL.RequestURI()
}
