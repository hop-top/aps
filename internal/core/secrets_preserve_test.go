package core

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/kit/go/storage/secret"
)

// goldenSecretsEnv exercises every statement shape the writer must leave alone.
const goldenSecretsEnv = `# header comment
# GITHUB_TOKEN=...

export A=one
B='single # not comment'
C="dq \"q\" \$x"
D=plain
# COMMENTED=old
E="line1
line2"
F=val # inline note
dup.key=first
G=a=b=c

H: yaml
dup.key=second
`

var goldenMap = map[string]string{
	"A": "one", "B": "single # not comment", "C": `dq "q" $x`, "D": "plain",
	"E": "line1\nline2", "F": "val", "dup.key": "second", "G": "a=b=c", "H": "yaml",
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// layoutVariant renders the golden text in one of the line-ending shapes.
func layoutVariant(name, lf string) string {
	switch name {
	case "crlf":
		return strings.ReplaceAll(lf, "\n", "\r\n")
	case "nofinal":
		return strings.TrimSuffix(lf, "\n")
	}
	return lf
}

func writeSecretsFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "secrets.env")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func readFileString(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	return string(b)
}

func TestDotenvStoreSetDeletePreserveLayout(t *testing.T) {
	ctx := context.Background()
	type op struct {
		name string
		do   func(*dotenvStore) error
		// want is the LF rendering of the expected bytes.
		want    string
		wantMap map[string]string
	}
	mk := func(edit func(string) string, mm func(map[string]string)) (string, map[string]string) {
		m := copyMap(goldenMap)
		mm(m)
		return edit(goldenSecretsEnv), m
	}
	var ops []op
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "D=plain\n", "D=changed\n", 1) },
			func(m map[string]string) { m["D"] = "changed" })
		ops = append(ops, op{"set existing", func(d *dotenvStore) error { return d.Set(ctx, "D", []byte("changed")) }, want, m})
	}
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "export A=one\n", "A=new\n", 1) },
			func(m map[string]string) { m["A"] = "new" })
		ops = append(ops, op{"set exported", func(d *dotenvStore) error { return d.Set(ctx, "A", []byte("new")) }, want, m})
	}
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "E=\"line1\nline2\"\n", "E=x y\n", 1) },
			func(m map[string]string) { m["E"] = "x y" })
		ops = append(ops, op{"set multiline", func(d *dotenvStore) error { return d.Set(ctx, "E", []byte("x y")) }, want, m})
	}
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "dup.key=second\n", "dup.key=third\n", 1) },
			func(m map[string]string) { m["dup.key"] = "third" })
		ops = append(ops, op{"set duplicate", func(d *dotenvStore) error { return d.Set(ctx, "dup.key", []byte("third")) }, want, m})
	}
	{
		want, m := mk(func(s string) string { return s + "NEW=n # v\n" },
			func(m map[string]string) { m["NEW"] = "n # v" })
		// value with " #" is single-quoted by the encoder
		want = strings.TrimSuffix(want, "NEW=n # v\n") + "NEW='n # v'\n"
		ops = append(ops, op{"set new", func(d *dotenvStore) error { return d.Set(ctx, "NEW", []byte("n # v")) }, want, m})
	}
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "F=val # inline note\n", "", 1) },
			func(m map[string]string) { delete(m, "F") })
		ops = append(ops, op{"delete", func(d *dotenvStore) error { return d.Delete(ctx, "F") }, want, m})
	}
	{
		want, m := mk(func(s string) string { return strings.Replace(s, "E=\"line1\nline2\"\n", "", 1) },
			func(m map[string]string) { delete(m, "E") })
		ops = append(ops, op{"delete multiline", func(d *dotenvStore) error { return d.Delete(ctx, "E") }, want, m})
	}
	{
		want, m := mk(func(s string) string {
			return strings.Replace(strings.Replace(s, "dup.key=first\n", "", 1), "dup.key=second\n", "", 1)
		}, func(m map[string]string) { delete(m, "dup.key") })
		ops = append(ops, op{"delete duplicate", func(d *dotenvStore) error { return d.Delete(ctx, "dup.key") }, want, m})
	}

	for _, variant := range []string{"lf", "crlf", "nofinal"} {
		for _, o := range ops {
			t.Run(variant+"/"+o.name, func(t *testing.T) {
				p := writeSecretsFile(t, layoutVariant(variant, goldenSecretsEnv))
				require.NoError(t, o.do(newDotenvStore(p)))
				wantBytes := layoutVariant(variant, o.want)
				if variant == "nofinal" {
					switch o.name {
					case "set new":
						// a newline is added before the appended line; the line itself ends in one
						wantBytes = strings.TrimSuffix(goldenSecretsEnv, "\n") + "\nNEW='n # v'\n"
					case "delete duplicate":
						// the unterminated last line is gone; the line before keeps its newline
						wantBytes = o.want
					default:
						// the unterminated tail stays unterminated
						wantBytes = strings.TrimSuffix(o.want, "\n")
					}
				}
				assert.Equal(t, wantBytes, readFileString(t, p))
				got, err := LoadSecrets(p)
				require.NoError(t, err)
				assert.Equal(t, o.wantMap, got)
			})
		}
	}
}

