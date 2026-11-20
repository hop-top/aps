package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// Structural-field protection for persisted JSONL.
//
// The kit PII pack carries digit-run rules (credit-card, phone, ssn,
// passport, ...) that have no notion of token shape: a random UUID
// whose tail holds 13+ digits across hyphens ("...-9131-824401265021")
// matches credit-card and is rewritten to "...-<credit-card>". For log
// lines that is noise; for a persisted record the damage is permanent:
// an event ID, resource key, or timestamp that is stored mangled can
// never be looked up again.
//
// Identifier-valued fields are therefore exempted from PII rules at
// persistence time. They are NOT exempted from credential rules: a
// value in an ID field that the redactor recognises as a secret is
// still redacted.

// piiRuleIDs are the rule ids contributed by the Presidio PII pack.
// A match from one of these on an identifier-valued field is a false
// positive by construction. Gitleaks credential rules are deliberately
// absent. The "-pii" variants exist because kit renames Presidio rules
// that collide with a gitleaks id.
var piiRuleIDs = map[string]bool{
	"email": true, "phone-e164": true, "phone-us": true,
	"us-ssn": true, "us-itin": true, "credit-card": true,
	"iban": true, "ipv4": true, "ipv6": true,
	"us-passport": true, "us-driver-license": true,
}

func isPIIRule(id string) bool {
	return piiRuleIDs[strings.TrimSuffix(id, "-pii")]
}

// isStructuralKey reports whether a JSON object key names a field whose
// value is an identifier, reference, or timestamp rather than free
// text: "id", "*_id", "resource", "caused_by", "signature", "version",
// "event_type", "timestamp", "*_at", "*_hmac" (keyed ID hashes).
func isStructuralKey(key string) bool {
	k := strings.ToLower(key)
	switch k {
	case "id", "resource", "caused_by", "signature", "version",
		"event_type", "timestamp":
		return true
	}
	return strings.HasSuffix(k, "_id") || strings.HasSuffix(k, "_at") ||
		strings.HasSuffix(k, "_hmac")
}

// structuralValueSafe reports whether s may be stored verbatim: any
// redaction match inside it comes from the PII pack. A match from a
// credential rule means the "identifier" is really a secret, so it
// goes through normal redaction.
func structuralValueSafe(s string) bool {
	for _, m := range Redactor().Scan(s) {
		if !isPIIRule(m.RuleID) {
			return false
		}
	}
	return true
}

type valueSpan struct{ start, end int }

// structuralSpans returns the byte spans (quotes included) of string
// values held by structural keys in the JSON document b. Spans are in
// document order and never overlap. ok is false when b is not a valid
// JSON document.
func structuralSpans(b []byte, p JSONLPolicy) (spans []valueSpan, ok bool) {
	type frame struct{ obj, expectKey bool }
	var (
		stack      []frame
		structural bool
	)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	for {
		before := int(dec.InputOffset())
		tok, err := dec.Token()
		if err == io.EOF {
			return spans, true
		}
		if err != nil {
			return nil, false
		}
		if d, isDelim := tok.(json.Delim); isDelim {
			switch d {
			case '{':
				stack = append(stack, frame{obj: true, expectKey: true})
			case '[':
				stack = append(stack, frame{})
			default:
				stack = stack[:len(stack)-1]
				if n := len(stack); n > 0 && stack[n-1].obj {
					stack[n-1].expectKey = true
				}
			}
			continue
		}
		n := len(stack)
		if n == 0 {
			continue
		}
		top := &stack[n-1]
		if top.obj && top.expectKey {
			key, _ := tok.(string)
			structural = isStructuralKey(key) && !p.redacts(key)
			top.expectKey = false
			continue
		}
		if s, isStr := tok.(string); isStr && top.obj && structural {
			// Only ':' and whitespace sit between the previous token
			// and the opening quote, so the first quote is the start.
			if q := bytes.IndexByte(b[before:], '"'); q >= 0 &&
				structuralValueSafe(s) {
				spans = append(spans, valueSpan{before + q, int(dec.InputOffset())})
			}
		}
		if top.obj {
			top.expectKey = true
		}
	}
}

