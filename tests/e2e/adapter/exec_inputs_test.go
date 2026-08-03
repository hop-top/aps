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

// TestExecInputs_MalformedPairIsError asserts an --input argument
// carrying no "=" separator aborts the command and is named in the
// diagnostic.
//
// A separator-less argument is a typo for a real pair. Dropping it
// silently would make a mistyped flag indistinguishable from an
// omitted one, so exec must fail before the action script runs and
// point at the offending argument by name.
func TestExecInputs_MalformedPairIsError(t *testing.T) {
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
	if err == nil {
		t.Fatalf("expected malformed input to fail\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}

	// The action script never ran, so no pair reached it.
	env := parseFixtureEnv(stdout)
	requireNoFixtureEnv(t, env, "FIXTURE_TO")
	requireNoFixtureEnv(t, env, "FIXTURE_SUBJECT")
	requireNoFixtureEnv(t, env, "FIXTURE_BODY")

	// The diagnostic names the offending argument, not merely the flag.
	diagnostic := diagnosticStderr(stderr)
	if !strings.Contains(diagnostic, "invalid input format") {
		t.Errorf("stderr = %q, want an invalid input format diagnostic",
			diagnostic)
	}
	if !strings.Contains(diagnostic, "'body'") {
		t.Errorf("stderr = %q, want it to name the offending argument",
			diagnostic)
	}
}

// diagnosticStderr strips the startup warnings an isolated HOME emits
// so assertions only see the command's own diagnostic.
func diagnosticStderr(stderr string) string {
	var kept []string
	for line := range strings.SplitSeq(strings.TrimSpace(stderr), "\n") {
		if line == "" || strings.HasPrefix(line, "warn: bus auth:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
