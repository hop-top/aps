package adapter_test

import (
	"errors"
	"testing"

	"hop.top/aps/internal/core/adapter"

	"github.com/stretchr/testify/assert"
)

func TestAdapterErrors(t *testing.T) {
	t.Run("ErrAdapterNotFound", func(t *testing.T) {
		err := adapter.ErrAdapterNotFound("test-adapter")
		assert.True(t, adapter.IsAdapterNotFound(err))
		assert.Contains(t, err.Error(), "test-adapter")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ErrAdapterAlreadyExists", func(t *testing.T) {
		err := adapter.ErrAdapterAlreadyExists("test-adapter")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("ErrAdapterAlreadyRunning", func(t *testing.T) {
		err := adapter.ErrAdapterAlreadyRunning("test-adapter")
		assert.True(t, adapter.IsAdapterAlreadyRunning(err))
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("ErrAdapterAlreadyStopped", func(t *testing.T) {
		err := adapter.ErrAdapterAlreadyStopped("test-adapter")
		assert.True(t, adapter.IsAdapterAlreadyStopped(err))
		assert.Contains(t, err.Error(), "already stopped")
	})

	t.Run("ErrAdapterTypeNotImplemented", func(t *testing.T) {
		err := adapter.ErrAdapterTypeNotImplemented(adapter.AdapterTypeSense)
		assert.True(t, adapter.IsAdapterTypeNotImplemented(err))
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("ErrAdapterTypeInvalid", func(t *testing.T) {
		err := adapter.ErrAdapterTypeInvalid("foobar")
		assert.Contains(t, err.Error(), "invalid adapter type")
	})

	t.Run("ErrStrategyInvalid", func(t *testing.T) {
		err := adapter.ErrStrategyInvalid("foobar")
		assert.Contains(t, err.Error(), "invalid loading strategy")
	})

	t.Run("ErrStartFailed", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := adapter.ErrStartFailed("test-adapter", cause)
		assert.Contains(t, err.Error(), "failed to start")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrStopFailed", func(t *testing.T) {
		cause := errors.New("process not responding")
		err := adapter.ErrStopFailed("test-adapter", cause)
		assert.Contains(t, err.Error(), "failed to stop")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrHealthCheckFailed", func(t *testing.T) {
		cause := errors.New("timeout")
		err := adapter.ErrHealthCheckFailed("test-adapter", cause)
		assert.Contains(t, err.Error(), "health check failed")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrMigrationFailed", func(t *testing.T) {
		cause := errors.New("parse error")
		err := adapter.ErrMigrationFailed("test-adapter", cause)
		assert.Contains(t, err.Error(), "migration failed")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrManifestInvalid", func(t *testing.T) {
		cause := errors.New("yaml error")
		err := adapter.ErrManifestInvalid("/path/to/manifest.yaml", cause)
		assert.Contains(t, err.Error(), "invalid manifest")
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("ErrConfigInvalid", func(t *testing.T) {
		err := adapter.ErrConfigInvalid("test-adapter", "missing required field")
		assert.Contains(t, err.Error(), "missing required field")
	})
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected adapter.ErrorCode
	}{
		{"not found", adapter.ErrAdapterNotFound("test"), adapter.ErrCodeNotFound},
		{"already exists", adapter.ErrAdapterAlreadyExists("test"), adapter.ErrCodeAlreadyExists},
		{"already running", adapter.ErrAdapterAlreadyRunning("test"), adapter.ErrCodeAlreadyRunning},
		{"already stopped", adapter.ErrAdapterAlreadyStopped("test"), adapter.ErrCodeAlreadyStopped},
		{"type not implemented", adapter.ErrAdapterTypeNotImplemented(adapter.AdapterTypeSense), adapter.ErrCodeTypeNotImpl},
		{"type invalid", adapter.ErrAdapterTypeInvalid("foo"), adapter.ErrCodeTypeInvalid},
		{"strategy invalid", adapter.ErrStrategyInvalid("foo"), adapter.ErrCodeStrategyInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var adapterErr *adapter.AdapterError
			if errors.As(tt.err, &adapterErr) {
				assert.Equal(t, tt.expected, adapterErr.Code)
			}
		})
	}
}
