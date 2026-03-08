package mobile_test

import (
	"strings"
	"testing"
	"time"

	"oss-aps-cli/internal/core/adapter/mobile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePairingCode(t *testing.T) {
	t.Run("default format is 3 groups of 3", func(t *testing.T) {
		code, err := mobile.DefaultPairingCode()
		require.NoError(t, err)

		parts := strings.Split(code, "-")
		assert.Len(t, parts, 3)
		for _, part := range parts {
			assert.Len(t, part, 3)
		}
	})

	t.Run("no ambiguous characters", func(t *testing.T) {
		ambiguous := "0O1lI"
		for i := 0; i < 50; i++ {
			code, err := mobile.DefaultPairingCode()
			require.NoError(t, err)
			for _, ch := range ambiguous {
				assert.NotContains(t, code, string(ch),
					"pairing code should not contain ambiguous character %q", string(ch))
			}
		}
	})

	t.Run("codes are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code, err := mobile.DefaultPairingCode()
			require.NoError(t, err)
			assert.False(t, seen[code], "duplicate pairing code generated: %s", code)
			seen[code] = true
		}
	})

	t.Run("custom group size and count", func(t *testing.T) {
		code, err := mobile.GeneratePairingCode(4, 4)
		require.NoError(t, err)

		parts := strings.Split(code, "-")
		assert.Len(t, parts, 4)
		for _, part := range parts {
			assert.Len(t, part, 4)
		}
	})
}

func TestEncodePairingPayload(t *testing.T) {
	t.Run("round-trip encode/decode", func(t *testing.T) {
		payload := &mobile.QRPayload{
			Version:         "1.0",
			ProfileID:       "myagent",
			Endpoint:        "https://192.168.1.42:8443/aps/adapter/myagent",
			PairingCode:     "ABC-123-XYZ",
			ExpiresAt:       time.Now().Add(15 * time.Minute).Format(time.RFC3339),
			CertFingerprint: "sha256:abcdef1234567890",
			Capabilities:    []string{"run:stateless", "run:streaming"},
		}

		encoded, err := mobile.EncodePairingPayload(payload)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		decoded, err := mobile.DecodePairingPayload(encoded)
		require.NoError(t, err)

		assert.Equal(t, payload.Version, decoded.Version)
		assert.Equal(t, payload.ProfileID, decoded.ProfileID)
		assert.Equal(t, payload.Endpoint, decoded.Endpoint)
		assert.Equal(t, payload.PairingCode, decoded.PairingCode)
		assert.Equal(t, payload.CertFingerprint, decoded.CertFingerprint)
		assert.Equal(t, payload.Capabilities, decoded.Capabilities)
	})

	t.Run("invalid base64 returns error", func(t *testing.T) {
		_, err := mobile.DecodePairingPayload("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		// Valid base64 but not valid JSON
		_, err := mobile.DecodePairingPayload("bm90LWpzb24=") // "not-json"
		assert.Error(t, err)
	})
}

func TestNewQRPayload(t *testing.T) {
	payload := mobile.NewQRPayload(
		"myagent",
		"https://192.168.1.42:8443/aps/adapter/myagent",
		"ABC-123-XYZ",
		"sha256:abcdef",
		[]string{"run:stateless"},
		15*time.Minute,
	)

	assert.Equal(t, "1.0", payload.Version)
	assert.Equal(t, "myagent", payload.ProfileID)
	assert.Equal(t, "ABC-123-XYZ", payload.PairingCode)
	assert.Equal(t, "sha256:abcdef", payload.CertFingerprint)
	assert.Equal(t, []string{"run:stateless"}, payload.Capabilities)
	assert.NotEmpty(t, payload.ExpiresAt)
}

func TestIsPayloadExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		payload := mobile.NewQRPayload("p", "e", "c", "", nil, 15*time.Minute)
		assert.False(t, mobile.IsPayloadExpired(payload))
	})

	t.Run("expired", func(t *testing.T) {
		payload := &mobile.QRPayload{
			ExpiresAt: time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
		}
		assert.True(t, mobile.IsPayloadExpired(payload))
	})

	t.Run("invalid timestamp treated as expired", func(t *testing.T) {
		payload := &mobile.QRPayload{ExpiresAt: "not-a-time"}
		assert.True(t, mobile.IsPayloadExpired(payload))
	})
}
