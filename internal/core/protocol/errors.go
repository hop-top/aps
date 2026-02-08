package protocol

import "oss-aps-cli/internal/core"

// Re-export error types from core package
type NotFoundError = core.NotFoundError
type InvalidInputError = core.InvalidInputError
type ValidationError = core.ValidationError
type ErrorCode = core.ErrorCode

func NewNotFoundError(resource string) *NotFoundError {
	return core.NewNotFoundError(resource)
}

func NewNotFoundErrorWithCode(resource string, code ErrorCode) *NotFoundError {
	return core.NewNotFoundErrorWithCode(resource, code)
}

func NewInvalidInputError(field, message string) *InvalidInputError {
	return core.NewInvalidInputError(field, message)
}

func NewInvalidInputErrorWithCode(field, message string, code ErrorCode) *InvalidInputError {
	return core.NewInvalidInputErrorWithCode(field, message, code)
}

func NewValidationError(field, message string) *ValidationError {
	return core.NewValidationError(field, message)
}

func NewValidationErrorWithCode(field, message string, code ErrorCode) *ValidationError {
	return core.NewValidationErrorWithCode(field, message, code)
}

// ExecutionError represents an error during action execution
type ExecutionError struct {
	Action  string
	Message string
}

func (e *ExecutionError) Error() string {
	return e.Message
}

// NewExecutionError creates a new ExecutionError
func NewExecutionError(action, message string) *ExecutionError {
	return &ExecutionError{
		Action:  action,
		Message: message,
	}
}
