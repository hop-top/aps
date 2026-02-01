package acp

import "fmt"

// ErrorCode represents JSON-RPC 2.0 error codes plus ACP-specific codes
type ErrorCode int

const (
	// Standard JSON-RPC 2.0 error codes
	ErrCodeParseError     ErrorCode = -32700 // Invalid JSON was received
	ErrCodeInvalidRequest ErrorCode = -32600 // JSON sent is not a valid Request
	ErrCodeMethodNotFound ErrorCode = -32601 // Method does not exist
	ErrCodeInvalidParams  ErrorCode = -32602 // Invalid method parameters
	ErrCodeInternalError  ErrorCode = -32603 // Internal JSON-RPC error

	// ACP-specific error codes
	ErrCodeAuthRequired   ErrorCode = -32000 // Authentication required
	ErrCodeResourceNotFound ErrorCode = -32002 // Resource not found
	ErrCodePermissionDenied ErrorCode = -32003 // Permission denied for operation
	ErrCodeOperationNotAllowed ErrorCode = -32004 // Operation not allowed in current mode
	ErrCodeSessionEnded   ErrorCode = -32005 // Session has ended
	ErrCodeNotImplemented ErrorCode = -32006 // Method not yet implemented
)

// ErrorResponse represents a JSON-RPC 2.0 error response
type ErrorResponse struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewErrorResponse creates an error response
func NewErrorResponse(code ErrorCode, message string, data interface{}) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

var (
	ErrAuthRequired       = NewErrorResponse(ErrCodeAuthRequired, "authentication required", nil)
	ErrMethodNotFound     = NewErrorResponse(ErrCodeMethodNotFound, "method not found", nil)
	ErrInvalidParams      = NewErrorResponse(ErrCodeInvalidParams, "invalid parameters", nil)
	ErrInternalError      = NewErrorResponse(ErrCodeInternalError, "internal server error", nil)
	ErrResourceNotFound   = NewErrorResponse(ErrCodeResourceNotFound, "resource not found", nil)
	ErrPermissionDenied   = NewErrorResponse(ErrCodePermissionDenied, "permission denied", nil)
	ErrOperationNotAllowed = NewErrorResponse(ErrCodeOperationNotAllowed, "operation not allowed", nil)
	ErrSessionEnded       = NewErrorResponse(ErrCodeSessionEnded, "session has ended", nil)
	ErrNotImplemented     = NewErrorResponse(ErrCodeNotImplemented, "not implemented", nil)
)
