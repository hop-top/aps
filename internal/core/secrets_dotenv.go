package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
)

// ErrSecretsUnencodable marks a value the secrets.env format cannot hold with
// the pinned reader. The message never includes the value.
var ErrSecretsUnencodable = errors.New("value cannot be stored in secrets.env")

// validSecretKey reports whether k can be written as a dotenv key and read
// back unchanged: letters, digits, '_' and '.', non-empty.
func validSecretKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if r != '_' && r != '.' && !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

// encodeSecretsEnv renders m as secrets.env content, keys sorted. Every value
// goes through encodeSecretValue.
func encodeSecretsEnv(m map[string]string) (string, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if !validSecretKey(k) {
			return "", fmt.Errorf("secrets.env: invalid key %q", k)
		}
		enc, err := encodeSecretValue(m[k])
		if err != nil {
			return "", fmt.Errorf("secrets.env: key %s: %w", k, err)
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(enc)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// encodeSecretValue is the single encoder for secrets.env values. godotenv
// has gaps (single quotes cannot hold ' or end in \; double quotes cannot end
// in " or \, and strip trailing quote chars; CRLF is rewritten inside any
// value; unquoted values lose edge whitespace, " #" tails and $VAR), so no
// one style covers every value. Candidates are tried in order of plainness
// and each is verified against the reader itself; the first that reads back
// byte for byte wins. ErrSecretsUnencodable if none does.
func encodeSecretValue(v string) (string, error) {
	dq := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "$", `\$`)
	candidates := []string{
		v,
		"'" + v + "'",
		`"` + dq.Replace(v) + `"`,
		strings.ReplaceAll(v, "$", `\$`),
	}
	for _, c := range candidates {
		if readsBack(c, v) {
			return c, nil
		}
	}
	return "", ErrSecretsUnencodable
}

func readsBack(encoded, want string) bool {
	m, err := godotenv.Unmarshal("K=" + encoded + "\nZ=z\n")
	return err == nil && len(m) == 2 && m["K"] == want && m["Z"] == "z"
}
