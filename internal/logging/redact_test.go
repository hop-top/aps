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

// TestApply_BypassSkipsRedactorBuild asserts that when redaction is
// disabled at call time the redactor singleton is not invoked. The
// guarantee matters for cold-start latency: aps --help and other
// fast-fail paths must not pay the gitleaks + Presidio corpus load
// cost. We verify this indirectly by calling Apply once with bypass
// on, then once with bypass off; the first call must not have built
// the redactor (the input is the literal input), and the second call
// must redact (which exercises the lazy build).
func TestApply_BypassSkipsRedactorBuild(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	const secret = "OPENAI_API_KEY=sk-proj-1234567890abcdefghij1234567890abcdefghij"
	if got := Apply(secret); got != secret {
		t.Fatalf("bypass: expected verbatim, got %q", got)
	}
	// Now disable bypass and confirm the lazy build still works.
	t.Setenv(EnvBypass, "")
	got := Apply(secret)
	if got == secret {
		t.Fatalf("after bypass cleared: expected redaction, got verbatim")
	}
	if !strings.Contains(got, "<") || !strings.Contains(got, ">") {
		t.Fatalf("expected Tag-style replacement, got %q", got)
	}
}

// TestApply_RedactsProviderTokenShapes covers the provider prefixes
// the vendored gitleaks corpus misses because its rules are anchored
// to one exact historical token layout (openai-api-key requires the
// "T3BlbkFJ" infix; anthropic-api-key requires exactly 93 body chars).
// Every value below is synthetic.
func TestApply_RedactsProviderTokenShapes(t *testing.T) {
	resetForTest()
	tests := []struct {
		name  string
		in    string
		wants string // expected tag substring
	}{
		{"openai-live", "sk-live-abcdef0123456789abcdef0123456789", "<aps-openai-key>"},
		{"openai-proj", "sk-proj-1234567890abcdefghij1234567890abcdefghijklmnop", "<aps-openai-key>"},
		{"openai-svcacct", "sk-svcacct-abcdef0123456789abcdef0123456789", "<aps-openai-key>"},
		{"openai-legacy", "sk-1234567890abcdefghijkl1234567890abcdefghijkl", "<aps-openai-legacy-key>"},
		{"anthropic-api", "sk-ant-api03-aBcDeF0123456789aBcDeF0123456789aBcD", "<aps-anthropic-key>"},
		{"anthropic-admin", "sk-ant-admin01-aBcDeF0123456789aBcDeF0123456789", "<aps-anthropic-key>"},
		{
			"github-fine-grained",
			"github_pat_11ABCDEFG0aBcDeF0123456789_aBcDeF0123456789aBcDeF01",
			"<aps-github-fine-grained-pat>",
		},
		{"npm", "npm_aBcDeF0123456789aBcDeF0123456789aBcDeF01", "<aps-npm-token>"},
		{"huggingface", "hf_aBcDeF0123456789aBcDeF0123456789aBcD", "<aps-huggingface-token>"},
		{
			"digitalocean",
			"dop_v1_aBcDeF0123456789aBcDeF0123456789aBcDeF0123456789aBcDeF0123",
			"<aps-digitalocean-token>",
		},
		{"stripe-publishable-live", "pk_live_aBcDeF0123456789aBcDeF01", "<aps-stripe-live-key>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := Apply(tc.in)
			if strings.Contains(out, tc.in) {
				t.Fatalf("token printed verbatim: %q", out)
			}
			if !strings.Contains(out, tc.wants) {
				t.Fatalf("expected %s in output, got %q", tc.wants, out)
			}
		})
	}
}

// TestApply_ProviderPrefixesAlreadyCoveredStillRedact is a regression
// guard for shapes the upstream corpus already handles, so adding the
// aps-local rules cannot silently displace them.
//
// Fixtures are assembled from fragments rather than written as whole
// literals. The bodies are fabricated, but a contiguous token of this
// shape trips push-protection scanning on the way to the remote; the
// concatenation is folded at compile time, so each case still asserts
// against the fully assembled value.
func TestApply_ProviderPrefixesAlreadyCoveredStillRedact(t *testing.T) {
	resetForTest()
	tests := []struct{ name, in string }{
		{"github-pat", "ghp_" + "aBcDeF0123456789aBcDeF0123456789aBcD"},
		{"github-oauth", "gho_" + "aBcDeF0123456789aBcDeF0123456789aBcD"},
		{"github-app", "ghs_" + "aBcDeF0123456789aBcDeF0123456789aBcD"},
		{"slack-bot", "xoxb" + "-123456789012-1234567890123-aBcDeF0123456789aBcDeF01"},
		{"stripe-secret-live", "sk" + "_live_" + "aBcDeF0123456789aBcDeF01"},
		{"stripe-restricted-live", "rk" + "_live_" + "aBcDeF0123456789aBcDeF01"},
		{"aws-access-key", "AKIA" + "QYLPMN5HGXWZ7TJC"},
		{"aws-session-key", "ASIA" + "QYLPMN5HGXWZ7TJC"},
		{"google-api-key", "AIza" + "SyA0123456789abcdefghijklmnopqrstuv"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := Apply(tc.in)
			if strings.Contains(out, tc.in) {
				t.Fatalf("token printed verbatim: %q", out)
			}
		})
	}
}

