package core

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/joho/godotenv"
)

// ErrSecretsParse marks a secrets.env file that the dotenv reader rejected.
// The error carries the file path and line number only: the reader's own
// message quotes the offending text, which may be secret material, so it is
// never wrapped or surfaced. Match with errors.Is(err, ErrSecretsParse) or
// errors.As(err, *SecretsParseError).
var ErrSecretsParse = errors.New("malformed secrets file")

// SecretsParseError reports a malformed secrets file by location only.
type SecretsParseError struct {
	Path string
	Line int
}

func (e *SecretsParseError) Error() string {
	return fmt.Sprintf("%s: %s at line %d (content withheld)", e.Path, ErrSecretsParse.Error(), e.Line)
}

// Is makes errors.Is(err, ErrSecretsParse) true.
func (e *SecretsParseError) Is(target error) bool { return target == ErrSecretsParse }

// parseSecretsEnv parses dotenv bytes. On failure the raw reader error is
// discarded and a *SecretsParseError locating the first bad statement is
// returned. It is the only place secrets.env bytes reach the dotenv reader.
func parseSecretsEnv(path string, data []byte) (map[string]string, error) {
	m, err := godotenv.UnmarshalBytes(data)
	if err == nil {
		return m, nil
	}
	return nil, &SecretsParseError{Path: path, Line: firstBadLine(data)}
}

// firstBadLine finds the 1-based line where the first statement the reader
// rejects begins, without reading the reader's error text. It consumes lines
// greedily, each statement being the shortest run of lines that parses (a
// quoted value may span lines); the first line from which no run parses is
// the bad statement.
func firstBadLine(data []byte) int {
	src := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(src, []byte("\n"))
	for start := 0; start < len(lines); {
		next := -1
		for end := start + 1; end <= len(lines); end++ {
			chunk := bytes.Join(lines[start:end], []byte("\n"))
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
