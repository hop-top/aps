package messenger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	msgtypes "hop.top/aps/internal/core/messenger"
)

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
	router     *MessageRouter
	normalizer *Normalizer
	logger     MessageLogger
}

// NewHandler creates a Handler with the given router, normalizer, and optional
// logger. The logger may be nil, in which case message logging is skipped.
func NewHandler(router *MessageRouter, normalizer *Normalizer, logger MessageLogger) *Handler {
	return &Handler{
		router:     router,
		normalizer: normalizer,
		logger:     logger,
	}
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

// handleWebhook processes a single webhook event for the given platform.
func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request, platform string) {
	// Decode the raw JSON payload.
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}

	// Normalize the platform-specific event into a NormalizedMessage.
	msg, err := h.normalizer.Normalize(platform, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("normalization failed: %v", err))
		return
	}

	// Log the received message if a logger is available.
	if h.logger != nil {
		// Log errors are not fatal to the processing pipeline.
		_ = h.logger.LogMessageReceived(msg)
	}

	// Route and execute the action.
	result, err := h.router.HandleMessage(r.Context(), msg)
	if err != nil {
		// Log the failure if possible.
		if h.logger != nil {
			_ = h.logger.LogActionExecuted(msg.ID, "", "failed", 0)
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

	writeJSON(w, http.StatusOK, response)
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
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON-formatted error response with the given HTTP status code.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error":     http.StatusText(status),
		"code":      status,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