// CRLF files get a CRLF appended line.
func TestDotenvStoreAppendUsesFileLineEnding(t *testing.T) {
	p := writeSecretsFile(t, "A=1\r\nB=2\r\n")
	require.NoError(t, newDotenvStore(p).Set(context.Background(), "C", []byte("3")))
	assert.Equal(t, "A=1\r\nB=2\r\nC=3\r\n", readFileString(t, p))
}

func TestDotenvStoreMissingOrEmptyFileWritesJustTheLine(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "secrets.env")
	require.NoError(t, newDotenvStore(p).Set(ctx, "A", []byte("1")))
	assert.Equal(t, "A=1\n", readFileString(t, p))

	q := writeSecretsFile(t, "")
	require.NoError(t, newDotenvStore(q).Set(ctx, "A", []byte("1")))
	assert.Equal(t, "A=1\n", readFileString(t, q))
}

// The header written at profile creation survives the first Set.
func TestProfileTemplateHeaderSurvivesFirstSet(t *testing.T) {
	tmpl := "# Add your secrets here. Format: KEY=VALUE\n# GITHUB_TOKEN=...\n"
	p := writeSecretsFile(t, tmpl)
	require.NoError(t, newDotenvStore(p).Set(context.Background(), "GITHUB_TOKEN", []byte("sentinel-1")))
	assert.Equal(t, tmpl+"GITHUB_TOKEN=sentinel-1\n", readFileString(t, p))
}

func TestDotenvStoreSetSameValueIsByteIdentical(t *testing.T) {
	for _, variant := range []string{"lf", "crlf", "nofinal"} {
		t.Run(variant, func(t *testing.T) {
			in := layoutVariant(variant, goldenSecretsEnv)
			p := writeSecretsFile(t, in)
			store := newDotenvStore(p)
			for k, v := range goldenMap {
				require.NoError(t, store.Set(context.Background(), k, []byte(v)))
			}
			assert.Equal(t, in, readFileString(t, p))
		})
	}
}

func TestDotenvStoreDeleteMissingKey(t *testing.T) {
	p := writeSecretsFile(t, goldenSecretsEnv)
	err := newDotenvStore(p).Delete(context.Background(), "NOPE")
	assert.True(t, errors.Is(err, secret.ErrNotFound))
	assert.Equal(t, goldenSecretsEnv, readFileString(t, p))
}

// An unparseable file fails Set/Delete with the location-only error and is
// not touched.
func TestDotenvStoreUnparseableFileUntouched(t *testing.T) {
	const sentinel = "sentinel-hunter2"
	bad := "A=1\nB=\"" + sentinel + "\n"
	p := writeSecretsFile(t, bad)
	store := newDotenvStore(p)
	for name, err := range map[string]error{
		"set":    store.Set(context.Background(), "C", []byte("x")),
		"delete": store.Delete(context.Background(), "A"),
	} {
		var pe *SecretsParseError
		require.ErrorAs(t, err, &pe, name)
		assert.Equal(t, 2, pe.Line, name)
		assert.NotContains(t, err.Error(), sentinel, name)
	}
	assert.Equal(t, bad, readFileString(t, p))
}

// Two keys on one line cannot be attributed: they are left alone.
func TestDotenvStoreUnattributableStatement(t *testing.T) {
	ctx := context.Background()
	in := "A=1\nB=\"x\" C=2\nD=4\n"
	p := writeSecretsFile(t, in)
	store := newDotenvStore(p)

	// Delete cannot remove a key it cannot isolate; nothing is written.
	require.Error(t, store.Delete(ctx, "C"))
	assert.Equal(t, in, readFileString(t, p))

	// Set appends: the later occurrence is the one the reader keeps.
	require.NoError(t, store.Set(ctx, "C", []byte("9")))
	assert.Equal(t, in+"C=9\n", readFileString(t, p))
	got, err := LoadSecrets(p)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"A": "1", "B": "x", "C": "9", "D": "4"}, got)
}

