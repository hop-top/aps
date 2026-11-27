package core

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// ErrSecretsParse marks a secrets.env file that the dotenv reader rejected.
// The error carries the file path and line number only: the reader's own
// message quotes the offending text, which may be secret material, so it is
// never wrapped or surfaced. Match with errors.Is(err, ErrSecretsParse) or
// errors.As(err, *SecretsParseError).
var ErrSecretsParse = errors.New("malformed secrets file")

// SecretsParseError reports a malformed secrets file by location only.
// Line locates a syntax error; Key names the entry whose base64 payload is
// corrupt (keys are never secret; values are never reported).
type SecretsParseError struct {
	Path string
	Line int
	Key  string
}

func (e *SecretsParseError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("%s: %s: key %s has an invalid %s payload (content withheld)",
			e.Path, ErrSecretsParse.Error(), e.Key, secretsB64Marker)
	}
	return fmt.Sprintf("%s: %s at line %d (content withheld)", e.Path, ErrSecretsParse.Error(), e.Line)
}

// firstBadLineBudget caps the bytes firstBadLine may re-parse while locating
// a syntax error. Far above what an ordinary secrets file needs; it only
// bounds the pathological case.
const firstBadLineBudget = 16 << 20

// Is makes errors.Is(err, ErrSecretsParse) true.
func (e *SecretsParseError) Is(target error) bool { return target == ErrSecretsParse }

// parseSecretsEnv parses dotenv bytes. On failure the raw reader error is
// discarded and a *SecretsParseError locating the first bad statement is
// returned. It is the only place secrets.env bytes reach the dotenv reader.
func parseSecretsEnv(path string, data []byte) (map[string]string, error) {
	m, err := godotenv.UnmarshalBytes(data)
	if err != nil {
		return nil, &SecretsParseError{Path: path, Line: firstBadLine(data)}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		payload, ok := strings.CutPrefix(v, secretsB64Marker)
		if !ok {
			continue
		}
		raw, derr := base64.StdEncoding.DecodeString(payload)
		if derr != nil {
			return nil, &SecretsParseError{Path: path, Key: k}
		}
		m[k] = string(raw)
	}
	return m, nil
}

// firstBadLine finds the 1-based line where the first statement the reader
// rejects begins, without reading the reader's error text. It consumes lines
// greedily, each statement being the shortest run of lines that parses (a
// quoted value may span lines); the first line from which no run parses is
// the bad statement.
//
// Re-parsing growing runs is quadratic in the worst case (a bad statement
// followed by many lines), so the scan stops after firstBadLineBudget bytes
// of parse work and reports the statement it was examining. On a file that
// large the line is then a lower bound, which still beats stalling the
// command that only wants to report the error.
func firstBadLine(data []byte) int {
	src := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(src, []byte("\n"))
	work := 0
	for start := 0; start < len(lines); {
		next := -1
		for end := start + 1; end <= len(lines); end++ {
			chunk := bytes.Join(lines[start:end], []byte("\n"))
			if work += len(chunk) + 1; work > firstBadLineBudget {
				return start + 1
			}
			if _, err := godotenv.UnmarshalBytes(chunk); err == nil {
				next = end
				break
			}
		}
		if next < 0 {
			return start + 1
		}
		start = next
	}
	return 1
}
