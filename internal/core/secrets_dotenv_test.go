package core

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/kit/go/storage/secret"
)

// roundTripCases are values that must survive Set then Get byte for byte.
var roundTripCases = map[string]string{
	"empty":              "",
	"plain":              "abc123",
	"space hash":         "abc #def",
	"tab hash":           "abc\t#def",
	"leading hash":       "#abc",
	"only hash":          "#",
	"newline":            "line1\nline2",
	"trailing newline":   "line\n",
	"cr":                 "a\rb",
	"crlf":               "a\r\nb",
	"tab":                "a\tb",
	"single quote":       "it's",
	"double quote":       `say "hi"`,
	"both quotes":        `it's "x"`,
	"quoted whole":       `'abc'`,
	"dquoted whole":      `"abc"`,
	"leading dquote":     `"abc`,
	"trailing dquote":    `abc"`,
	"backslash":          `a\b`,
	"trailing backslash": `C:\dir\`,
	"backslash n text":   `a\nb`,
	"dollar":             "pa$$word",
	"dollar var":         "$HOME",
	"dollar brace":       "${HOME}",
	"escaped dollar":     `\$HOME`,
	"dollar paren":       "$(id)",
	"leading space":      "  lead",
	"trailing space":     "trail  ",
	"only spaces":        "   ",
	"equals":             "a=b=c",
	"colon":              "a:b",
	"unicode":            "héllo ✓ 日本語",
	"nbsp edge":          "\u00a0x\u00a0",
	"json envelope":      `{"token":"abc # def","n":1,"s":"it's"}`,
	"json multiline":     "{\n  \"a\": \"b\"\n}\n",
	"json trailing hash": `{"a":"b"} # c`,
	"export word":        "export FOO=bar",
	"hash then quote":    `x # "y"`,
	"dollar then quote":  `$X "y"`,
}

func TestDotenvStoreValuesRoundTrip(t *testing.T) {
	ctx := context.Background()
	for name, v := range roundTripCases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "secrets.env")
			store := newDotenvStore(path)
			require.NoError(t, store.Set(ctx, "BEFORE", []byte("pre $X # y")))
			require.NoError(t, store.Set(ctx, "VAL", []byte(v)))
			require.NoError(t, store.Set(ctx, "X", []byte("expanded")))

			got, err := store.Get(ctx, "VAL")
			require.NoError(t, err)
			assert.Equal(t, v, string(got.Value))

			// A fresh store and the public loader agree.
			m, err := LoadSecrets(path)
			require.NoError(t, err)
			assert.Equal(t, v, m["VAL"])
			assert.Equal(t, "pre $X # y", m["BEFORE"])
			assert.Equal(t, "expanded", m["X"])
			assert.Len(t, m, 3)
		})
	}
}

// Whatever Set accepts must read back exactly; whatever it cannot hold must be
// rejected, never stored corrupted.
func TestDotenvStoreRandomValuesNeverCorrupt(t *testing.T) {
	alphabet := []string{"a", "b", " ", "\t", "#", "\n", "\r", "'", `"`, `\`, "$", "{", "}", "=", "é", "X", "(", ")"}
	rng := rand.New(rand.NewSource(1))
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "secrets.env")
	store := newDotenvStore(path)
	for i := 0; i < 3000; i++ {
		var b strings.Builder
		for n := rng.Intn(8); n >= 0; n-- {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		v := b.String()
		err := store.Set(ctx, "K", []byte(v))
		if err != nil {
			require.ErrorIs(t, err, ErrSecretsUnencodable, "value %q", v)
			assert.NotContains(t, err.Error(), v)
			continue
		}
		got, gerr := store.Get(ctx, "K")
		require.NoError(t, gerr, "value %q", v)
		require.Equal(t, v, string(got.Value))
	}
}

func TestDotenvStoreRejectsInvalidKey(t *testing.T) {
	store := newDotenvStore(filepath.Join(t.TempDir(), "secrets.env"))
	for _, k := range []string{"", "A B", "A=B", "A\nB", "#A", "A-B", "A:B"} {
		require.Error(t, store.Set(context.Background(), k, []byte("v")), "key %q", k)
	}
}

// Files written by older versions (unquoted) or by hand (quoted) keep reading
// exactly as before: the reader is unchanged and these pin its behaviour.
func TestLoadSecretsLegacyFilesUnchanged(t *testing.T) {
	legacy := "# comment\n" +
		"PLAIN=abc123\n" +
		"export EXPORTED=yes\n" +
		"SPACED = padded value  \n" +
		"TRAILING=value # trailing comment\n" +
		"HASHED=abc#def\n" +
		"DQ=\"two words\"\n" +
		"DQESC=\"a\\nb\"\n" +
		"SQ='literal $HOME \\n'\n" +
		"JSON='{\"a\":\"b # c\"}'\n" +
		"MULTI=\"l1\nl2\"\n" +
		"EMPTY=\n" +
		"EXPAND=${PLAIN}-x\n" +
		"\n"
	want := map[string]string{
		"PLAIN":    "abc123",
		"EXPORTED": "yes",
		"SPACED":   "padded value",
		"TRAILING": "value",
		"HASHED":   "abc#def",
		"DQ":       "two words",
		"DQESC":    "a\nb",
		"SQ":       `literal $HOME \n`,
		"JSON":     `{"a":"b # c"}`,
		"MULTI":    "l1\nl2",
		"EMPTY":    "",
		"EXPAND":   "abc123-x",
	}
	for name, content := range map[string]string{"lf": legacy, "crlf": strings.ReplaceAll(legacy, "\n", "\r\n")} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "secrets.env")
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			got, err := LoadSecrets(path)
			require.NoError(t, err)
			ref, rerr := godotenv.Read(path)
			require.NoError(t, rerr)
			assert.Equal(t, ref, got, "must equal the bare reader")
			if name == "lf" {
				assert.Equal(t, want, got)
			}
		})
	}
}

// A rewrite of a legacy file keeps every value it read.
func TestDotenvStoreRewritePreservesLegacyValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	require.NoError(t, os.WriteFile(path, []byte("A=one\nB='x # y'\nC=\"p\\nq\"\n"), 0o600))
	store := newDotenvStore(path)
	require.NoError(t, store.Set(context.Background(), "D", []byte("new # v")))
	got, err := LoadSecrets(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"A": "one", "B": "x # y", "C": "p\nq", "D": "new # v"}, got)
}

func TestEncodeSecretValueUnencodable(t *testing.T) {
	// Pinned gaps of the reader: every candidate style fails these.
	for _, v := range []string{"it's\n\\", " it's\"", "\n\\"} {
		_, err := encodeSecretValue(v)
		assert.ErrorIs(t, err, ErrSecretsUnencodable, "value %q", v)
	}
}

var _ secret.MutableStore = (*dotenvStore)(nil)
