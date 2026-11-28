package core

import (
	"bytes"
	"fmt"

	"hop.top/kit/go/storage/secret"
)

// editSecretsEnv returns data with key set to *val, or removed when val is
// nil, changing only the statement(s) that hold key:
//
//   - set of a key already in the file replaces the statement the reader takes
//     the value from (the last one) with KEY=<encoded>, keeping that
//     statement's line terminator; a set that would not change the value
//     leaves the file as is (changed is false);
//   - set of a new key appends one line, in the file's line-ending style;
//   - delete removes every statement holding key (all of its lines).
//
// Everything else, including comments, order, quoting and statements that
// expand other keys, stays byte for byte. A statement holding several keys
// cannot be attributed to one: delete refuses it, set appends instead (the
// later assignment is the one the reader keeps). A file the reader rejects
// fails with its location-only *SecretsParseError. Before returning, the
// result is parsed again in memory and must hold the intended value for key
// and the old values for every other key, else the edit is refused.
func editSecretsEnv(path string, data []byte, key string, val *string) (out []byte, changed bool, err error) {
	old, err := parseSecretsEnv(path, data)
	if err != nil {
		return nil, false, err
	}
	stmts, bad := segmentSecretsEnv(data)
	if bad > 0 {
		// The file parsed above, so it is not malformed: the segmenter gave
		// up. Say so without claiming a syntax error.
		return nil, false, fmt.Errorf("secrets.env: %s: could not edit in place near line %d; edit the file by hand", path, bad)
	}
	cur, present := old[key]

	if val == nil {
		if !present {
			return nil, false, secret.ErrNotFound
		}
		var b bytes.Buffer
		pos := 0
		for _, s := range stmts {
			if !stmtHasKey(s, key) {
				continue
			}
			if len(s.Keys) > 1 {
				return nil, false, fmt.Errorf("secrets.env: key %s shares a line with other keys; edit the file by hand", key)
			}
			b.Write(data[pos:s.Start])
			pos = s.End
		}
		b.Write(data[pos:])
		out = b.Bytes()
		if err := verifySecretsEdit(path, old, out, key, nil, expandingKeys(data, stmts, key)); err != nil {
			return nil, false, err
		}
		return out, true, nil
	}

	if !validSecretKey(key) {
		return nil, false, fmt.Errorf("secrets.env: invalid key %q", key)
	}
	if present && cur == *val {
		return data, false, nil
	}
	target := -1
	for i, s := range stmts {
		if stmtHasKey(s, key) {
			target = i
		}
	}
	skip := expandingKeys(data, stmts, key)
	for _, enc := range secretValueCandidates(*val) {
		// Each candidate is checked in the context of the real file: a value
		// that reads back alone must still do so after the statements before it.
		for _, cand := range editedContents(data, stmts, target, key, enc) {
			if verifySecretsEdit(path, old, cand, key, val, skip) == nil {
				return cand, true, nil
			}
		}
	}
	return nil, false, fmt.Errorf("secrets.env: key %s did not read back after the edit; file left unchanged", key)
}

// stmtPrefix is the leading indentation plus, when present, the export
// keyword and the whitespace after it (the reader requires at least one
// space or tab after export), so a rewritten line stays sourceable.
func stmtPrefix(text []byte) string {
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	rest := text[i:]
	if bytes.HasPrefix(rest, []byte("export")) {
		j := len("export")
		k := j
		for k < len(rest) && (rest[k] == ' ' || rest[k] == '\t') {
			k++
		}
		if k > j {
			return string(text[:i+k])
		}
	}
	return string(text[:i])
}

// editedContents lists the file contents that set key to the encoded value
// enc, most faithful first: the statement at index target rewritten with its
// indentation and export keyword, then without them; or, with no single-key
// statement to rewrite, the file with a line appended.
func editedContents(data []byte, stmts []secretsStmt, target int, key, enc string) [][]byte {
	if target >= 0 && len(stmts[target].Keys) == 1 {
		s := stmts[target]
		text := data[s.Start:s.End]
		eol := lineEnding(text)
		var out [][]byte
		for _, pre := range []string{stmtPrefix(text), ""} {
			c := append([]byte{}, data[:s.Start]...)
			c = append(c, pre+key+"="+enc+eol...)
			c = append(c, data[s.End:]...)
			out = append(out, c)
			if pre == "" {
				break
			}
		}
		return out
	}
	eol := fileLineEnding(data)
	c := append([]byte{}, data...)
	if len(c) > 0 && c[len(c)-1] != '\n' {
		c = append(c, eol...)
	}
	return [][]byte{append(c, key+"="+enc+eol...)}
}

func stmtHasKey(s secretsStmt, key string) bool {
	for _, k := range s.Keys {
		if k == key {
			return true
		}
	}
	return false
}

// lineEnding is the terminator of the last line of text: CRLF, LF, or "" for
// an unterminated final line.
func lineEnding(text []byte) string {
	switch {
	case bytes.HasSuffix(text, []byte("\r\n")):
		return "\r\n"
	case bytes.HasSuffix(text, []byte("\n")):
		return "\n"
	}
	return ""
}

// fileLineEnding is the style of the last terminated line, LF for a file
// with none.
func fileLineEnding(data []byte) string {
	i := bytes.LastIndexByte(data, '\n')
	if i > 0 && data[i-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}

// expandingKeys names the keys, other than key, whose statements may expand
// other keys ($NAME outside single quotes). Their lines stay as written, so
// their values may follow a change to the key they reference.
func expandingKeys(data []byte, stmts []secretsStmt, key string) map[string]bool {
	skip := map[string]bool{}
	for _, s := range stmts {
		if len(s.Keys) == 0 || stmtHasKey(s, key) {
			continue
		}
		text := data[s.Start:s.End]
		if !bytes.Contains(text, []byte("$")) {
			continue
		}
		if len(s.Keys) == 1 {
			// KEY='...$...' is literal.
			if i := bytes.IndexAny(text, "=:"); i >= 0 {
				rest := bytes.TrimLeft(text[i+1:], " \t")
				if bytes.HasPrefix(rest, []byte("'")) {
					continue
				}
			}
		}
		for _, k := range s.Keys {
			skip[k] = true
		}
	}
	return skip
}

// verifySecretsEdit is the safety net of every edit: newData must parse and
// hold want for key (or no key when want is nil) and, for every other key,
// the value it had in old. Keys in skip (statements that expand other keys)
// may differ in value. Errors name keys only, never values.
func verifySecretsEdit(path string, old map[string]string, newData []byte, key string, want *string, skip map[string]bool) error {
	got, err := parseSecretsEnv(path, newData)
	if err != nil {
		return err
	}
	if want == nil {
		if _, ok := got[key]; ok {
			return fmt.Errorf("secrets.env: key %s still present after the edit", key)
		}
	} else if v, ok := got[key]; !ok || v != *want {
		return fmt.Errorf("secrets.env: key %s did not read back after the edit", key)
	}
	for k, v := range old {
		if k == key || skip[k] {
			continue
		}
		if g, ok := got[k]; !ok || g != v {
			return fmt.Errorf("secrets.env: key %s changed by the edit", k)
		}
	}
	for k := range got {
		if _, ok := old[k]; !ok && k != key {
			return fmt.Errorf("secrets.env: key %s appeared in the edit", k)
		}
	}
	return nil
}
