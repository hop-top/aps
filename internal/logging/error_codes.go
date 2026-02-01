package logging

// ErrorCode represents standardized error codes used throughout the application
type ErrorCode string

const (
	// Not Found errors
	ErrProfileNotFound     ErrorCode = "PROFILE_NOT_FOUND"
	ErrActionNotFound      ErrorCode = "ACTION_NOT_FOUND"
	ErrCapabilityNotFound  ErrorCode = "CAPABILITY_NOT_FOUND"
	ErrToolNotFound        ErrorCode = "TOOL_NOT_FOUND"
	ErrSessionNotFound     ErrorCode = "SESSION_NOT_FOUND"
	ErrRunNotFound         ErrorCode = "RUN_NOT_FOUND"

	// Invalid Input errors
	ErrInvalidInput        ErrorCode = "INVALID_INPUT"
	ErrInvalidConfig       ErrorCode = "INVALID_CONFIG"
	ErrInvalidIsolation    ErrorCode = "INVALID_ISOLATION"

	// Execution errors
	ErrExecutionFailed     ErrorCode = "EXECUTION_FAILED"
	ErrActionFailed        ErrorCode = "ACTION_FAILED"
	ErrStreamingFailed     ErrorCode = "STREAMING_FAILED"

	// Validation errors
	ErrValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrResourceExists      ErrorCode = "RESOURCE_EXISTS"

	// Permission errors
	ErrPermissionDenied    ErrorCode = "PERMISSION_DENIED"
	ErrAccessDenied        ErrorCode = "ACCESS_DENIED"

	// Internal errors
	ErrInternal            ErrorCode = "INTERNAL_ERROR"
	ErrUnknown             ErrorCode = "UNKNOWN_ERROR"
)

// String returns the error code as a string
func (ec ErrorCode) String() string {
	return string(ec)
}

// ErrorCodeForError maps core error types to error codes
func ErrorCodeForError(err error) ErrorCode {
	switch err.(type) {
	default:
		return ErrInternal
	}
}

// HTTPStatusForErrorCode maps error codes to HTTP status codes
func HTTPStatusForErrorCode(code ErrorCode) int {
	switch code {
	case ErrProfileNotFound, ErrActionNotFound, ErrCapabilityNotFound, ErrToolNotFound,
		ErrSessionNotFound, ErrRunNotFound:
		return 404

	case ErrInvalidInput, ErrInvalidConfig, ErrValidationFailed:
		return 400

	case ErrPermissionDenied, ErrAccessDenied:
		return 403

	case ErrResourceExists:
		return 409

	case ErrInternal, ErrExecutionFailed, ErrActionFailed, ErrStreamingFailed:
		return 500

	default:
		return 500
	}
}