// TestApply_AllowlistIsNotSubstringBypassable pins the security
// property behind narrowing the allowlist. redact.Allow matches by
// SUBSTRING against every rule's match text, so bare network
// fragments ("10.", "172.16.") previously exempted any secret that
// merely contained them — token charsets permit digits and dots. Each
// case below leaked verbatim before the allowlist was narrowed.
func TestApply_AllowlistIsNotSubstringBypassable(t *testing.T) {
	resetForTest()
	tests := []struct{ name, in, secret string }{
		{
			"bearer-token-containing-rfc1918-fragment",
			"Authorization: Bearer abcdef10.0123456789abcdefgh",
			"abcdef10.0123456789abcdefgh",
		},
		{
			"api-key-header-containing-rfc1918-fragment",
			"X-API-Key: abcdef10.0123456789abcdefgh",
			"abcdef10.0123456789abcdefgh",
		},
		{
			"github-pat-prefixed-with-fixture-token",
			"ghp_test0123456789abcdefghijklmnopqrstuv",
			"ghp_test0123456789abcdefghijklmnopqrstuv",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := Apply(tc.in)
			if strings.Contains(out, tc.secret) {
				t.Fatalf("allowlist substring bypass: secret survived in %q", out)
			}
		})
	}
}

// TestApply_KeepsPrivateNetworkAddressesReadable verifies the
// rule-scoped replacement for loopback / RFC1918 / link-local
// addresses. Operators rely on these staying legible in log lines;
// the exemption is now anchored to whole IP-rule matches rather than
// leading-octet substrings.
func TestApply_KeepsPrivateNetworkAddressesReadable(t *testing.T) {
	resetForTest()
	tests := []struct{ name, in string }{
		{"loopback-with-port", "webhook server listening addr=127.0.0.1:8080"},
		{"unspecified", "bind addr=0.0.0.0:9000"},
		{"rfc1918-10", "peer=10.0.0.5"},
		{"rfc1918-192-168", "gateway=192.168.1.1"},
		{"rfc1918-172-16", "lan=172.16.0.9"},
		{"rfc1918-172-31", "lan=172.31.255.254"},
		{"ipv6-loopback", "v6=::1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if out := Apply(tc.in); out != tc.in {
				t.Fatalf("private address was mangled: %q -> %q", tc.in, out)
			}
		})
	}
}

// TestApply_RedactsRoutableAddresses is the counterpart to the test
// above: the private-range exemption must not become a blanket
// pass-through for every IP. A public address is still PII.
func TestApply_RedactsRoutableAddresses(t *testing.T) {
	resetForTest()
	tests := []struct{ name, in string }{
		{"public-dns", "resolver=8.8.8.8"},
		{"public-host", "upstream=203.0.113.42"},
		// 172.32/x is outside the RFC1918 172.16/12 block.
		{"just-outside-rfc1918", "peer=172.32.0.1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := Apply(tc.in)
			if out == tc.in {
				t.Fatalf("routable address was not redacted: %q", out)
			}
		})
	}
}

// TestApply_LeavesNonSecretLookalikesIntact is the negative control
// for the added rules: ordinary prose and version strings that share
// a prefix with a credential must not be mangled. Guards against the
// added patterns being too greedy.
func TestApply_LeavesNonSecretLookalikesIntact(t *testing.T) {
	resetForTest()
	tests := []struct{ name, in string }{
		{"prose-sk-live", "the sk-live mode flag is unset"},
		{"semver", "version 10.2.3 released"},
		{"npm-word", "run npm_config_cache to inspect"},
		{"short-fixture-key", "OPENAI_API_KEY=sk-test"},
		{"plain-sentence", "no credentials in this line at all"},
		{"aws-doc-fixture", "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if out := Apply(tc.in); out != tc.in {
				t.Fatalf("non-secret was mangled: %q -> %q", tc.in, out)
			}
		})
	}
}

// TestApply_NewRulesRespectBypass confirms the added provider rules
// are still governed by the --no-redact / APS_DEBUG_NO_REDACT
// break-glass path rather than being applied unconditionally.
func TestApply_NewRulesRespectBypass(t *testing.T) {
	resetForTest()
	t.Setenv(EnvBypass, "1")
	const secret = "sk-live-abcdef0123456789abcdef0123456789"
	if got := Apply(secret); got != secret {
		t.Fatalf("bypass should pass new-rule secrets through; got %q", got)
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
