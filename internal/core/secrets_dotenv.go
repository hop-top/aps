package core

import (
	"encoding/base64"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
)

// secretsB64Marker prefixes a value stored as base64. It is aps-specific on
// purpose: "base64:" is a literal value in common dotenv files (APP_KEY=base64:...)
// and must keep reading literally.
const secretsB64Marker = "aps-b64:" // #nosec G101 -- format marker, not a credential

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

// encodeSecretValue is the single encoder for secrets.env values. godotenv
// has gaps (single quotes cannot hold ' or end in \; double quotes cannot end
// in " or \, and strip trailing quote chars; CRLF is rewritten inside any
// value; unquoted values lose edge whitespace, " #" tails and $VAR), so no
// one style covers every value. Plain candidates are tried in order and each
// is verified against the reader itself; the first that reads back byte for
// byte wins. A value no plain style carries, and any value that itself starts
// with the marker, is stored as marker + standard base64, which is decoded by
// parseSecretsEnv.
func encodeSecretValue(v string) string {
	if !strings.HasPrefix(v, secretsB64Marker) {
		if c, ok := encodePlain(v); ok {
			return c
		}
	}
	return secretsB64Marker + base64.StdEncoding.EncodeToString([]byte(v))
}

// encodePlain returns the first plain (non-base64) encoding the reader reads
// back exactly.
func encodePlain(v string) (string, bool) {
	for _, c := range plainCandidates(v) {
		if readsBack(c, v) {
			return c, true
		}
	}
	return "", false
}

// plainCandidates lists the plain encodings of v in preference order.
func plainCandidates(v string) []string {
	dq := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "$", `\$`)
	return []string{
		v,
		"'" + v + "'",
		`"` + dq.Replace(v) + `"`,
		strings.ReplaceAll(v, "$", `\$`),
	}
}

// secretValueCandidates lists, in preference order, the encodings of v that
// read back alone: the plain ones that do, then the base64 form. The first is
// what encodeSecretValue returns; the editor verifies them in the context of
// the real file and takes the first that still reads back there.
func secretValueCandidates(v string) []string {
	var out []string
	if !strings.HasPrefix(v, secretsB64Marker) {
		for _, c := range plainCandidates(v) {
			if readsBack(c, v) {
				out = append(out, c)
			}
		}
	}
	return append(out, secretsB64Marker+base64.StdEncoding.EncodeToString([]byte(v)))
}

func readsBack(encoded, want string) bool {
	m, err := godotenv.Unmarshal("K=" + encoded + "\nZ=z\n")
	return err == nil && len(m) == 2 && m["K"] == want && m["Z"] == "z"
}
