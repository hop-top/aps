package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// malformed shapes the pinned reader rejects; each embeds the sentinel.
const leakSentinel = "SENTINEL-s3cr3t-value"

func malformedShapes() map[string]string {
	return map[string]string{
		"unterminated double": "GOOD=ok\nTOKEN=\"" + leakSentinel + "\nNEXT=" + leakSentinel + "2\n",
		"unterminated single": "GOOD=ok\nTOKEN='" + leakSentinel + "\nNEXT=x\n",
		"bad key char":        "GOOD=ok\nBAD!KEY" + leakSentinel + "=v\n",
		"bad key at start":    "-" + leakSentinel + "=v\n",
		"missing equals":      "GOOD=ok\n" + leakSentinel + "-no-equals\nAFTER=1\n",
		"space in word":       "GOOD=ok\nsk " + leakSentinel + "\n",
		"junk after quote":    "GOOD=ok\nK=\"v\" " + leakSentinel + "!\n",
	}
}

func TestLoadSecretsParseErrorWithholdsContent(t *testing.T) {
	for name, content := range malformedShapes() {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "secrets.env")
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr)
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
			defer slog.SetDefault(prev)

			_, err := LoadSecrets(path)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), leakSentinel)
			assert.NotContains(t, strings.ToLower(err.Error()), "sentinel")
			assert.Contains(t, err.Error(), path)
			assert.ErrorIs(t, err, ErrSecretsParse)
			var pe *SecretsParseError
			require.True(t, errors.As(err, &pe))
			assert.Equal(t, path, pe.Path)
			assert.False(t, errors.Is(err, os.ErrNotExist))

			// Chain: no wrapped error carries the text.
			for e := err; e != nil; e = errors.Unwrap(e) {
				assert.NotContains(t, e.Error(), leakSentinel)
			}

			// Wrapped by callers, still clean.
			assert.NotContains(t, errorsJoinString(err), leakSentinel)

			_, err = LoadResolvedSecrets("p", path)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), leakSentinel)

			_, err = newDotenvStore(path).Get(context.Background(), "GOOD")
			require.Error(t, err)
			assert.NotContains(t, err.Error(), leakSentinel)
			assert.ErrorIs(t, err, ErrSecretsParse)

			// Set must not touch a file it cannot parse.
			err = newDotenvStore(path).Set(context.Background(), "X", []byte("y"))
			require.Error(t, err)
			assert.NotContains(t, err.Error(), leakSentinel)

			assert.NotContains(t, logBuf.String(), leakSentinel)
		})
	}
}

func errorsJoinString(err error) string {
	return errors.Join(io.ErrUnexpectedEOF, err).Error()
}

func TestParseErrorLineNumbers(t *testing.T) {
	cases := []struct {
		name, content string
		line          int
	}{
		{"bad key line 2", "GOOD=ok\nBAD!KEY=v\n", 2},
		{"unterminated starts at its line", "A=1\n\nB=\"never closed\nC=2\n", 3},
		{"valid multiline value then bad", "A=\"l1\nl2\nl3\"\nB=2\nBAD!=x\n", 5},
		{"bad first line", "BAD!=x\nA=1\n", 1},
		{"crlf file", "A=1\r\nB=2\r\nBAD!=x\r\n", 3},
		{"after comments", "# c\n# d\nBAD!=x\n", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "s.env")
			require.NoError(t, os.WriteFile(path, []byte(c.content), 0o600))
			_, err := LoadSecrets(path)
			var pe *SecretsParseError
			require.True(t, errors.As(err, &pe), "err=%v", err)
			assert.Equal(t, c.line, pe.Line)
			assert.Contains(t, err.Error(), "line "+strconv.Itoa(c.line))
		})
	}
}

func TestLoadSecretsMissingFileStillNil(t *testing.T) {
	m, err := LoadSecrets(filepath.Join(t.TempDir(), "nope.env"))
	require.NoError(t, err)
	assert.Nil(t, m)
}
