package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitSecretRef(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantRef string
		wantOK  bool
	}{
		{name: "plain literal", value: "xoxb-literal", wantRef: "", wantOK: false},
		{name: "empty", value: "", wantRef: "", wantOK: false},
		{name: "secret ref", value: "secret:MY_KEY", wantRef: "MY_KEY", wantOK: true},
		{name: "secret ref trims space", value: "secret:  MY_KEY  ", wantRef: "MY_KEY", wantOK: true},
		{name: "lowercase ref preserved", value: "secret:twilio_sid", wantRef: "twilio_sid", wantOK: true},
		{name: "outer space tolerated", value: "  secret:MY_KEY", wantRef: "MY_KEY", wantOK: true},
		{name: "empty ref is not a ref", value: "secret:", wantRef: "", wantOK: false},
		{name: "empty ref with space", value: "secret:   ", wantRef: "", wantOK: false},
		{name: "prefix only in middle is literal", value: "not-a-secret:KEY", wantRef: "", wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, ok := SplitSecretRef(tc.value)
			if ok != tc.wantOK || ref != tc.wantRef {
				t.Fatalf("SplitSecretRef(%q) = (%q, %v), want (%q, %v)", tc.value, ref, ok, tc.wantRef, tc.wantOK)
			}
		})
	}
}

// stubLookup records the keys queried so tests can assert the resolution order
// and, critically, that a failed secret: reference never queries the binding key.
type stubLookup struct {
	values map[string]string
	asked  []string
	err    error
}

func (s *stubLookup) lookup(name string) (string, bool) {
	s.asked = append(s.asked, name)
	if s.err != nil {
		return "", false
	}
	v, ok := s.values[name]
	return v, ok
}

func TestResolveSecretValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		store     map[string]string
		env       map[string]string
		want      string
		wantFound bool
	}{
		{
			name:      "literal wins without any lookup",
			value:     "xoxb-literal",
			want:      "xoxb-literal",
			wantFound: true,
		},
		{
			name:      "secret ref resolves from store",
			value:     "secret:twilio_sid",
			store:     map[string]string{"twilio_sid": "AC-from-store"},
			env:       map[string]string{"twilio_sid": "AC-from-env"},
			want:      "AC-from-store",
			wantFound: true,
		},
		{
			name:      "secret ref falls back to env when store misses",
			value:     "secret:TWILIO_SID",
			env:       map[string]string{"TWILIO_SID": "AC-from-env"},
			want:      "AC-from-env",
			wantFound: true,
		},
		{
			name:      "unresolved secret ref yields not-found",
			value:     "secret:ABSENT",
			want:      "",
			wantFound: false,
		},
		{
			name:      "store empty value is treated as absent",
			value:     "secret:BLANK",
			store:     map[string]string{"BLANK": ""},
			env:       map[string]string{"BLANK": "env-value"},
			want:      "env-value",
			wantFound: true,
		},
		{
			name:      "empty value is not found",
			value:     "",
			want:      "",
			wantFound: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &stubLookup{values: tc.store}
			env := &stubLookup{values: tc.env}
			got, found := ResolveSecretValue(tc.value, store.lookup, env.lookup)
			if got != tc.want || found != tc.wantFound {
				t.Fatalf("ResolveSecretValue(%q) = (%q, %v), want (%q, %v)", tc.value, got, found, tc.want, tc.wantFound)
			}
		})
	}
}

// The defect being fixed: an explicit secret: reference that fails to resolve
// must NOT silently fall through to a different variable the operator never
// named. Guards against reintroducing the os.Getenv(key) fallthrough.
func TestResolveSecretValueNeverConsultsBindingKey(t *testing.T) {
	store := &stubLookup{}
	env := &stubLookup{values: map[string]string{"TWILIO_ACCOUNT_SID": "WRONG-ambient-value"}}

	got, found := ResolveSecretValue("secret:twilio_sid", store.lookup, env.lookup)
	if found || got != "" {
		t.Fatalf("unresolved secret ref returned (%q, %v); want empty and not-found", got, found)
	}
	for _, asked := range append(store.asked, env.asked...) {
		if asked == "TWILIO_ACCOUNT_SID" {
			t.Fatalf("resolver consulted binding key %q; a secret: ref must only consult the name it declares", asked)
		}
	}
	if len(store.asked) != 1 || store.asked[0] != "twilio_sid" {
		t.Fatalf("store lookups = %v, want exactly [twilio_sid]", store.asked)
	}
	if len(env.asked) != 1 || env.asked[0] != "twilio_sid" {
		t.Fatalf("env lookups = %v, want exactly [twilio_sid]", env.asked)
	}
}

func TestResolveSecretValueLiteralSkipsLookups(t *testing.T) {
	store := &stubLookup{values: map[string]string{"anything": "x"}}
	env := &stubLookup{values: map[string]string{"anything": "y"}}

	got, found := ResolveSecretValue("plain-literal", store.lookup, env.lookup)
	if got != "plain-literal" || !found {
		t.Fatalf("literal resolution = (%q, %v), want (plain-literal, true)", got, found)
	}
	if len(store.asked) != 0 || len(env.asked) != 0 {
		t.Fatalf("literal triggered lookups store=%v env=%v; want none", store.asked, env.asked)
	}
}