// A key that only ever expands from an earlier one keeps its line when the
// source changes; the value written for a key reads back exactly even when
// it mentions a name defined earlier in the same file.
func TestDotenvStoreExpansionInContext(t *testing.T) {
	ctx := context.Background()
	t.Run("dependent line untouched", func(t *testing.T) {
		in := "A=hello\nB=${A}\n"
		p := writeSecretsFile(t, in)
		require.NoError(t, newDotenvStore(p).Set(ctx, "A", []byte("bye")))
		assert.Equal(t, "A=bye\nB=${A}\n", readFileString(t, p))
		got, err := LoadSecrets(p)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "bye", "B": "bye"}, got)
	})
	for _, v := range []string{"$A", "${A}", "x$Ay", "x${A}y", `\$A`, `\\$A`, "$$A", `"$A"`, "$A$A", "$(A)", "pre-$A-$B"} {
		for _, where := range []string{"append", "replace"} {
			t.Run(where+"/"+v, func(t *testing.T) {
				in := "A=hello\nB=old\nC=tail\n"
				if where == "append" {
					in = "A=hello\nC=tail\n"
				}
				p := writeSecretsFile(t, in)
				require.NoError(t, newDotenvStore(p).Set(ctx, "B", []byte(v)))
				got, err := LoadSecrets(p)
				require.NoError(t, err)
				assert.Equal(t, v, got["B"])
				assert.Equal(t, "hello", got["A"])
				assert.Equal(t, "tail", got["C"])
				assert.True(t, strings.HasPrefix(readFileString(t, p), "A=hello\n"))
			})
		}
	}
}

// The read-back safety net refuses edits that do not read back.
func TestVerifySecretsEditRejectsMismatch(t *testing.T) {
	old := map[string]string{"A": "1", "B": "2"}
	want := "9"
	ok := []byte("A=9\nB=2\n")
	assert.NoError(t, verifySecretsEdit("p", old, ok, "A", &want, nil))

	for name, content := range map[string]string{
		"wrong value":  "A=8\nB=2\n",
		"missing key":  "B=2\n",
		"other moved":  "A=9\nB=3\n",
		"extra key":    "A=9\nB=2\nC=1\n",
		"unparseable":  "A=9\nB=\"\n",
		"wrong b64":    "A=aps-b64:!!!\nB=2\n",
		"delete stays": "A=1\nB=2\n",
	} {
		w := want
		wp := &w
		if name == "delete stays" {
			wp = nil
		}
		err := verifySecretsEdit("p", old, []byte(content), "A", wp, nil)
		require.Error(t, err, name)
		assert.NotContains(t, err.Error(), "aps-b64:!!!", name)
	}
	// dependents may move
	assert.NoError(t, verifySecretsEdit("p", old, []byte("A=9\nB=3\n"), "A", &want, map[string]bool{"B": true}))
}

// ---- property test ----

type propStmt struct {
	text string // exact bytes, including terminator
	key  string // "" for comment/blank
	val  string
}

func propValue(r *rand.Rand) string {
	pool := []string{"plain", "a b", "x # y", "it's", `say "hi"`, "l1\nl2", "a=b", "tail\\", "  pad  ", "", "é✓", "k:v"}
	return pool[r.Intn(len(pool))]
}

// propRender writes (key, val) in a randomly chosen style the reader reads back.
func propRender(r *rand.Rand, key, val string) string {
	enc := encodeSecretValue(val)
	switch r.Intn(5) {
	case 0:
		return "export " + key + "=" + enc
	case 1:
		return key + " = " + enc
	case 2:
		if !strings.ContainsAny(val, "\n") && readsBack("'"+val+"'", val) {
			return key + "='" + val + "'"
		}
	case 3:
		if !strings.ContainsAny(val, "\n\"\\$") && readsBack(`"`+val+`"`, val) {
			return key + `="` + val + `"`
		}
	}
	return key + "=" + enc
}

func propFile(r *rand.Rand, keys []string, eol string) []propStmt {
	var out []propStmt
	n := r.Intn(9)
	for i := 0; i < n; i++ {
		switch r.Intn(5) {
		case 0:
			out = append(out, propStmt{text: "# comment " + fmt.Sprint(i) + eol})
		case 1:
			out = append(out, propStmt{text: eol})
		default:
			k := keys[r.Intn(len(keys))]
			v := propValue(r)
			t := propRender(r, k, v)
			if r.Intn(4) == 0 && v != "" && !strings.ContainsAny(t, "'\"") {
				t += " # inline"
			}
			t = strings.ReplaceAll(t, "\n", eol)
			out = append(out, propStmt{text: t + eol, key: k, val: v})
		}
	}
	return out
}

func propModel(stmts []propStmt) map[string]string {
	m := map[string]string{}
	for _, s := range stmts {
		if s.key != "" {
			m[s.key] = s.val
		}
	}
	return m
}

