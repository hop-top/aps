package mobile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"oss-aps-cli/internal/core/device/mobile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTokenManager(t *testing.T) *mobile.TokenManager {
	t.Helper()
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "keys")
	tm, err := mobile.NewTokenManager("test-profile", keyDir)
	require.NoError(t, err)
	return tm
}

func TestTokenManagerCreation(t *testing.T) {
	t.Run("creates key directory", func(t *testing.T) {
		dir := t.TempDir()
		keyDir := filepath.Join(dir, "keys")

		_, err := mobile.NewTokenManager("test-profile", keyDir)
		require.NoError(t, err)

		_, err = os.Stat(keyDir)
		assert.NoError(t, err)
	})

	t.Run("generates RSA keys on first run", func(t *testing.T) {
		dir := t.TempDir()
		keyDir := filepath.Join(dir, "keys")

		_, err := mobile.NewTokenManager("test-profile", keyDir)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(keyDir, "device.key"))
		assert.NoError(t, err, "private key should exist")

		_, err = os.Stat(filepath.Join(keyDir, "device.pub"))
		assert.NoError(t, err, "public key should exist")
	})

	t.Run("reloads existing keys", func(t *testing.T) {
		dir := t.TempDir()
		keyDir := filepath.Join(dir, "keys")

		tm1, err := mobile.NewTokenManager("test-profile", keyDir)
		require.NoError(t, err)

		device := &mobile.MobileDevice{
			DeviceID:     "dev-1",
			DeviceName:   "Test Phone",
			DeviceOS:     "iOS",
			Capabilities: []string{"run:stateless"},
		}
		token1, err := tm1.CreateToken(device, time.Hour)
		require.NoError(t, err)

		// Create second manager with same keys
		tm2, err := mobile.NewTokenManager("test-profile", keyDir)
		require.NoError(t, err)

		// Should validate token from first manager
		claims, err := tm2.ValidateToken(token1)
		require.NoError(t, err)
		assert.Equal(t, "dev-1", claims.DeviceID)
	})
}

func TestCreateToken(t *testing.T) {
	tm := newTestTokenManager(t)

	t.Run("creates valid JWT", func(t *testing.T) {
		device := &mobile.MobileDevice{
			DeviceID:     "iphone-test-001",
			DeviceName:   "Test iPhone",
			DeviceOS:     "iOS",
			Capabilities: []string{"run:stateless", "run:streaming"},
		}

		token, err := tm.CreateToken(device, 24*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Contains(t, token, ".") // JWT has 3 dot-separated parts
	})

	t.Run("default expiry is 14 days", func(t *testing.T) {
		device := &mobile.MobileDevice{
			DeviceID:   "dev-default",
			DeviceName: "Default Expiry Phone",
			DeviceOS:   "Android",
		}

		token, err := tm.CreateToken(device, 0) // 0 = use default
		require.NoError(t, err)

		claims, err := tm.ValidateToken(token)
		require.NoError(t, err)

		expectedExpiry := time.Now().Add(14 * 24 * time.Hour)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
	})

	t.Run("custom expiry", func(t *testing.T) {
		device := &mobile.MobileDevice{
			DeviceID:   "dev-custom",
			DeviceName: "Custom Expiry Phone",
			DeviceOS:   "iOS",
		}

		token, err := tm.CreateToken(device, 1*time.Hour)
		require.NoError(t, err)

		claims, err := tm.ValidateToken(token)
		require.NoError(t, err)

		expectedExpiry := time.Now().Add(1 * time.Hour)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
	})
}

func TestValidateToken(t *testing.T) {
	tm := newTestTokenManager(t)

	t.Run("valid token returns correct claims", func(t *testing.T) {
		device := &mobile.MobileDevice{
			DeviceID:     "dev-claims",
			DeviceName:   "Claims Phone",
			DeviceOS:     "Android",
			Capabilities: []string{"run:stateless", "monitor:sessions"},
		}

		token, err := tm.CreateToken(device, time.Hour)
		require.NoError(t, err)

		claims, err := tm.ValidateToken(token)
		require.NoError(t, err)

		assert.Equal(t, "dev-claims", claims.DeviceID)
		assert.Equal(t, "Claims Phone", claims.DeviceName)
		assert.Equal(t, "Android", claims.DeviceOS)
		assert.Equal(t, "test-profile", claims.ProfileID)
		assert.Equal(t, []string{"run:stateless", "monitor:sessions"}, claims.Capabilities)
		assert.Equal(t, "aps", claims.Issuer)
	})

	t.Run("tampered token fails validation", func(t *testing.T) {
		device := &mobile.MobileDevice{
			DeviceID: "dev-tamper",
		}
		token, err := tm.CreateToken(device, time.Hour)
		require.NoError(t, err)

		// Tamper with the token
		tampered := token[:len(token)-5] + "XXXXX"
		_, err = tm.ValidateToken(tampered)
		assert.Error(t, err)
		assert.True(t, mobile.IsMobileError(err, mobile.ErrCodeTokenInvalid))
	})

	t.Run("garbage token fails validation", func(t *testing.T) {
		_, err := tm.ValidateToken("not.a.valid.token")
		assert.Error(t, err)
	})

	t.Run("token from different key fails", func(t *testing.T) {
		tm2 := newTestTokenManager(t) // different keys
		device := &mobile.MobileDevice{DeviceID: "dev-cross"}
		token, err := tm2.CreateToken(device, time.Hour)
		require.NoError(t, err)

		_, err = tm.ValidateToken(token)
		assert.Error(t, err, "token signed by different key should fail")
	})
}

func TestHashToken(t *testing.T) {
	t.Run("returns sha256 prefixed hash", func(t *testing.T) {
		hash := mobile.HashToken("test-token-string")
		assert.True(t, len(hash) > 7, "hash should be longer than prefix")
		assert.Equal(t, "sha256:", hash[:7])
	})

	t.Run("same input produces same hash", func(t *testing.T) {
		h1 := mobile.HashToken("same-token")
		h2 := mobile.HashToken("same-token")
		assert.Equal(t, h1, h2)
	})

	t.Run("different input produces different hash", func(t *testing.T) {
		h1 := mobile.HashToken("token-a")
		h2 := mobile.HashToken("token-b")
		assert.NotEqual(t, h1, h2)
	})
}

func TestCertFingerprint(t *testing.T) {
	tm := newTestTokenManager(t)

	fingerprint, err := tm.CertFingerprint()
	require.NoError(t, err)
	assert.True(t, len(fingerprint) > 7)
	assert.Equal(t, "sha256:", fingerprint[:7])
}
