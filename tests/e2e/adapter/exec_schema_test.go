package adapter_e2e

import (
	"strings"
	"testing"
)

// Characterization coverage for the manifest `input` schema.
//
// `aps adapter exec` resolves an action's script from the manifest and
// forwards whatever `--input` pairs the caller supplied as
// <PREFIX>_<KEY> env vars. It never reads the action's declared `input`
// list, so `required` and `default` are documentation rather than
// behaviour, and undeclared keys are forwarded verbatim.
//
// The tests below pin that current, unenforced behaviour. They are
// expected to be inverted — asserting rejection, defaulting, and
// filtering — once schema enforcement lands in the runtime. A failure
// here after such a change is the intended signal, not a regression.

// TestExecSchema_RequiredInputsUnenforced pins current behaviour of an
// unenforced manifest input schema: the fixture's `send` action declares
// three inputs with `required: true`, yet invoking it with none of them
// still runs the script to a zero exit and produces no validation error.
// Expected to be inverted once enforcement lands.
func TestExecSchema_RequiredInputsUnenforced(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-required",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
	)
	if err != nil {
		t.Fatalf("expected exec to succeed despite missing required inputs, got: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	if !strings.Contains(stdout, "ACTION send") {
		t.Fatalf("stub script did not run; stdout:\n%s\nstderr:\n%s",
			stdout, stderr)
	}

	env := parseFixtureEnv(stdout)

	// The declared-required inputs never reach the script, and nothing
	// rejects the call.
	for _, key := range []string{
		"FIXTURE_TO", "FIXTURE_SUBJECT", "FIXTURE_BODY",
	} {
		if got, ok := env[key]; ok {
			t.Errorf("env %s unexpectedly present (=%q); required inputs are neither supplied nor validated, got keys %v",
				key, got, fixtureEnvKeys(env))
		}
	}

	// Profile-derived vars still arrive, proving the script ran with a
	// real environment rather than being short-circuited.
	if got := env["APS_EMAIL_FROM"]; got != "ops@example.com" {
		t.Errorf("env APS_EMAIL_FROM = %q, want %q; got keys %v",
			got, "ops@example.com", fixtureEnvKeys(env))
	}

	if strings.Contains(strings.ToLower(stderr), "required") {
		t.Errorf("unexpected validation diagnostic on stderr:\n%s", stderr)
	}
}

// TestExecSchema_DeclaredDefaultsNeverApplied pins current behaviour of
// an unenforced manifest input schema: the fixture's `list` action
// declares `limit` (default 10) and `folder` (default INBOX), but
// invoking `list` without them leaves FIXTURE_LIMIT and FIXTURE_FOLDER
// absent from the script's environment — defaulting never happens in Go
// and silently falls through to the backend script. Expected to be
// inverted once enforcement lands.
func TestExecSchema_DeclaredDefaultsNeverApplied(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-defaults",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
	)
	if err != nil {
		t.Fatalf("exec fixture list: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	if !strings.Contains(stdout, "ACTION list") {
		t.Fatalf("stub script did not run; stdout:\n%s\nstderr:\n%s",
			stdout, stderr)
	}

	env := parseFixtureEnv(stdout)

	for _, key := range []string{"FIXTURE_LIMIT", "FIXTURE_FOLDER"} {
		if got, ok := env[key]; ok {
			t.Errorf("env %s unexpectedly present (=%q); manifest defaults are not applied by the runtime, got keys %v",
				key, got, fixtureEnvKeys(env))
		}
	}

	// The purely optional input is likewise absent, so the script sees
	// no <PREFIX>_* input vars at all for an all-defaulted invocation.
	if got, ok := env["FIXTURE_QUERY"]; ok {
		t.Errorf("env FIXTURE_QUERY unexpectedly present (=%q); got keys %v",
			got, fixtureEnvKeys(env))
	}
}

// TestExecSchema_UndeclaredInputsPassThrough pins current behaviour of
// an unenforced manifest input schema: an `--input` key the action does
// not declare at all is still forwarded to the script as a prefixed env
// var, because the runtime iterates the caller-supplied map rather than
// the declared input list. Expected to be inverted once enforcement
// lands.
func TestExecSchema_UndeclaredInputsPassThrough(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-undeclared",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "read",
		"--from", "ops@example.com",
		"--input", "id=42",
		"--input", "not-in-manifest=leaked",
	)
	if err != nil {
		t.Fatalf("expected exec to succeed with an undeclared input, got: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	if !strings.Contains(stdout, "ACTION read") {
		t.Fatalf("stub script did not run; stdout:\n%s\nstderr:\n%s",
			stdout, stderr)
	}

	env := parseFixtureEnv(stdout)

	got, ok := env["FIXTURE_NOT_IN_MANIFEST"]
	if !ok {
		t.Fatalf("env FIXTURE_NOT_IN_MANIFEST absent; undeclared inputs are currently forwarded unfiltered, got keys %v",
			fixtureEnvKeys(env))
	}
	if got != "leaked" {
		t.Errorf("env FIXTURE_NOT_IN_MANIFEST = %q, want %q", got, "leaked")
	}

	// The declared input is forwarded by the same undiscriminating
	// path, so both arrive identically.
	if got := env["FIXTURE_ID"]; got != "42" {
		t.Errorf("env FIXTURE_ID = %q, want %q; got keys %v",
			got, "42", fixtureEnvKeys(env))
	}
}
