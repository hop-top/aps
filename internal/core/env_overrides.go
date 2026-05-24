package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// BuildOverrideEnv resolves --env-file paths and --env entries into a single
// KEY=VALUE slice in precedence order: file entries first (in flag order),
// then inline entries (in flag order). Callers append the result to the
// profile-derived env; later entries win for duplicate keys.
//
// Validation is fatal-by-error:
//   - missing env-file path → error
//   - malformed env-file line → error citing the file + line number
//   - --env value without "=" → error
//   - --env value with empty key → error
func BuildOverrideEnv(envFiles []string, envInline []string) ([]string, error) {
	out := make([]string, 0, len(envInline))
	for _, path := range envFiles {
		entries, err := parseEnvFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, entries...)
	}
	for _, kv := range envInline {
		k, v, err := parseInlineEnv(kv)
		if err != nil {
			return nil, err
		}
		out = append(out, k+"="+v)
	}
	return out, nil
}

func parseInlineEnv(kv string) (key, value string, err error) {
	idx := strings.IndexByte(kv, '=')
	if idx < 0 {
		return "", "", fmt.Errorf("invalid --env %q: expected KEY=VALUE", kv)
	}
	k := kv[:idx]
	v := kv[idx+1:]
	if k == "" {
		return "", "", fmt.Errorf("invalid --env %q: empty key", kv)
	}
	if !isValidEnvKey(k) {
		return "", "", fmt.Errorf("invalid --env %q: key must match [A-Za-z_][A-Za-z0-9_]*", kv)
	}
	return k, v, nil
}

func parseEnvFile(path string) ([]string, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from CLI flag, owner-controlled
	if err != nil {
		return nil, fmt.Errorf("--env-file %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var out []string
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			return nil, fmt.Errorf("--env-file %q: malformed line %d: expected KEY=VALUE, got %q", path, lineNo, raw)
		}
		key := strings.TrimSpace(line[:idx])
		val := line[idx+1:]
		if !isValidEnvKey(key) {
			return nil, fmt.Errorf("--env-file %q: malformed line %d: key %q must match [A-Za-z_][A-Za-z0-9_]*", path, lineNo, key)
		}
		val = unquoteEnvValue(val)
		out = append(out, key+"="+val)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("--env-file %q: %w", path, err)
	}
	return out, nil
}

func isValidEnvKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// unquoteEnvValue strips surrounding matching quotes from a dotenv value and
// trims trailing inline comments on unquoted values. Mirrors the common
// dotenv conventions without expanding shell escapes.
func unquoteEnvValue(v string) string {
	if len(v) >= 2 {
		first := v[0]
		last := v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	if idx := strings.Index(v, " #"); idx >= 0 {
		v = v[:idx]
	}
	return strings.TrimRight(v, " \t")
}
