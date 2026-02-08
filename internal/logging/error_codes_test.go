package logging

import (
	"net/http"
	"testing"
)

// TestErrorCodeString tests error code string representation
func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrProfileNotFound, "PROFILE_NOT_FOUND"},
		{ErrActionNotFound, "ACTION_NOT_FOUND"},
		{ErrInvalidInput, "INVALID_INPUT"},
		{ErrExecutionFailed, "EXECUTION_FAILED"},
		{ErrValidationFailed, "VALIDATION_FAILED"},
		{ErrPermissionDenied, "PERMISSION_DENIED"},
		{ErrInternal, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if tt.code.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.code.String())
			}
		})
	}
}

// TestHTTPStatusForErrorCode tests HTTP status code mapping
func TestHTTPStatusForErrorCode(t *testing.T) {
	tests := []struct {
		code           ErrorCode
		expectedStatus int
		description    string
	}{
		// 404 errors
		{ErrProfileNotFound, http.StatusNotFound, "profile not found"},
		{ErrActionNotFound, http.StatusNotFound, "action not found"},
		{ErrCapabilityNotFound, http.StatusNotFound, "capability not found"},
		{ErrToolNotFound, http.StatusNotFound, "tool not found"},
		{ErrSessionNotFound, http.StatusNotFound, "session not found"},
		{ErrRunNotFound, http.StatusNotFound, "run not found"},

		// 400 errors
		{ErrInvalidInput, http.StatusBadRequest, "invalid input"},
		{ErrInvalidConfig, http.StatusBadRequest, "invalid config"},
		{ErrValidationFailed, http.StatusBadRequest, "validation failed"},

		// 403 errors
		{ErrPermissionDenied, http.StatusForbidden, "permission denied"},
		{ErrAccessDenied, http.StatusForbidden, "access denied"},

		// 409 errors
		{ErrResourceExists, http.StatusConflict, "resource exists"},

		// 500 errors
		{ErrInternal, http.StatusInternalServerError, "internal error"},
		{ErrExecutionFailed, http.StatusInternalServerError, "execution failed"},
		{ErrActionFailed, http.StatusInternalServerError, "action failed"},
		{ErrStreamingFailed, http.StatusInternalServerError, "streaming failed"},
		{ErrUnknown, http.StatusInternalServerError, "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			status := HTTPStatusForErrorCode(tt.code)
			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d for %s",
					tt.expectedStatus, status, tt.code.String())
			}
		})
	}
}

// TestHTTPStatusForUnknownCode tests unknown error code defaults to 500
func TestHTTPStatusForUnknownCode(t *testing.T) {
	unknownCode := ErrorCode("UNKNOWN_CODE")
	status := HTTPStatusForErrorCode(unknownCode)
	if status != http.StatusInternalServerError {
		t.Errorf("expected 500 for unknown error code, got %d", status)
	}
}

// TestErrorCodeGrouping tests that error codes are properly categorized
func TestErrorCodeGrouping(t *testing.T) {
	// NotFound errors should map to 404
	notFoundCodes := []ErrorCode{
		ErrProfileNotFound,
		ErrActionNotFound,
		ErrCapabilityNotFound,
		ErrToolNotFound,
		ErrSessionNotFound,
		ErrRunNotFound,
	}

	for _, code := range notFoundCodes {
		status := HTTPStatusForErrorCode(code)
		if status != http.StatusNotFound {
			t.Errorf("expected 404 for %s, got %d", code.String(), status)
		}
	}

	// Invalid input errors should map to 400
	invalidCodes := []ErrorCode{
		ErrInvalidInput,
		ErrInvalidConfig,
		ErrValidationFailed,
	}

	for _, code := range invalidCodes {
		status := HTTPStatusForErrorCode(code)
		if status != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", code.String(), status)
		}
	}

	// Execution error should map to 500
	execCodes := []ErrorCode{
		ErrExecutionFailed,
		ErrActionFailed,
		ErrStreamingFailed,
	}

	for _, code := range execCodes {
		status := HTTPStatusForErrorCode(code)
		if status != http.StatusInternalServerError {
			t.Errorf("expected 500 for %s, got %d", code.String(), status)
		}
	}
}

// BenchmarkHTTPStatusForErrorCode benchmarks the status code lookup
func BenchmarkHTTPStatusForErrorCode(b *testing.B) {
	code := ErrProfileNotFound
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HTTPStatusForErrorCode(code)
	}
}

// BenchmarkErrorCodeString benchmarks error code string conversion
func BenchmarkErrorCodeString(b *testing.B) {
	code := ErrInvalidInput
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = code.String()
	}
}
