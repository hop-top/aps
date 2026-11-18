package logging

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
)

const streamSecret = "sk-proj-1234567890abcdefghijABCDEFGHIJ1234567890abcdef"

// A secret split across two Write calls must still be redacted.
func TestStreamWriter_SecretSplitAcrossWrites(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "")
	var out bytes.Buffer
	w := NewStreamWriter(&out)
	line := "OPENAI_API_KEY=" + streamSecret + "\n"
	cut := len("OPENAI_API_KEY=sk-proj-1234")
	if _, err := w.Write([]byte(line[:cut])); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(line[cut:])); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, streamSecret) || strings.Contains(got, "ABCDEFGHIJ1234567890") {
		t.Fatalf("split secret leaked: %q", got)
	}
}

// Large binary stream must come out byte-identical.
func TestStreamWriter_BinaryPassthrough(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "")
	rng := rand.New(rand.NewSource(1))
	data := make([]byte, 4<<20)
	rng.Read(data)
	// Seed text that would otherwise trigger a rule inside binary.
	copy(data[1000:], []byte(`"supplier":"Organization: Go Authors","externalRefs":[{"a":1}]`))
	var out bytes.Buffer
	w := NewStreamWriter(&out)
	for off := 0; off < len(data); off += 7777 {
		end := off + 7777
		if end > len(data) {
			end = len(data)
		}
		if _, err := w.Write(data[off:end]); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()
	if !bytes.Equal(out.Bytes(), data) {
		t.Fatalf("binary stream altered: got %d bytes want %d", out.Len(), len(data))
	}
}

// A long line with no newline is flushed at the cap, and the
// trailing partial line is flushed on Close.
func TestStreamWriter_LongLineAndClose(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "")
	var out bytes.Buffer
	w := NewStreamWriter(&out)
	long := strings.Repeat("a", 3*maxStreamBuffer)
	_, _ = w.Write([]byte(long))
	if out.Len() == 0 {
		t.Fatalf("expected flush at buffer cap")
	}
	_, _ = w.Write([]byte("tail"))
	_ = w.Close()
	if out.String() != long+"tail" {
		t.Fatalf("long line altered: len=%d", out.Len())
	}
}

// Multi-byte UTF-8 split by the cap must survive intact.
func TestStreamWriter_UTF8AtCap(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "")
	var out bytes.Buffer
	w := NewStreamWriter(&out)
	s := strings.Repeat("é", 2*maxStreamBuffer)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	if out.String() != s {
		t.Fatalf("utf8 text altered")
	}
}

func TestStreamWriter_DisabledPassthrough(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	var out bytes.Buffer
	w := NewStreamWriter(&out)
	in := "OPENAI_API_KEY=" + streamSecret
	_, _ = w.Write([]byte(in))
	if out.String() != in {
		t.Fatalf("disabled redaction should pass through immediately, got %q", out.String())
	}
}
