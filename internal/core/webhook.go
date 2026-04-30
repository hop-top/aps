package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"hop.top/aps/internal/logging"
)

// WebhookEvent represents an incoming webhook
type WebhookEvent struct {
	ID         string            `json:"delivery_id"`
	Type       string            `json:"event"`
	Payload    []byte            `json:"-"`
	Headers    map[string]string `json:"-"`
	ReceivedAt time.Time         `json:"received_at"`
}

type WebhookServerConfig struct {
	Addr        string
	EventMap    map[string]string // event -> "profile:action"
	AllowEvents []string
	Secret      string
	DryRun      bool
}

// ServeWebhooks starts the webhook server
func ServeWebhooks(config WebhookServerConfig) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. Read Body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// 2. Validate Signature
		if config.Secret != "" {
			sigHeader := r.Header.Get("X-APS-Signature")
			if sigHeader == "" {
				http.Error(w, "Unauthorized: Missing signature", http.StatusUnauthorized)
				return
			}
			// Format: sha256=<hex>
			parts := strings.SplitN(sigHeader, "=", 2)
			if len(parts) != 2 || parts[0] != "sha256" {
				http.Error(w, "Unauthorized: Invalid signature format", http.StatusUnauthorized)
				return
			}

			if !CheckSignature(config.Secret, body, parts[1]) {
				http.Error(w, "Unauthorized: Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		// 3. Identify Event
		eventType := r.Header.Get("X-APS-Event")
		if eventType == "" {
			http.Error(w, "Missing X-APS-Event header", http.StatusBadRequest)
			return
		}

		// 4. Check Allowlist
		if len(config.AllowEvents) > 0 && !slices.Contains(config.AllowEvents, eventType) {
			http.Error(w, "Event not allowed", http.StatusForbidden)
			return
		}

		// 5. Route Event
		target, ok := config.EventMap[eventType]
		if !ok {
			// Spec clarification: 400 Bad Request if no mapping
			resp := map[string]string{"error": fmt.Sprintf("No mapping for event '%s'", eventType)}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(resp)
			return
		}

		parts := strings.SplitN(target, ":", 2)
		if len(parts) != 2 {
			http.Error(w, "Invalid mapping configuration", http.StatusInternalServerError)
			return
		}
		profileID, actionID := parts[0], parts[1]

		deliveryID := r.Header.Get("X-Request-Id")
		if deliveryID == "" {
			deliveryID = uuid.New().String()
		}

		// 6. Execute (or Dry Run)
		logging.GetLogger().Info("webhook event",
			"event", eventType, "delivery_id", deliveryID,
			"profile", profileID, "action", actionID)

		if config.DryRun {
			respondJSON(w, http.StatusOK, map[string]string{
				"status":      "dry_run",
				"delivery_id": deliveryID,
				"event":       eventType,
				"profile":     profileID,
				"action":      actionID,
			})
			return
		}

		// Run Action
		// We pass body as payload
		// How to handle response? Spec says "return action exit code" for CLI, but for webhook?
		// Spec says "Success response: status 200 OK".
		// It implies async dispatch mostly, but implementation is sync here via RunAction.
		// If action fails, should we return 500?
		// Spec 14.8: "Failure response: appropriate 4xx/5xx"

		err = RunAction(profileID, actionID, body)

		// TODO: RunAction currently streams stdout/err to os.Stdout/Stderr.
		// Webhook might want to capture it? Spec doesn't require capturing output in response, just status.
		// Logging rules: "execution result".

		if err != nil {
			logging.GetLogger().Error("webhook execution failed", err,
				"delivery_id", deliveryID, "event", eventType)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error":       err.Error(),
				"delivery_id": deliveryID,
				"event":       eventType,
			})
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"status":      "executed",
			"delivery_id": deliveryID,
			"event":       eventType,
			"profile":     profileID,
			"action":      actionID,
		})
	})

	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	logging.GetLogger().Info("webhook server listening", "addr", listener.Addr().String())
	return http.Serve(listener, mux)
}

func CheckSignature(secret string, body []byte, signatureHex string) bool {
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(expectedMAC, sigBytes)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
