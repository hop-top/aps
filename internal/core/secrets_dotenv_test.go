package core

import (
	"context"
	"encoding/base64"
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

// Every value, including NUL bytes and invalid UTF-8, reads back exactly.
func TestDotenvStoreRandomValuesAlwaysRoundTrip(t *testing.T) {
	alphabet := []string{"a", "b", " ", "\t", "#", "\n", "\r", "\r\n", "'", `"`, `\`, "$", "{", "}", "=", "\u00e9", "X", "(", ")",
		"\x00", "\xff", "aps-b64:", "base64:", "\u00a0", "\\n", "${A}", `\"`, `\'`}
	rng := rand.New(rand.NewSource(1))
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "secrets.env")
	store := newDotenvStore(path)
	for i := 0; i < 5000; i++ {
		var b strings.Builder
		for n := rng.Intn(9); n >= 0; n-- {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		v := b.String()
		require.NoError(t, store.Set(ctx, "K", []byte(v)), "value %q", v)
		got, err := store.Get(ctx, "K")
		require.NoError(t, err, "value %q", v)
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

// Values the reader cannot hold in any plain style, found by enumeration:
// each ends in '"' or '\\' (double quotes cannot close after them), cannot be
// carried raw, and cannot be single-quoted.
func TestPlainEncodingGapClass(t *testing.T) {
	alpha := []string{"a", " ", "#", "\n", "\r", "'", `"`, `\`, "$"}
	var walk func(p string, n int)
	gaps := 0
	walk = func(p string, n int) {
		_, ok := encodePlain(p)
		if !ok {
			gaps++
			assert.True(t, strings.HasSuffix(p, `"`) || strings.HasSuffix(p, `\`), "gap value %q must end in a quote or backslash", p)
		}
		if n == 0 {
			return
		}
		for _, a := range alpha {
			walk(p+a, n-1)
		}
	}
	walk("", 5)
	assert.Positive(t, gaps)
}

// Named residual cases from real use.
func TestResidualValuesRoundTrip(t *testing.T) {
	ctx := context.Background()
	big := strings.Repeat("x # \"'\n\\", 150000)[:1<<20]
	cases := map[string]string{
		"json string with apostrophe": `"it's"`,
		"apostrophe then backslash":   "it's\n\\",
		"space apostrophe dquote":     ` it's"`,
		"newline backslash":           "\n\\",
		"dquote end with squote":      `x # it's "y"`,
		"dquote end no squote":        "x\n\"",
		"backslash end spaces":        " x\\",
		"backslash end hash":          "a #b\\",
		"crlf trailing backslash":     "a\r\nb\\",
		"crlf squote dquote end":      "it's\r\n\"",
		"marker literal":              "aps-b64:aGVsbG8=",
		"marker alone":                "aps-b64:",
		"marker garbage":              "aps-b64:!!!not base64",
		"nul":                         "a\x00b",
		"invalid utf8":                "a\xff\xfeb",
		"1MiB":                        big,
		"base64 padding 1":            "\xff\xfe",
		"base64 padding 2":            "\xff",
	}
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "secrets.env")
			store := newDotenvStore(path)
			require.NoError(t, store.Set(ctx, "A_BEFORE", []byte("x")))
			require.NoError(t, store.Set(ctx, "K", []byte(v)))
			require.NoError(t, store.Set(ctx, "Z_AFTER", []byte("y")))
			got, err := LoadSecrets(path)
			require.NoError(t, err)
			require.Equal(t, v, got["K"])
			assert.Equal(t, "x", got["A_BEFORE"])
			assert.Equal(t, "y", got["Z_AFTER"])
		})
	}
}

// Ordinary values keep today's on-disk form; the marker is the last resort.
func TestEncodingOrderUnchanged(t *testing.T) {
	cases := map[string]string{
		"abc":        "abc",
		"":           "",
		"a #b":       "'a #b'",
		"it's":       "it's",
		"l1\nl2":     "'l1\nl2'",
		"a\r\nb":     `"a\r\nb"`,
		"$X":         "'$X'",
		`"it's"`:     "aps-b64:" + base64.StdEncoding.EncodeToString([]byte(`"it's"`)),
		"base64:abc": "base64:abc",
		"aps-b64:x":  "aps-b64:" + base64.StdEncoding.EncodeToString([]byte("aps-b64:x")),
	}
	for v, want := range cases {
		got := encodeSecretValue(v)
		assert.Equal(t, want, got, "value %q", v)
	}
}

// Files written by other tools: marker look-alikes read literally.
func TestMarkerLookalikesReadLiterally(t *testing.T) {
	content := "A=base64:abc\nB=base64:SGVsbG8=\nC=aps-b64\nD=aps-b64x:SGVsbG8=\nE=APS-B64:SGVsbG8=\nF=xaps-b64:SGVsbG8=\n"
	path := filepath.Join(t.TempDir(), "secrets.env")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	got, err := LoadSecrets(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"A": "base64:abc", "B": "base64:SGVsbG8=", "C": "aps-b64",
		"D": "aps-b64x:SGVsbG8=", "E": "APS-B64:SGVsbG8=", "F": "xaps-b64:SGVsbG8=",
	}, got)
}

func TestMarkerValueDecodes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	require.NoError(t, os.WriteFile(path, []byte("K=aps-b64:SGVsbG8=\n"), 0o600))
	got, err := LoadSecrets(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello", got["K"])
}

var _ secret.MutableStore = (*dotenvStore)(nil)
