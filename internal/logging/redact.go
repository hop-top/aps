// Package logging — redact integration.
//
// Wires kit/core/redact at four canonical choke points:
//
//   1. Logger sink: SetViper wraps the kit logger's writer with a
//      redacting io.Writer so every logging.GetLogger().X(...) line
//      passes through redact.Apply before reaching os.Stderr.
//   2. Stdout/stderr formatter: Print/Println/Printf/Fprint helpers
//      below mirror the fmt.* shape but redact their formatted output.
//      Migrate HIGH-severity stdout sites (env, run, adapter exec,
//      session inspect, a2a get-task) to use these.
//   3. HTTP response body: ApplyBytes wraps json.Marshal output before
//      w.Write in webhook + agentprotocol handlers.
//   4. Persisted log files: NewWriter wraps the os.OpenFile target for
//      adapter subprocess stdout.log / stderr.log.
//
// Default policy: ON, Tag strategy (<openai-api-key>, <bearer-token>).
//
// Bypass: --no-redact persistent flag (kit/cli Global) and the env var
// APS_DEBUG_NO_REDACT=1. Both must be explicit; never default. Bypass
// is honored at Apply/ApplyBytes time so a single Redactor instance
// can be shared across goroutines without rebuilding it for each
// request.
//
// See docs/cli/redaction.md for the threat model.
package logging

import (
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"hop.top/kit/go/core/redact"
)

// EnvBypass is the env var that disables redaction when set to a
// truthy value. Documented in docs/cli/redaction.md and gated only
// for break-glass diagnosis where the operator has confirmed the
// output sink is private.
const EnvBypass = "APS_DEBUG_NO_REDACT"

// ViperKeyEnabled is the viper key controlling whether redaction is
// applied. Bound to the --no-redact persistent flag (inverted) at
// CLI init time. Default true.
const ViperKeyEnabled = "redact.enabled"

var (
	defaultOnce sync.Once
	defaultRdc  *redact.Redactor

	// vMu guards viperRef so readers see a consistent pointer.
	vMu      sync.RWMutex
	viperRef *viper.Viper
)

// SetViperForRedact records the viper instance that controls the
// runtime --no-redact toggle. Called from logger.go::SetViper so
// callers don't have to wire it twice. Safe to call multiple times.
func SetViperForRedact(v *viper.Viper) {
	vMu.Lock()
	viperRef = v
	vMu.Unlock()
}

// Redactor returns the package-singleton redactor. First call eagerly
// loads the vendored gitleaks corpus + Presidio PII pack via
// redact.Default(). Subsequent calls return the same instance.
//
// kit/core/redact's Default uses the Mask strategy by default; aps
// overrides to Tag for diagnosable output (<openai-api-key>) per the
// recommendation in kit/core/redact/README.md.
//
// aps-domain extensions added on top of Default():
//
//   - aps-bearer-header — matches "Authorization: Bearer <token>"
//     verbatim. The gitleaks Bearer pattern only fires inside
//     curl -H contexts; we need it to fire on raw header strings
//     (e.g. when a future log line includes r.Header). Keeps the
//     "Authorization: Bearer " prefix and drops the token.
//   - aps-x-api-key-header — matches "X-API-Key: <token>" and
//     "X-Api-Key: <token>". Same rationale.
//   - aps-aps-signature — matches "X-APS-Signature: sha256=<hex>"
//     used by webhook HMAC verification.
//   - aps-generic-bearer — matches a bare "Bearer <token>" that
//     wasn't caught by the header rules (e.g. "saw token Bearer xyz").
func Redactor() *redact.Redactor {
	defaultOnce.Do(func() {
		r := redact.Default()
		// Pass-through known-safe placeholders so docs/test fixtures
		// don't get mangled.
		r.Allow("sk-test", "AKIAIOSFODNN7EXAMPLE", "ghp_test")
		// aps-domain rules. Order matters: header-shape rules fire
		// first so they keep the key name visible; the bare Bearer
		// rule mops up tokens not in a header context.
		_, _ = r.AddRule(
			"aps-bearer-header",
			`(?i)(authorization)\s*[:=]\s*Bearer\s+([\w=~@.+/-]{8,})`,
			"",
		)
		_, _ = r.AddRule(
			"aps-x-api-key-header",
			`(?i)(x-api-key)\s*[:=]\s*([\w=~@.+/-]{8,})`,
			"",
		)
		_, _ = r.AddRule(
			"aps-aps-signature",
			`(?i)(x-aps-signature)\s*[:=]\s*sha256=([a-f0-9]{32,})`,
			"",
		)
		_, _ = r.AddRule(
			"aps-generic-bearer",
			`\bBearer\s+([\w=~@.+/-]{16,})`,
			"",
		)
		// Custom strategy: aps-domain header rules keep the key name
		// visible (e.g. "Authorization: <aps-bearer-header>"), every
		// other rule falls back to the standard Tag strategy.
		_, _ = r.SetReplacement(redact.Custom, apsCustomReplacement)
		defaultRdc = r
	})
	return defaultRdc
}

