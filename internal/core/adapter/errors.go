package adapter

import (
	"fmt"
)

type ErrorCode string

const (
	ErrCodeNotFound        ErrorCode = "adapter_not_found"
	ErrCodeAlreadyExists   ErrorCode = "adapter_already_exists"
	ErrCodeAlreadyRunning  ErrorCode = "adapter_already_running"
	ErrCodeAlreadyStopped  ErrorCode = "adapter_already_stopped"
	ErrCodeTypeNotImpl     ErrorCode = "adapter_type_not_implemented"
	ErrCodeTypeInvalid     ErrorCode = "adapter_type_invalid"
	ErrCodeStrategyInvalid ErrorCode = "strategy_invalid"
	ErrCodeStartFailed     ErrorCode = "adapter_start_failed"
	ErrCodeStopFailed      ErrorCode = "adapter_stop_failed"
	ErrCodeHealthCheck     ErrorCode = "health_check_failed"
	ErrCodeMigrationFailed ErrorCode = "migration_failed"
	ErrCodeManifestInvalid ErrorCode = "manifest_invalid"
	ErrCodeConfigInvalid   ErrorCode = "config_invalid"
)

type AdapterError struct {
	Name    string
	Message string
	Code    ErrorCode
	Cause   error
}

func (e *AdapterError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func (e *AdapterError) Unwrap() error {
	return e.Cause
}

func ErrAdapterNotFound(name string) error {
	return &AdapterError{
		Name:    name,
		Message: "adapter not found",
		Code:    ErrCodeNotFound,
	}
}

func ErrAdapterAlreadyExists(name string) error {
	return &AdapterError{
		Name:    name,
		Message: "adapter already exists",
		Code:    ErrCodeAlreadyExists,
	}
}

func ErrAdapterAlreadyRunning(name string) error {
	return &AdapterError{
		Name:    name,
		Message: "adapter is already running",
		Code:    ErrCodeAlreadyRunning,
	}
}

func ErrAdapterAlreadyStopped(name string) error {
	return &AdapterError{
		Name:    name,
		Message: "adapter is already stopped",
		Code:    ErrCodeAlreadyStopped,
	}
}

func ErrAdapterTypeNotImplemented(t AdapterType) error {
	return &AdapterError{
		Name:    string(t),
		Message: fmt.Sprintf("adapter type '%s' is not yet implemented", t),
		Code:    ErrCodeTypeNotImpl,
	}
}

func ErrAdapterTypeInvalid(t string) error {
	return &AdapterError{
		Name:    t,
		Message: fmt.Sprintf("invalid adapter type '%s'", t),
		Code:    ErrCodeTypeInvalid,
	}
}

func ErrStrategyInvalid(s string) error {
	return &AdapterError{
		Name:    s,
		Message: fmt.Sprintf("invalid loading strategy '%s'", s),
		Code:    ErrCodeStrategyInvalid,
	}
}

func ErrStartFailed(name string, cause error) error {
	return &AdapterError{
		Name:    name,
		Message: "failed to start adapter",
		Code:    ErrCodeStartFailed,
		Cause:   cause,
	}
}

func ErrStopFailed(name string, cause error) error {
	return &AdapterError{
		Name:    name,
		Message: "failed to stop adapter",
		Code:    ErrCodeStopFailed,
		Cause:   cause,
	}
}

func ErrHealthCheckFailed(name string, cause error) error {
	return &AdapterError{
		Name:    name,
		Message: "health check failed",
		Code:    ErrCodeHealthCheck,
		Cause:   cause,
	}
}

func ErrMigrationFailed(name string, cause error) error {
	return &AdapterError{
		Name:    name,
		Message: "migration failed",
		Code:    ErrCodeMigrationFailed,
		Cause:   cause,
	}
}

func ErrManifestInvalid(path string, cause error) error {
	return &AdapterError{
		Name:    path,
		Message: "invalid manifest",
		Code:    ErrCodeManifestInvalid,
		Cause:   cause,
	}
}

func ErrConfigInvalid(name string, message string) error {
	return &AdapterError{
		Name:    name,
		Message: message,
		Code:    ErrCodeConfigInvalid,
	}
}

func IsAdapterNotFound(err error) bool {
	if e, ok := err.(*AdapterError); ok {
		return e.Code == ErrCodeNotFound
	}
	return false
}

func IsAdapterAlreadyRunning(err error) bool {
	if e, ok := err.(*AdapterError); ok {
		return e.Code == ErrCodeAlreadyRunning
	}
	return false
}

func IsAdapterAlreadyStopped(err error) bool {
	if e, ok := err.(*AdapterError); ok {
		return e.Code == ErrCodeAlreadyStopped
	}
	return false
}

func IsAdapterTypeNotImplemented(err error) bool {
	if e, ok := err.(*AdapterError); ok {
		return e.Code == ErrCodeTypeNotImpl
	}
	return false
}

func IsStrategyInvalid(err error) bool {
	if e, ok := err.(*AdapterError); ok {
		return e.Code == ErrCodeStrategyInvalid
	}
	return false
}
