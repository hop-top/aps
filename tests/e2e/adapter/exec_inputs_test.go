package adapter_e2e

import (
	"strings"
	"testing"
)

// End-to-end coverage of `aps adapter exec --input` parsing.
//
// Every assertion reads the env map the stub script echoed, so it
// pins what the CLI actually forwarded to the action script rather
// than what the flag layer merely accepted.

// requireFixtureEnv fails the test when key is absent from env, and
// reports a mismatch when its value differs from want.
func requireFixtureEnv(t *testing.T, env map[string]string, key, want string) {
	t.Helper()
	got, ok := env[key]
	if !ok {
		t.Fatalf("env %s not received by script; got keys %v",
			key, fixtureEnvKeys(env))
	}
	if got != want {
		t.Errorf("env %s = %q, want %q", key, got, want)
	}
}

// requireNoFixtureEnv fails when key reached the script at all.
func requireNoFixtureEnv(t *testing.T, env map[string]string, key string) {
	t.Helper()
	if got, ok := env[key]; ok {
		t.Errorf("env %s unexpectedly set to %q; got keys %v",
			key, got, fixtureEnvKeys(env))
	}
}

// TestExecInputs_RepeatedPairsReachScript asserts every repeated
// --input pair arrives as <PREFIX>_<UPPER_KEY>, including the
// optional input the action declares but does not require.
func TestExecInputs_RepeatedPairsReachScript(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Quarterly report",
		"--input", "body=See attachment",
		"--input", "cc=team@example.com",
	)
	if err != nil {
		t.Fatalf("exec fixture send: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	requireFixtureEnv(t, env, "FIXTURE_TO", "user@example.com")
	requireFixtureEnv(t, env, "FIXTURE_SUBJECT", "Quarterly report")
	requireFixtureEnv(t, env, "FIXTURE_BODY", "See attachment")
	requireFixtureEnv(t, env, "FIXTURE_CC", "team@example.com")
	requireFixtureEnv(t, env, "APS_EMAIL_FROM", "ops@example.com")
}

// TestExecInputs_ValueKeepsLaterSeparators asserts the key/value split
// happens on the FIRST "=" only: everything after it is the value,
// separators included. Covers a plain trailing "=" too.
func TestExecInputs_ValueKeepsLaterSeparators(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=a=b",
		"--input", "body=k=v&x=y=z",
		"--input", "cc=trailing=",
	)
	if err != nil {
		t.Fatalf("exec fixture send: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	requireFixtureEnv(t, env, "FIXTURE_SUBJECT", "a=b")
	requireFixtureEnv(t, env, "FIXTURE_BODY", "k=v&x=y=z")
	requireFixtureEnv(t, env, "FIXTURE_CC", "trailing=")
}

// TestExecInputs_EmptyValueIsForwarded asserts `--input cc=` is a
// well-formed pair: the key reaches the script with an empty value,
// distinct from the key being absent entirely.
func TestExecInputs_EmptyValueIsForwarded(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Subject",
		"--input", "body=Body",
		"--input", "cc=",
	)
	if err != nil {
		t.Fatalf("exec fixture send: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	requireFixtureEnv(t, env, "FIXTURE_CC", "")
	requireFixtureEnv(t, env, "FIXTURE_TO", "user@example.com")
}

// TestExecInputs_ShortAndLongSpellings asserts -i and --input are
// interchangeable and accumulate into the same input map.
func TestExecInputs_ShortAndLongSpellings(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "reply",
		"--from", "ops@example.com",
		"-i", "id=7131",
		"--input", "body=Thanks!",
	)
	if err != nil {
		t.Fatalf("exec fixture reply: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	requireFixtureEnv(t, env, "FIXTURE_ID", "7131")
	requireFixtureEnv(t, env, "FIXTURE_BODY", "Thanks!")
}

// TestExecInputs_HyphenKeyBecomesUnderscore asserts hyphenated input
// names are normalised to underscores and upper-cased on the way to
// the script env.
func TestExecInputs_HyphenKeyBecomesUnderscore(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "read",
		"--from", "ops@example.com",
		"-i", "id=7131",
		"-i", "preview-only=true",
	)
	if err != nil {
		t.Fatalf("exec fixture read: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	requireFixtureEnv(t, env, "FIXTURE_ID", "7131")
	requireFixtureEnv(t, env, "FIXTURE_PREVIEW_ONLY", "true")
}

// TestExecInputs_MalformedPairSilentlyDropped is a characterization
// test: it pins the CURRENT behaviour of an --input argument that
// carries no "=" separator.
//
// parseInputs keeps a pair only when strings.SplitN(kv, "=", 2)
// yields two parts, so a separator-less argument is discarded with no
// diagnostic. The command still succeeds, the action script never
// sees a corresponding env var, and no output names the dropped
// argument or calls it malformed — a mistyped flag is therefore
// indistinguishable from an omitted one.
//
// This behaviour is a known defect, not a guarantee. When malformed
// input is promoted to an error, this test is expected to be
// inverted: the assertions below become "command fails and names the
// offending argument".
func TestExecInputs_MalformedPairSilentlyDropped(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Subject",
		"--input", "body",
	)
	// Current behaviour: no error despite a required input missing.
	if err != nil {
		t.Fatalf("expected malformed input to be tolerated, got: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	// The well-formed pairs still arrive.
	requireFixtureEnv(t, env, "FIXTURE_TO", "user@example.com")
	requireFixtureEnv(t, env, "FIXTURE_SUBJECT", "Subject")
	// The separator-less argument produces no env var at all: not
	// under its own name, and not as an empty-valued key.
	requireNoFixtureEnv(t, env, "FIXTURE_BODY")

	// No diagnostic: the only stderr lines are unrelated startup
	// warnings, and none of them mentions the input flag.
	for line := range strings.SplitSeq(strings.TrimSpace(stderr), "\n") {
		if line == "" || strings.HasPrefix(line, "warn: bus auth:") {
			continue
		}
		t.Errorf("unexpected stderr line for dropped input: %q", line)
	}
	if strings.Contains(strings.ToLower(stderr), "input") {
		t.Errorf("expected stderr to say nothing about the dropped input, got: %q",
			stderr)
	}
}
