package device_test

import (
	"errors"
	"testing"

	"oss-aps-cli/internal/core/device"

	"github.com/stretchr/testify/assert"
)

func TestDeviceErrors(t *testing.T) {
	t.Run("ErrDeviceNotFound", func(t *testing.T) {
		err := device.ErrDeviceNotFound("test-device")
		assert.True(t, device.IsDeviceNotFound(err))
		assert.Contains(t, err.Error(), "test-device")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ErrDeviceAlreadyExists", func(t *testing.T) {
		err := device.ErrDeviceAlreadyExists("test-device")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("ErrDeviceAlreadyRunning", func(t *testing.T) {
		err := device.ErrDeviceAlreadyRunning("test-device")
		assert.True(t, device.IsDeviceAlreadyRunning(err))
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("ErrDeviceAlreadyStopped", func(t *testing.T) {
		err := device.ErrDeviceAlreadyStopped("test-device")
		assert.True(t, device.IsDeviceAlreadyStopped(err))
		assert.Contains(t, err.Error(), "already stopped")
	})

	t.Run("ErrDeviceTypeNotImplemented", func(t *testing.T) {
		err := device.ErrDeviceTypeNotImplemented(device.DeviceTypeSense)
		assert.True(t, device.IsDeviceTypeNotImplemented(err))
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("ErrDeviceTypeInvalid", func(t *testing.T) {
		err := device.ErrDeviceTypeInvalid("foobar")
		assert.Contains(t, err.Error(), "invalid device type")
	})

	t.Run("ErrStrategyInvalid", func(t *testing.T) {
		err := device.ErrStrategyInvalid("foobar")
		assert.Contains(t, err.Error(), "invalid loading strategy")
	})

	t.Run("ErrStartFailed", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := device.ErrStartFailed("test-device", cause)
		assert.Contains(t, err.Error(), "failed to start")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrStopFailed", func(t *testing.T) {
		cause := errors.New("process not responding")
		err := device.ErrStopFailed("test-device", cause)
		assert.Contains(t, err.Error(), "failed to stop")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrHealthCheckFailed", func(t *testing.T) {
		cause := errors.New("timeout")
		err := device.ErrHealthCheckFailed("test-device", cause)
		assert.Contains(t, err.Error(), "health check failed")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrMigrationFailed", func(t *testing.T) {
		cause := errors.New("parse error")
		err := device.ErrMigrationFailed("test-device", cause)
		assert.Contains(t, err.Error(), "migration failed")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrManifestInvalid", func(t *testing.T) {
		cause := errors.New("yaml error")
		err := device.ErrManifestInvalid("/path/to/manifest.yaml", cause)
		assert.Contains(t, err.Error(), "invalid manifest")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrConfigInvalid", func(t *testing.T) {
		err := device.ErrConfigInvalid("test-device", "missing required field")
		assert.Contains(t, err.Error(), "missing required field")
	})
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected device.ErrorCode
	}{
		{"not found", device.ErrDeviceNotFound("test"), device.ErrCodeNotFound},
		{"already exists", device.ErrDeviceAlreadyExists("test"), device.ErrCodeAlreadyExists},
		{"already running", device.ErrDeviceAlreadyRunning("test"), device.ErrCodeAlreadyRunning},
		{"already stopped", device.ErrDeviceAlreadyStopped("test"), device.ErrCodeAlreadyStopped},
		{"type not implemented", device.ErrDeviceTypeNotImplemented(device.DeviceTypeSense), device.ErrCodeTypeNotImpl},
		{"type invalid", device.ErrDeviceTypeInvalid("foo"), device.ErrCodeTypeInvalid},
		{"strategy invalid", device.ErrStrategyInvalid("foo"), device.ErrCodeStrategyInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var devErr *device.DeviceError
			if errors.As(tt.err, &devErr) {
				assert.Equal(t, tt.expected, devErr.Code)
			}
		})
	}
}
