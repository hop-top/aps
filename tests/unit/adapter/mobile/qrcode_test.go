package mobile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core/adapter/mobile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateQRTerminal(t *testing.T) {
	t.Run("generates non-empty output", func(t *testing.T) {
		result, err := mobile.GenerateQRTerminal("https://example.com/test")
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("uses Unicode half-block characters", func(t *testing.T) {
		result, err := mobile.GenerateQRTerminal("test-content")
		require.NoError(t, err)

		// Should contain at least one half-block character
		hasHalfBlock := strings.ContainsRune(result, '\u2580') || // upper half
			strings.ContainsRune(result, '\u2584') || // lower half
			strings.ContainsRune(result, '\u2588')    // full block
		assert.True(t, hasHalfBlock, "QR should contain Unicode half-block characters")
	})

	t.Run("has multiple lines", func(t *testing.T) {
		result, err := mobile.GenerateQRTerminal("multi-line-test")
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(result), "\n")
		assert.Greater(t, len(lines), 1, "QR code should span multiple lines")
	})
}

func TestGenerateQRPNG(t *testing.T) {
	t.Run("writes PNG file", func(t *testing.T) {
		dir := t.TempDir()
		outPath := filepath.Join(dir, "test-qr.png")

		err := mobile.GenerateQRPNG("https://example.com", outPath, 256)
		require.NoError(t, err)

		info, err := os.Stat(outPath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "PNG file should not be empty")
	})
}

func TestQRModuleCount(t *testing.T) {
	t.Run("returns positive count for valid content", func(t *testing.T) {
		count, err := mobile.QRModuleCount("test")
		require.NoError(t, err)
		assert.Greater(t, count, 0)
	})

	t.Run("longer content produces larger QR", func(t *testing.T) {
		short, err := mobile.QRModuleCount("hi")
		require.NoError(t, err)

		long, err := mobile.QRModuleCount(strings.Repeat("a", 200))
		require.NoError(t, err)

		assert.GreaterOrEqual(t, long, short)
	})
}