func TestDotenvStorePropertyUntouchedStatementsIdentical(t *testing.T) {
	ctx := context.Background()
	keys := []string{"A", "B", "C.d", "E_1", "F"}
	for seed := int64(1); seed <= 150; seed++ {
		r := rand.New(rand.NewSource(seed))
		eol := "\n"
		if seed%3 == 0 {
			eol = "\r\n"
		}
		stmts := propFile(r, keys, eol)
		join := func() string {
			var b strings.Builder
			for _, s := range stmts {
				b.WriteString(s.text)
			}
			return b.String()
		}
		p := writeSecretsFile(t, join())
		store := newDotenvStore(p)
		for step := 0; step < 8; step++ {
			k := keys[r.Intn(len(keys))]
			model := propModel(stmts)
			desc := fmt.Sprintf("seed=%d step=%d key=%s", seed, step, k)
			var err error
			stmts, err = propStep(ctx, r, store, stmts, model, k, eol)
			require.NoError(t, err, desc)
			assert.Equal(t, join(), readFileString(t, p), desc)
			got, err := LoadSecrets(p)
			require.NoError(t, err, desc)
			if len(propModel(stmts)) == 0 {
				assert.Empty(t, got, desc)
			} else {
				assert.Equal(t, propModel(stmts), got, desc)
			}
		}
	}
}

// eolOf is the line terminator of the last line of text.
func eolOf(text, def string) string {
	if strings.HasSuffix(text, "\r\n") {
		return "\r\n"
	}
	if strings.HasSuffix(text, "\n") {
		return "\n"
	}
	return ""
}

// lastEOL is the style of the last terminated line of the file, or def.
func lastEOL(stmts []propStmt, def string) string {
	for i := len(stmts) - 1; i >= 0; i-- {
		if e := eolOf(stmts[i].text, def); e != "" {
			return e
		}
	}
	return "\n"
}

// The pinned reader keeps the last assignment of a repeated key; Set updates
// that occurrence and Delete removes all of them.
func TestSecretsDuplicateKeyLastWins(t *testing.T) {
	got, err := parseSecretsEnv("p", []byte("A=1\nB=x\nA=2\n"))
	require.NoError(t, err)
	assert.Equal(t, "2", got["A"])

	ctx := context.Background()
	p := writeSecretsFile(t, "A=1\nB=x\nA=2\n")
	require.NoError(t, newDotenvStore(p).Set(ctx, "A", []byte("3")))
	assert.Equal(t, "A=1\nB=x\nA=3\n", readFileString(t, p))
	require.NoError(t, newDotenvStore(p).Delete(ctx, "A"))
	assert.Equal(t, "B=x\n", readFileString(t, p))
}

// Whatever encodes alone also reads back after earlier statements that define
// the names it mentions: expansion cannot make an isolated candidate lie.
func TestPlainCandidatesReadBackInContext(t *testing.T) {
	prefix := "A=hello\nB_1=world\nexport C=\"c d\"\n"
	alpha := []string{"$", "A", "B_1", "{", "}", "(", ")", `\`, `"`, "'", " ", "#", "x"}
	var walk func(p string, n int)
	walk = func(v string, n int) {
		for _, c := range secretValueCandidates(v) {
			m, err := parseSecretsEnv("p", []byte(prefix+"K="+c+"\n"))
			require.NoError(t, err, "%q via %q", v, c)
			require.Equal(t, v, m["K"], "%q via %q", v, c)
		}
		if n == 0 {
			return
		}
		for _, a := range alpha {
			walk(v+a, n-1)
		}
	}
	walk("", 4)
}

// propStep applies one random Set or Delete to the store and returns the
// model statements it must leave behind.
func propStep(ctx context.Context, r *rand.Rand, store *dotenvStore, stmts []propStmt, model map[string]string, k, eol string) ([]propStmt, error) {
	if r.Intn(3) == 0 {
		err := store.Delete(ctx, k)
		if _, present := model[k]; !present {
			if !errors.Is(err, secret.ErrNotFound) {
				return nil, fmt.Errorf("delete of absent key: %w", err)
			}
			return stmts, nil
		}
		kept := stmts[:0:0]
		for _, s := range stmts {
			if s.key != k {
				kept = append(kept, s)
			}
		}
		return kept, err
	}
	v := propValue(r)
	if err := store.Set(ctx, k, []byte(v)); err != nil {
		return nil, err
	}
	cur, present := model[k]
	switch {
	case present && cur == v:
		return stmts, nil
	case present:
		last := -1
		for i, s := range stmts {
			if s.key == k {
				last = i
			}
		}
		stmts[last] = propStmt{text: k + "=" + encodeSecretValue(v) + eolOf(stmts[last].text, eol), key: k, val: v}
		return stmts, nil
	}
	if len(stmts) > 0 && !strings.HasSuffix(stmts[len(stmts)-1].text, "\n") {
		stmts[len(stmts)-1].text += eol
	}
	return append(stmts, propStmt{text: k + "=" + encodeSecretValue(v) + lastEOL(stmts, eol), key: k, val: v}), nil
}
