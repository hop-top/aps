package core

import "fmt"

// ErrorCode represents an error code for structured error logging
type ErrorCode string

// String returns the error code as a string
func (ec ErrorCode) String() string {
	return string(ec)
}

// NotFoundError represents a resource that was not found
type NotFoundError struct {
	Resource string
	Message  string
	Code     ErrorCode
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("not found: %s", e.Resource)
}

// GetCode returns the error code
func (e *NotFoundError) GetCode() ErrorCode {
	return e.Code
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(resource string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Message:  fmt.Sprintf("not found: %s", resource),
		Code:     "NOT_FOUND",
	}
}

// NewNotFoundErrorWithCode creates a new NotFoundError with a specific error code
func NewNotFoundErrorWithCode(resource string, code ErrorCode) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Message:  fmt.Sprintf("not found: %s", resource),
		Code:     code,
	}
}

// InvalidInputError represents invalid input or configuration
type InvalidInputError struct {
	Field   string
	Message string
	Code    ErrorCode
}

func (e *InvalidInputError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("invalid input: %s", e.Field)
}

// GetCode returns the error code
func (e *InvalidInputError) GetCode() ErrorCode {
	return e.Code
}

// NewInvalidInputError creates a new InvalidInputError
func NewInvalidInputError(field, message string) *InvalidInputError {
	return &InvalidInputError{
		Field:   field,
		Message: message,
		Code:    "INVALID_INPUT",
	}
}

// NewInvalidInputErrorWithCode creates a new InvalidInputError with a specific error code
func NewInvalidInputErrorWithCode(field, message string, code ErrorCode) *InvalidInputError {
	return &InvalidInputError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

// ValidationError represents validation failures
type ValidationError struct {
	Field   string
	Message string
	Code    ErrorCode
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("validation error: %s", e.Field)
}

// GetCode returns the error code
func (e *ValidationError) GetCode() ErrorCode {
	return e.Code
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Code:    "VALIDATION_FAILED",
	}
}

// NewValidationErrorWithCode creates a new ValidationError with a specific error code
func NewValidationErrorWithCode(field, message string, code ErrorCode) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}