// apsCustomReplacement renders a redaction tag while preserving the
// key portion of header-shape matches. For the rule classes:
//
//   - aps-bearer-header: matches "Authorization: Bearer <token>";
//     replacement keeps the leading "<key>: Bearer " prefix and tags
//     only the token portion.
//   - aps-x-api-key-header: matches "X-API-Key: <token>"; same shape.
//   - aps-aps-signature: matches "X-APS-Signature: sha256=<hex>";
//     keeps the key, tags the hex value.
//
// Other rules (gitleaks, Presidio, aps-generic-bearer) fall back to
// the kit Tag default — the entire match is replaced with the rule
// label.
func apsCustomReplacement(m redact.Match) string {
	switch m.RuleID {
	case "aps-bearer-header":
		key, _ := splitHeaderMatch(m.Original, headerSepBearer)
		return key + " Bearer <aps-bearer-header>"
	case "aps-x-api-key-header":
		key, _ := splitHeaderMatch(m.Original, headerSepKV)
		return key + " <aps-x-api-key-header>"
	case "aps-aps-signature":
		key, _ := splitHeaderMatch(m.Original, headerSepKV)
		return key + " sha256=<aps-aps-signature>"
	}
	// Default: behave like Tag (rule label, value-free).
	return "<" + m.RuleID + ">"
}

// headerSepRE captures the key portion up to (and including) the
// separator. Mirrors the AddRule patterns above.
type headerSep int

const (
	headerSepBearer headerSep = iota
	headerSepKV
)

var (
	headerKeyBearerRE = regexp.MustCompile(`(?i)^(\S+?)\s*[:=]\s*Bearer\s+`)
	headerKeyKVRE     = regexp.MustCompile(`(?i)^(\S+?)\s*[:=]\s*`)
)

// splitHeaderMatch returns the "key:" prefix and the value tail for a
// matched header. Used by apsCustomReplacement so the rendered output
// preserves the exact key spelling that appeared on the wire (case +
// hyphenation), regardless of which rule case-folded it during match.
func splitHeaderMatch(orig string, sep headerSep) (keyWithColon, val string) {
	var re *regexp.Regexp
	switch sep {
	case headerSepBearer:
		re = headerKeyBearerRE
	default:
		re = headerKeyKVRE
	}
	loc := re.FindStringSubmatchIndex(orig)
	if loc == nil || len(loc) < 4 {
		return "", orig
	}
	key := orig[loc[2]:loc[3]]
	val = strings.TrimSpace(orig[loc[1]:])
	return key + ":", val
}

// Enabled reports whether redaction should be applied. Order:
//
//  1. APS_DEBUG_NO_REDACT env (truthy) → false.
//  2. viper key "redact.enabled" present → its value (default true,
//     inverted by --no-redact flag).
//  3. Otherwise → true.
//
// Read fresh on every call so flag/env changes during a long-running
// process (rare in CLI; relevant for `aps serve`) take effect.
func Enabled() bool {
	if v := os.Getenv(EnvBypass); v != "" && truthy(v) {
		return false
	}
	vMu.RLock()
	v := viperRef
	vMu.RUnlock()
	if v == nil {
		return true
	}
	if !v.IsSet(ViperKeyEnabled) {
		return true
	}
	return v.GetBool(ViperKeyEnabled)
}

// Apply runs s through the package redactor when enabled; returns s
// unchanged when disabled.
func Apply(s string) string {
	if !Enabled() || s == "" {
		return s
	}
	return Redactor().Apply(s)
}

// ApplyBytes is the []byte counterpart to Apply.
func ApplyBytes(b []byte) []byte {
	if !Enabled() || len(b) == 0 {
		return b
	}
	return Redactor().ApplyBytes(b)
}

// NewWriter wraps w so every Write passes through ApplyBytes. When
// redaction is disabled at write-time the wrapper is a transparent
// pass-through (zero copy). Safe for concurrent Write only insofar as
// the wrapped writer is.
//
// Use for kit logger sinks, persisted log files, and any os.File where
// the raw stream is the leak surface.
func NewWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return &redactWriter{inner: w}
}

type redactWriter struct {
	inner io.Writer
}

func (rw *redactWriter) Write(p []byte) (int, error) {
	if !Enabled() {
		return rw.inner.Write(p)
	}
	out := Redactor().ApplyBytes(p)
	// Honor the io.Writer contract: return n based on input length
	// regardless of whether redaction shortened/lengthened the bytes.
	// Callers care about "did Write consume my buffer", not about the
	// post-redaction byte count.
	if _, err := rw.inner.Write(out); err != nil {
		return 0, err
	}
	return len(p), nil
}

// truthy mirrors strconv.ParseBool's permissive subset for ergonomic
// env-var parsing without dragging strconv into the hot path.
func truthy(v string) bool {
	switch v {
	case "1", "t", "T", "true", "TRUE", "True", "y", "Y", "yes", "YES", "Yes", "on", "ON", "On":
		return true
	}
	return false
}
