package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/logging"
)

// maxWebhookBody bounds the inbound ticket payload read into memory.
const maxWebhookBody = 4 << 20

// responseStatusKey and responseBodyKey carry the action status and output
// in webhook responses.
const (
	responseStatusKey = "status"
	responseBodyKey   = "body"
)

// Handler serves the per-service ticket webhook: it loads the service, gates
// the request with the shared service auth (core/messenger ServiceValidator:
// bearer/token/HMAC/timestamp/replay from service options), normalizes the
// raw payload into a NormalizedTicket, applies the service sender allowlist,
// then routes and executes the profile action.
type Handler struct {
	router     *Router
	normalizer *Normalizer
	validator  *coremessenger.ServiceValidator
}

// NewHandler creates a Handler. A nil validator falls back to the default
// service validator.
func NewHandler(router *Router, normalizer *Normalizer, validator *coremessenger.ServiceValidator) *Handler {
	if validator == nil {
		validator = coremessenger.NewServiceValidator()
	}
	return &Handler{router: router, normalizer: normalizer, validator: validator}
}

// ServeHTTP expects to be mounted on POST /services/{service}/ticket/{adapter}.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST requests are accepted")
		return
	}
	serviceID := strings.TrimSpace(r.PathValue("service"))
	adapter := strings.ToLower(strings.TrimSpace(r.PathValue("adapter")))
	if serviceID == "" || adapter == "" {
		writeError(w, http.StatusNotFound, "expected /services/{service}/ticket/{adapter}")
		return
	}
	service, err := core.LoadService(serviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q not found", serviceID))
		return
	}
	if service.Type != ServiceType {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has type %q, not ticket", serviceID, service.Type))
		return
	}
	if strings.ToLower(strings.TrimSpace(service.Adapter)) != adapter {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q has adapter %q, not %q", serviceID, service.Adapter, adapter))
		return
	}
	h.handleServiceWebhook(w, r, service, adapter)
}

func (h *Handler) handleServiceWebhook(w http.ResponseWriter, r *http.Request, service *core.ServiceConfig, adapter string) {
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	if err := h.validator.ValidateRequest(r.Context(), coremessenger.RequestValidationInput{
		Service:    serviceValidationConfig(service),
		Method:     r.Method,
		URL:        publicRequestURL(r, service),
		Headers:    r.Header,
		Body:       rawBody,
		RemoteAddr: r.RemoteAddr,
	}); err != nil {
		writeValidationError(w, err)
		return
	}

	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}
	ticket, err := h.normalizer.Normalize(adapter, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("normalization failed: %v", err))
		return
	}
	if ticket.Metadata == nil {
		ticket.Metadata = map[string]any{}
	}
	ticket.Metadata[MetadataServiceID] = service.ID
	if err := validateTicketSender(service, ticket); err != nil {
		writeValidationError(w, err)
		return
	}
	_ = core.RecordServiceInboundEvent(service.ID, core.ServiceEventMeta{
		MessageID: ticket.ID,
		Platform:  adapter,
		ChannelID: ticket.ChannelID,
		SenderID:  ticket.Author.ID,
		Status:    "received",
	})

	result, err := h.router.HandleTicket(r.Context(), ticket)
	if err != nil {
		_ = core.RecordServiceOutboundEvent(service.ID, core.ServiceEventMeta{
			MessageID: ticket.ID,
			Platform:  adapter,
			ChannelID: ticket.ChannelID,
			SenderID:  ticket.Author.ID,
			Status:    StatusFailed,
			Detail:    err.Error(),
		})
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("ticket handling failed: %v", err))
		return
	}
	if result != nil && IsUnrouted(result.Error) {
		_ = core.RecordServiceOutboundEvent(service.ID, core.ServiceEventMeta{
			MessageID: ticket.ID,
			Platform:  adapter,
			ChannelID: ticket.ChannelID,
			SenderID:  ticket.Author.ID,
			Status:    "unrouted",
			Detail:    result.Error.Error(),
		})
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("service %q has no route for this ticket: set option %s or a routing table", service.ID, core.OptionDefaultAction))
		return
	}

	response, err := h.normalizer.Denormalize(adapter, result, ticket)
	if err != nil {
		response = map[string]any{responseStatusKey: result.Status, responseBodyKey: result.Output, "ticket_id": ticket.ID}
	}
	response["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	_ = core.RecordServiceOutboundEvent(service.ID, core.ServiceEventMeta{
		MessageID: ticket.ID,
		Platform:  adapter,
		ChannelID: ticket.ChannelID,
		SenderID:  ticket.Author.ID,
		Status:    result.Status,
		Detail:    result.Output,
	})
	writeJSON(w, http.StatusOK, response)
}

// validateTicketSender applies option allowed_senders (comma-separated
// addresses, IDs, handles, or globs such as *@corp.example) against the
// ticket author. Empty means every sender is accepted.
func validateTicketSender(service *core.ServiceConfig, ticket *NormalizedTicket) error {
	if service == nil || ticket == nil || service.Options == nil {
		return nil
	}
	raw := strings.TrimSpace(service.Options["allowed_senders"])
	if raw == "" {
		return nil
	}
	candidates := []string{ticket.Author.Email, ticket.Author.ID, ticket.Author.Handle}
	for _, pattern := range strings.Split(raw, ",") {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		for _, candidate := range candidates {
			candidate = strings.ToLower(strings.TrimSpace(candidate))
			if candidate == "" {
				continue
			}
			if candidate == pattern {
				return nil
			}
			if matched, err := path.Match(pattern, candidate); err == nil && matched {
				return nil
			}
		}
	}
	return fmt.Errorf("service %s: %w", service.ID, coremessenger.ErrSenderNotAllowed(service.ID, "ticket sender is not allowed"))
}

func serviceValidationConfig(service *core.ServiceConfig) coremessenger.ServiceValidationConfig {
	return coremessenger.ServiceValidationConfig{
		ID:      service.ID,
		Adapter: service.Adapter,
		Env:     service.Env,
		Options: service.Options,
	}
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

func writeValidationError(w http.ResponseWriter, err error) {
	switch {
	case coremessenger.IsAuthFailed(err):
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("ticket service validation failed: %v", err))
	case coremessenger.IsSenderNotAllowed(err):
		writeError(w, http.StatusForbidden, fmt.Sprintf("ticket sender not allowed: %v", err))
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("ticket service validation failed: %v", err))
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body, err := json.Marshal(data)
	if err != nil {
		_ = json.NewEncoder(logging.NewWriter(w)).Encode(data)
		return
	}
	_, _ = w.Write(logging.ApplyBytes(body))
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":     http.StatusText(status),
		"code":      status,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
