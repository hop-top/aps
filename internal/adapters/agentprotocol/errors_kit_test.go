package agentprotocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hop.top/kit/go/runtime/domain"
)

// TestSendDomainError_MapsKitSentinels verifies sendDomainError routes
// kit's domain sentinels to the conventional HTTP statuses via
// kitapi.MapError. Drives the structured-error adoption in T-0350.
func TestSendDomainError_MapsKitSentinels(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"not_found", domain.ErrNotFound, http.StatusNotFound},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"validation", domain.ErrValidation, http.StatusUnprocessableEntity},
		{"invalid_transition", domain.ErrInvalidTransition, http.StatusConflict},
		{"wrapped_not_found", fmt.Errorf("agent: %w", domain.ErrNotFound), http.StatusNotFound},
		{"unknown", errors.New("totally unmapped"), http.StatusInternalServerError},
	}

	a := NewAgentProtocolAdapter()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			a.sendDomainError(w, tc.err)

			if w.Code != tc.status {
				t.Fatalf("status: got %d, want %d", w.Code, tc.status)
			}

			var body ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v; raw=%s", err, w.Body.String())
			}
			if body.Code != tc.status {
				t.Fatalf("body.Code: got %d, want %d", body.Code, tc.status)
			}
			if body.Message == "" {
				t.Fatalf("body.Message: empty")
			}
		})
	}
}
