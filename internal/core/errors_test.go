package core

import (
	"errors"
	"testing"
)

// TestNotFoundError tests NotFoundError creation and methods
func TestNotFoundError(t *testing.T) {
	resource := "test-profile"
	err := NewNotFoundError(resource)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.Resource != resource {
		t.Errorf("expected resource %s, got %s", resource, err.Resource)
	}

	expectedMsg := "not found: test-profile"
	if err.Error() != expectedMsg {
		t.Errorf("expected message '%s', got '%s'", expectedMsg, err.Error())
	}

	if err.GetCode() != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", err.GetCode().String())
	}
}

// TestNotFoundErrorWithCustomMessage tests NotFoundError with custom message
func TestNotFoundErrorWithCustomMessage(t *testing.T) {
	err := &NotFoundError{
		Resource: "action",
		Message:  "custom message",
	}

	if err.Error() != "custom message" {
		t.Errorf("expected 'custom message', got '%s'", err.Error())
	}
}

// TestNotFoundErrorWithCode tests NotFoundError with specific error code
func TestNotFoundErrorWithCode(t *testing.T) {
	resource := "test-action"
	code := ErrorCode("ACTION_NOT_FOUND")
	err := NewNotFoundErrorWithCode(resource, code)

	if err.GetCode() != code {
		t.Errorf("expected code %s, got %s", code.String(), err.GetCode().String())
	}
}

// TestInvalidInputError tests InvalidInputError creation and methods
func TestInvalidInputError(t *testing.T) {
	field := "agent_id"
	message := "agent_id is required"
	err := NewInvalidInputError(field, message)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.Field != field {
		t.Errorf("expected field %s, got %s", field, err.Field)
	}

	if err.Message != message {
		t.Errorf("expected message '%s', got '%s'", message, err.Message)
	}

	if err.Error() != message {
		t.Errorf("expected error message '%s', got '%s'", message, err.Error())
	}

	if err.GetCode() != "INVALID_INPUT" {
		t.Errorf("expected code INVALID_INPUT, got %s", err.GetCode().String())
	}
}

// TestInvalidInputErrorWithCode tests InvalidInputError with specific code
func TestInvalidInputErrorWithCode(t *testing.T) {
	field := "profile_id"
	message := "profile_id format invalid"
	code := ErrorCode("INVALID_CONFIG")
	err := NewInvalidInputErrorWithCode(field, message, code)

	if err.GetCode() != code {
		t.Errorf("expected code %s, got %s", code.String(), err.GetCode().String())
	}
}

// TestValidationError tests ValidationError creation and methods
func TestValidationError(t *testing.T) {
	field := "isolation_level"
	message := "isolation level must be one of: process, platform, container"
	err := NewValidationError(field, message)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.Field != field {
		t.Errorf("expected field %s, got %s", field, err.Field)
	}

	if err.Error() != message {
		t.Errorf("expected message '%s', got '%s'", message, err.Error())
	}

	if err.GetCode() != "VALIDATION_FAILED" {
		t.Errorf("expected code VALIDATION_FAILED, got %s", err.GetCode().String())
	}
}

// TestValidationErrorWithCode tests ValidationError with specific code
func TestValidationErrorWithCode(t *testing.T) {
	field := "config"
	message := "config validation failed"
	code := ErrorCode("INVALID_CONFIG")
	err := NewValidationErrorWithCode(field, message, code)

	if err.GetCode() != code {
		t.Errorf("expected code %s, got %s", code.String(), err.GetCode().String())
	}
}

// TestErrorCodeString tests ErrorCode string conversion
func TestErrorCodeString(t *testing.T) {
	code := ErrorCode("TEST_CODE")
	if code.String() != "TEST_CODE" {
		t.Errorf("expected 'TEST_CODE', got '%s'", code.String())
	}
}

// TestErrorsAsType tests that errors.As works with custom error types
func TestErrorsAsType(t *testing.T) {
	var err error = NewNotFoundError("test-resource")

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatal("expected errors.As to recognize NotFoundError")
	}

	if notFound.Resource != "test-resource" {
		t.Errorf("expected resource 'test-resource', got '%s'", notFound.Resource)
	}
}

// TestErrorInterface tests that custom errors implement error interface
func TestErrorInterface(t *testing.T) {
	tests := []error{
		NewNotFoundError("resource"),
		NewInvalidInputError("field", "message"),
		NewValidationError("field", "message"),
	}

	for i, err := range tests {
		if err == nil {
			t.Errorf("error %d is nil", i)
		}
		msg := err.Error()
		if len(msg) == 0 {
			t.Errorf("error %d has empty message", i)
		}
	}
}

// BenchmarkNotFoundError benchmarks NotFoundError creation
func BenchmarkNotFoundError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewNotFoundError("test-resource")
	}
}

// BenchmarkInvalidInputError benchmarks InvalidInputError creation
func BenchmarkInvalidInputError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewInvalidInputError("field", "message")
	}
}

// BenchmarkValidationError benchmarks ValidationError creation
func BenchmarkValidationError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewValidationError("field", "message")
	}
}