// A broken/unavailable secret backend must degrade to the env fallback rather
// than fail the webhook outright.
func TestResolveSecretValueStoreErrorFallsBackToEnv(t *testing.T) {
	store := &stubLookup{err: errors.New("keyring locked")}
	env := &stubLookup{values: map[string]string{"KEY": "env-value"}}

	got, found := ResolveSecretValue("secret:KEY", store.lookup, env.lookup)
	if got != "env-value" || !found {
		t.Fatalf("got (%q, %v), want (env-value, true)", got, found)
	}
}

func TestResolveSecretValueNilLookups(t *testing.T) {
	got, found := ResolveSecretValue("secret:KEY", nil, nil)
	if found || got != "" {
		t.Fatalf("got (%q, %v), want empty and not-found", got, found)
	}
	if got, found := ResolveSecretValue("literal", nil, nil); got != "literal" || !found {
		t.Fatalf("literal with nil lookups = (%q, %v), want (literal, true)", got, found)
	}
}

// Credential resolution runs on every inbound webhook, so the backing store
// must be read once per profile, not once per resolution: a keyring backend
// would otherwise touch the OS keychain on every message.
func TestProfileSecretLookupReadsStoreOncePerProfile(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	// Only the data dir is redirected: XDG_CONFIG_HOME is process-global and
	// races tests that set it via os.Setenv while parallel. The default file
	// backend needs no config file.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	profileDir, err := GetProfileDir("p1")
	if err != nil {
		t.Skipf("profile dir unavailable: %v", err)
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	secretsFile := filepath.Join(profileDir, "secrets.env")
	if err := os.WriteFile(secretsFile, []byte("twilio_sid=AC-original\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if v, ok := ProfileSecretLookup("p1")("twilio_sid"); !ok || v != "AC-original" {
		t.Fatalf("first lookup = (%q, %v), want (AC-original, true)", v, ok)
	}

	// Rewrite the file. A fresh ProfileSecretLookup call must still serve the
	// cached value; if it re-read, the memoisation is per-call, not per-profile.
	if err := os.WriteFile(secretsFile, []byte("twilio_sid=AC-rewritten\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if v, _ := ProfileSecretLookup("p1")("twilio_sid"); v != "AC-original" {
		t.Fatalf("second lookup re-read the store (got %q); memoisation is not shared across calls", v)
	}

	// After an explicit reset the new contents become visible.
	ResetProfileSecretCache()
	if v, _ := ProfileSecretLookup("p1")("twilio_sid"); v != "AC-rewritten" {
		t.Fatalf("after reset = %q, want AC-rewritten", v)
	}
}

func TestProfileSecretLookupEmptyProfile(t *testing.T) {
	if lookup := ProfileSecretLookup("   "); lookup != nil {
		t.Fatal("empty profile should yield a nil lookup")
	}
}

// A failed store read must not be cached. A vault that is briefly unreachable
// at the first webhook would otherwise poison the profile for the lifetime of
// the process: every later request resolves empty even after the vault
// recovers, and only a restart clears it.
func TestProfileSecretLookupRetriesAfterFailure(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	profileDir, err := GetProfileDir("p1")
	if err != nil {
		t.Skipf("profile dir unavailable: %v", err)
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	secretsFile := filepath.Join(profileDir, "secrets.env")

	// Make the store unreadable: a directory where a file is expected makes
	// LoadProfileSecrets fail rather than report "no secrets".
	if err := os.MkdirAll(secretsFile, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, ok := ProfileSecretLookup("p1")("tok"); ok {
		t.Fatal("expected no value while the store is unreadable")
	}

	// The store becomes readable.
	if err := os.Remove(secretsFile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretsFile, []byte("tok=real-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := ProfileSecretLookup("p1")("tok")
	if !ok || got != "real-secret" {
		t.Fatalf("after recovery got (%q, %v); want (real-secret, true) — a failed read was cached", got, ok)
	}
}

// A successful read is still cached, so the happy path does not regress into
// re-reading the backing store on every request.
func TestProfileSecretLookupCachesSuccessfulRead(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	profileDir, err := GetProfileDir("p2")
	if err != nil {
		t.Skipf("profile dir unavailable: %v", err)
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	secretsFile := filepath.Join(profileDir, "secrets.env")
	if err := os.WriteFile(secretsFile, []byte("tok=first\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, _ := ProfileSecretLookup("p2")("tok"); got != "first" {
		t.Fatalf("first read = %q, want first", got)
	}

	// Rewrite: a cached success must not be re-read.
	if err := os.WriteFile(secretsFile, []byte("tok=second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := ProfileSecretLookup("p2")("tok"); got != "first" {
		t.Fatalf("second read = %q, want the cached \"first\"", got)
	}
}

// An absent store is a successful, empty read — not a failure — so it must be
// cached rather than retried on every request.
func TestProfileSecretLookupCachesAbsentStore(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	if _, ok := ProfileSecretLookup("p3")("tok"); ok {
		t.Fatal("expected not-found for a profile with no secrets")
	}
	if _, ok := ProfileSecretLookup("p3")("tok"); ok {
		t.Fatal("expected not-found on the second lookup too")
	}
}
