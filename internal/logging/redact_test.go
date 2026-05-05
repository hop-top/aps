package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestApply_RedactsOpenAIKey covers the canonical leak case from the
// 2026-05-02 incident: an OPENAI_API_KEY value should never reach a
// stream verbatim with default settings.
//
// The gitleaks generic-api-key rule matches the entire "KEY=VALUE"
// form so the assertion is just "raw value gone, replacement tag
// present". The key prefix may or may not be retained depending on
// which rule matched (openai-api-key keeps prefix; generic-api-key
// swallows the whole assignment).
func TestApply_RedactsOpenAIKey(t *testing.T) {
	resetForTest()
	const secret = "sk-proj-1234567890abcdefghij1234567890abcdefghijklmnop"
	in := "OPENAI_API_KEY=" + secret
	out := Apply(in)
	if strings.Contains(out, secret) {
		t.Fatalf("redact failed; raw key visible in %q", out)
	}
	if !strings.Contains(out, "<") || !strings.Contains(out, ">") {
		t.Fatalf("expected Tag-style replacement (<rule-id>), got %q", out)
	}
}

// TestApply_RedactsBearerToken covers the structured-field key-aware
// requirement: an "Authorization: Bearer xyz" line must redact xyz
// while keeping the Authorization key intact.
func TestApply_RedactsBearerToken(t *testing.T) {
	resetForTest()
	in := "Authorization: Bearer abc123def456ghi789jkl012mno345pqr678stu"
	out := Apply(in)
	if strings.Contains(out, "abc123def456ghi789jkl012mno345pqr678stu") {
		t.Fatalf("bearer token leaked: %q", out)
	}
	if !strings.Contains(out, "Authorization") {
		t.Fatalf("expected Authorization key preserved, got %q", out)
	}
}

// TestEnabled_DefaultTrue verifies the package default: redact ON.
func TestEnabled_DefaultTrue(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "")
	if !Enabled() {
		t.Fatalf("expected default Enabled() == true")
	}
}

// TestEnabled_EnvBypass verifies APS_DEBUG_NO_REDACT=1 disables.
func TestEnabled_EnvBypass(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	if Enabled() {
		t.Fatalf("expected APS_DEBUG_NO_REDACT=1 to disable")
	}
}

// TestEnabled_ViperKeyOverride verifies the redact.enabled viper key.
func TestEnabled_ViperKeyOverride(t *testing.T) {
	resetForTest()
	v := viper.New()
	v.Set(ViperKeyEnabled, false)
	SetViperForRedact(v)
	if Enabled() {
		t.Fatalf("expected viper redact.enabled=false to disable")
	}
}

// TestApply_BypassReturnsRaw verifies that when redaction is off,
// the input is returned verbatim (no Apply call cost, no rule
// initialization side-effects).
func TestApply_BypassReturnsRaw(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	in := "OPENAI_API_KEY=sk-proj-this-should-NOT-be-redacted-1234567890abcdef"
	out := Apply(in)
	if out != in {
		t.Fatalf("bypass should pass through; got %q want %q", out, in)
	}
}

// TestNewWriter_FiltersWritesByDefault wraps a buffer with NewWriter
// and asserts that secret-bearing writes are tagged before reaching
// the underlying buffer.
func TestNewWriter_FiltersWritesByDefault(t *testing.T) {
	resetForTest()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	const secret = "sk-proj-1234567890abcdefghij1234567890abcdefghij"
	n, err := w.Write([]byte("token=" + secret + "\n"))
	if err != nil {
		t.Fatalf("Write err: %v", err)
	}
	if n != len("token="+secret+"\n") {
		t.Fatalf("Write n=%d want %d (must reflect input length)", n, len("token="+secret+"\n"))
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("secret leaked through writer wrap: %q", buf.String())
	}
}

// TestNewWriter_PassesThroughWhenDisabled verifies the writer is
// a transparent forwarder when redaction is off.
func TestNewWriter_PassesThroughWhenDisabled(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	var buf bytes.Buffer
	w := NewWriter(&buf)
	const secret = "sk-proj-1234567890abcdefghij1234567890abcdefghij"
	in := "token=" + secret + "\n"
	if _, err := w.Write([]byte(in)); err != nil {
		t.Fatalf("Write err: %v", err)
	}
	if buf.String() != in {
		t.Fatalf("disabled writer altered output; got %q want %q", buf.String(), in)
	}
}

// TestApplyBytes_KeepsAllowlistedFixtures verifies the global
// allowlist (sk-test, AKIAIOSFODNN7EXAMPLE) prevents redaction of
// well-known docs/test placeholders.
func TestApplyBytes_KeepsAllowlistedFixtures(t *testing.T) {
	resetForTest()
	in := []byte("OPENAI_API_KEY=sk-test\nAWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
	out := ApplyBytes(in)
	if !bytes.Contains(out, []byte("sk-test")) {
		t.Fatalf("sk-test allowlist failed: %q", out)
	}
	if !bytes.Contains(out, []byte("AKIAIOSFODNN7EXAMPLE")) {
		t.Fatalf("AKIA EXAMPLE allowlist failed: %q", out)
	}
}

// resetForTest clears redaction state so each test starts from a
// known baseline. The Redactor singleton stays (sync.Once); we only
// need to reset the viper ref and env, which Apply re-reads each
// call.
func resetForTest() {
	vMu.Lock()
	viperRef = nil
	vMu.Unlock()
}
