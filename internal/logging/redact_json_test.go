package logging

import (
	"bytes"
	"strings"
	"testing"
)

const cardShapedID = "evt-bffab4c3-9d88-4349-9131-824401265021"

func TestApplyJSONLine_IdentifiersRoundTrip(t *testing.T) {
	resetForTest()
	in := `{"id":"` + cardShapedID + `","workspace_id":"ws-1",` +
		`"timestamp":"2026-11-19T10:11:12.123456789Z","caused_by":"` +
		cardShapedID + `","payload":{"resource":"profile:` +
		`9131-824401265021","id":"9131-824401265021"}}`
	if got := string(ApplyJSONLine([]byte(in))); got != in {
		t.Fatalf("identifiers mangled:\n in: %s\ngot: %s", in, got)
	}
}

func TestApplyJSONLine_FreeTextStillRedacted(t *testing.T) {
	resetForTest()
	in := `{"id":"` + cardShapedID + `","note":"card 4111 1111 1111 1111"}`
	got := string(ApplyJSONLine([]byte(in)))
	if strings.Contains(got, "4111") {
		t.Fatalf("free-text card leaked: %s", got)
	}
	if !strings.Contains(got, `"id":"`+cardShapedID+`"`) {
		t.Fatalf("id mangled: %s", got)
	}
}

// A credential sitting in an identifier-named field is still a
// credential.
func TestApplyJSONLine_SecretInIDFieldStillRedacted(t *testing.T) {
	resetForTest()
	const secret = "sk-proj-1234567890abcdefghij1234567890abcdefghijklmnop"
	in := `{"id":"` + secret + `"}`
	if got := string(ApplyJSONLine([]byte(in))); strings.Contains(got, secret) {
		t.Fatalf("secret in id field leaked: %s", got)
	}
}

func TestApplyJSONLine_NonJSONFallsBack(t *testing.T) {
	resetForTest()
	got := string(ApplyJSONLine([]byte("card 4111 1111 1111 1111")))
	if strings.Contains(got, "4111") {
		t.Fatalf("fallback did not redact: %s", got)
	}
}

func TestApplyJSONLine_SentinelCollision(t *testing.T) {
	resetForTest()
	in := `{"id":"` + cardShapedID + `","note":"apskeepa"}`
	if got := string(ApplyJSONLine([]byte(in))); got != in {
		t.Fatalf("got %s", got)
	}
}

func TestJSONLWriter_MultiLineAndBypass(t *testing.T) {
	resetForTest()
	var buf bytes.Buffer
	w := NewJSONLWriter(&buf)
	in := `{"id":"` + cardShapedID + `"}` + "\n" +
		`{"id":"` + cardShapedID + `","n":"4111 1111 1111 1111"}` + "\n"
	n, err := w.Write([]byte(in))
	if err != nil || n != len(in) {
		t.Fatalf("n=%d err=%v", n, err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 || lines[0] != `{"id":"`+cardShapedID+`"}` ||
		strings.Contains(lines[1], "4111") {
		t.Fatalf("got %q", buf.String())
	}
}