// sentinelFor builds an inert, letters-only placeholder for span i.
// Letters-only keeps every digit/key-shaped rule from matching it.
func sentinelFor(salt string, i int) string {
	var sb strings.Builder
	sb.WriteString(salt)
	for {
		sb.WriteByte(byte('a' + i%26))
		i /= 26
		if i == 0 {
			break
		}
	}
	return sb.String()
}

// ApplyJSONLine redacts one JSON document (typically a JSONL record)
// like ApplyBytes, except that string values held by structural keys
// (see isStructuralKey) are stored verbatim unless a credential rule
// matches them. Non-JSON input falls back to ApplyBytes. Output keeps
// the input's key order and spacing.
func ApplyJSONLine(b []byte) []byte {
	return ApplyJSONLinePolicy(b, JSONLPolicy{})
}

// JSONLPolicy narrows the structural-field exemption of ApplyJSONLine.
type JSONLPolicy struct {
	// RedactKeys names structural keys (case-insensitive) whose values
	// are NOT exempted from PII rules: they carry personal data (for
	// example phone-number IDs) and go through normal redaction.
	RedactKeys []string
}

func (p JSONLPolicy) redacts(key string) bool {
	for _, k := range p.RedactKeys {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

// ApplyJSONLinePolicy is ApplyJSONLine with a per-call policy.
func ApplyJSONLinePolicy(b []byte, p JSONLPolicy) []byte {
	if !Enabled() || len(b) == 0 {
		return b
	}
	spans, ok := structuralSpans(b, p)
	if !ok || len(spans) == 0 {
		return Redactor().ApplyBytes(b)
	}

	salt := "apskeep"
	for bytes.Contains(b, []byte(salt)) {
		salt += "x"
	}

	masked := make([]byte, 0, len(b))
	sentinels := make([]string, len(spans))
	prev := 0
	for i, sp := range spans {
		sentinels[i] = `"` + sentinelFor(salt, i) + `"`
		masked = append(masked, b[prev:sp.start]...)
		masked = append(masked, sentinels[i]...)
		prev = sp.end
	}
	masked = append(masked, b[prev:]...)

	out := string(Redactor().ApplyBytes(masked))
	for i, sp := range spans {
		out = strings.Replace(out, sentinels[i], string(b[sp.start:sp.end]), 1)
	}
	return []byte(out)
}

// NewJSONLWriter is NewWriter for JSONL sinks whose records carry
// identifiers that must round-trip (event IDs, resource keys,
// timestamps). Each newline-terminated line is redacted via
// ApplyJSONLine; see there for the structural-field rule.
func NewJSONLWriter(w io.Writer) io.Writer {
	return NewJSONLWriterPolicy(w, JSONLPolicy{})
}

// NewJSONLWriterPolicy is NewJSONLWriter with a per-writer policy.
func NewJSONLWriterPolicy(w io.Writer, p JSONLPolicy) io.Writer {
	if w == nil {
		return io.Discard
	}
	return &jsonlWriter{inner: w, policy: p}
}

type jsonlWriter struct {
	inner  io.Writer
	policy JSONLPolicy
}

func (jw *jsonlWriter) Write(p []byte) (int, error) {
	if !Enabled() {
		return jw.inner.Write(p)
	}
	var out bytes.Buffer
	rest := p
	for len(rest) > 0 {
		line := rest
		nl := bytes.IndexByte(rest, '\n')
		if nl >= 0 {
			line, rest = rest[:nl], rest[nl+1:]
		} else {
			rest = nil
		}
		out.Write(ApplyJSONLinePolicy(line, jw.policy))
		if nl >= 0 {
			out.WriteByte('\n')
		}
	}
	if _, err := jw.inner.Write(out.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}
