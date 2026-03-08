package mobile_test

import (
	"testing"
	"time"

	"oss-aps-cli/internal/core/adapter/mobile"

	"github.com/stretchr/testify/assert"
)

func TestCapabilities(t *testing.T) {
	t.Run("AllCapabilities returns all known capabilities", func(t *testing.T) {
		all := mobile.AllCapabilities()
		assert.Len(t, all, 4)
		assert.Contains(t, all, mobile.CapRunStateless)
		assert.Contains(t, all, mobile.CapRunStreaming)
		assert.Contains(t, all, mobile.CapMonitorSessions)
		assert.Contains(t, all, mobile.CapMonitorLogs)
	})

	t.Run("DefaultCapabilities excludes monitor:logs", func(t *testing.T) {
		defaults := mobile.DefaultCapabilities()
		assert.Len(t, defaults, 3)
		assert.NotContains(t, defaults, mobile.CapMonitorLogs)
	})

	t.Run("IsValidCapability", func(t *testing.T) {
		assert.True(t, mobile.IsValidCapability("run:stateless"))
		assert.True(t, mobile.IsValidCapability("run:streaming"))
		assert.True(t, mobile.IsValidCapability("monitor:sessions"))
		assert.True(t, mobile.IsValidCapability("monitor:logs"))
		assert.False(t, mobile.IsValidCapability("invalid"))
		assert.False(t, mobile.IsValidCapability(""))
	})
}

func TestMobileAdapterIsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, d.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		assert.True(t, d.IsExpired())
	})
}

func TestMobileAdapterIsActive(t *testing.T) {
	t.Run("active and not expired", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			Status:    mobile.PairingStateActive,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		assert.True(t, d.IsActive())
	})

	t.Run("active but expired", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			Status:    mobile.PairingStateActive,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		assert.False(t, d.IsActive())
	})

	t.Run("pending is not active", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			Status:    mobile.PairingStatePending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, d.IsActive())
	})

	t.Run("revoked is not active", func(t *testing.T) {
		d := &mobile.MobileAdapter{
			Status:    mobile.PairingStateRevoked,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, d.IsActive())
	})
}

func TestMobileErrors(t *testing.T) {
	t.Run("ErrPairingExpired", func(t *testing.T) {
		err := mobile.ErrPairingExpired()
		assert.Contains(t, err.Error(), "expired")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodePairingExpired))
	})

	t.Run("ErrPairingInvalid", func(t *testing.T) {
		err := mobile.ErrPairingInvalid()
		assert.Contains(t, err.Error(), "invalid")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodePairingInvalid))
	})

	t.Run("ErrMobileAdapterNotFound", func(t *testing.T) {
		err := mobile.ErrMobileAdapterNotFound("dev-123")
		assert.Contains(t, err.Error(), "dev-123")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeAdapterNotFound))
	})

	t.Run("ErrMaxAdaptersReached", func(t *testing.T) {
		err := mobile.ErrMaxAdaptersReached("myagent", 10)
		assert.Contains(t, err.Error(), "maximum")
		assert.Contains(t, err.Error(), "10/10")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeMaxAdaptersReached))
	})

	t.Run("ErrTokenExpired", func(t *testing.T) {
		err := mobile.ErrTokenExpired()
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeTokenExpired))
	})

	t.Run("ErrTokenInvalid with cause", func(t *testing.T) {
		cause := assert.AnError
		err := mobile.ErrTokenInvalid(cause)
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeTokenInvalid))
		assert.ErrorIs(t, err, cause)
	})

	t.Run("ErrPortInUse", func(t *testing.T) {
		err := mobile.ErrPortInUse(8443, assert.AnError)
		assert.Contains(t, err.Error(), "8443")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodePortInUse))
	})

	t.Run("ErrMobileNotEnabled", func(t *testing.T) {
		err := mobile.ErrMobileNotEnabled("myagent")
		assert.Contains(t, err.Error(), "myagent")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeMobileNotEnabled))
	})

	t.Run("ErrCapabilityInvalid", func(t *testing.T) {
		err := mobile.ErrCapabilityInvalid("foo:bar")
		assert.Contains(t, err.Error(), "foo:bar")
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeCapabilityInvalid))
	})

	t.Run("IsMobileError returns false for non-mobile errors", func(t *testing.T) {
		assert.False(t, mobile.IsMobileError(assert.AnError, mobile.ErrCodeTokenInvalid))
	})
}
