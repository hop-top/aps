package device

import (
	"fmt"
)

type ErrorCode string

const (
	ErrCodeNotFound        ErrorCode = "device_not_found"
	ErrCodeAlreadyExists   ErrorCode = "device_already_exists"
	ErrCodeAlreadyRunning  ErrorCode = "device_already_running"
	ErrCodeAlreadyStopped  ErrorCode = "device_already_stopped"
	ErrCodeTypeNotImpl     ErrorCode = "device_type_not_implemented"
	ErrCodeTypeInvalid     ErrorCode = "device_type_invalid"
	ErrCodeStrategyInvalid ErrorCode = "strategy_invalid"
	ErrCodeStartFailed     ErrorCode = "device_start_failed"
	ErrCodeStopFailed      ErrorCode = "device_stop_failed"
	ErrCodeHealthCheck     ErrorCode = "health_check_failed"
	ErrCodeMigrationFailed ErrorCode = "migration_failed"
	ErrCodeManifestInvalid ErrorCode = "manifest_invalid"
	ErrCodeConfigInvalid   ErrorCode = "config_invalid"
)

type DeviceError struct {
	Name    string
	Message string
	Code    ErrorCode
	Cause   error
}

func (e *DeviceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func (e *DeviceError) Unwrap() error {
	return e.Cause
}

func ErrDeviceNotFound(name string) error {
	return &DeviceError{
		Name:    name,
		Message: "device not found",
		Code:    ErrCodeNotFound,
	}
}

func ErrDeviceAlreadyExists(name string) error {
	return &DeviceError{
		Name:    name,
		Message: "device already exists",
		Code:    ErrCodeAlreadyExists,
	}
}

func ErrDeviceAlreadyRunning(name string) error {
	return &DeviceError{
		Name:    name,
		Message: "device is already running",
		Code:    ErrCodeAlreadyRunning,
	}
}

func ErrDeviceAlreadyStopped(name string) error {
	return &DeviceError{
		Name:    name,
		Message: "device is already stopped",
		Code:    ErrCodeAlreadyStopped,
	}
}

func ErrDeviceTypeNotImplemented(t DeviceType) error {
	return &DeviceError{
		Name:    string(t),
		Message: fmt.Sprintf("device type '%s' is not yet implemented", t),
		Code:    ErrCodeTypeNotImpl,
	}
}

func ErrDeviceTypeInvalid(t string) error {
	return &DeviceError{
		Name:    t,
		Message: fmt.Sprintf("invalid device type '%s'", t),
		Code:    ErrCodeTypeInvalid,
	}
}

func ErrStrategyInvalid(s string) error {
	return &DeviceError{
		Name:    s,
		Message: fmt.Sprintf("invalid loading strategy '%s'", s),
		Code:    ErrCodeStrategyInvalid,
	}
}

func ErrStartFailed(name string, cause error) error {
	return &DeviceError{
		Name:    name,
		Message: "failed to start device",
		Code:    ErrCodeStartFailed,
		Cause:   cause,
	}
}

func ErrStopFailed(name string, cause error) error {
	return &DeviceError{
		Name:    name,
		Message: "failed to stop device",
		Code:    ErrCodeStopFailed,
		Cause:   cause,
	}
}

func ErrHealthCheckFailed(name string, cause error) error {
	return &DeviceError{
		Name:    name,
		Message: "health check failed",
		Code:    ErrCodeHealthCheck,
		Cause:   cause,
	}
}

func ErrMigrationFailed(name string, cause error) error {
	return &DeviceError{
		Name:    name,
		Message: "migration failed",
		Code:    ErrCodeMigrationFailed,
		Cause:   cause,
	}
}

func ErrManifestInvalid(path string, cause error) error {
	return &DeviceError{
		Name:    path,
		Message: "invalid manifest",
		Code:    ErrCodeManifestInvalid,
		Cause:   cause,
	}
}

func ErrConfigInvalid(name string, message string) error {
	return &DeviceError{
		Name:    name,
		Message: message,
		Code:    ErrCodeConfigInvalid,
	}
}

func IsDeviceNotFound(err error) bool {
	if e, ok := err.(*DeviceError); ok {
		return e.Code == ErrCodeNotFound
	}
	return false
}

func IsDeviceAlreadyRunning(err error) bool {
	if e, ok := err.(*DeviceError); ok {
		return e.Code == ErrCodeAlreadyRunning
	}
	return false
}

func IsDeviceAlreadyStopped(err error) bool {
	if e, ok := err.(*DeviceError); ok {
		return e.Code == ErrCodeAlreadyStopped
	}
	return false
}

func IsDeviceTypeNotImplemented(err error) bool {
	if e, ok := err.(*DeviceError); ok {
		return e.Code == ErrCodeTypeNotImpl
	}
	return false
}
