package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for the manifest `input` schema as the runtime treats it.
//
// `aps adapter exec` resolves an action's script from the manifest and
// forwards the caller's `--input` pairs as <PREFIX>_<KEY> env vars.
// `required: true` is enforced ahead of the spawn — see the enforced
// tests below.
//
// The remaining tests are characterization: `default:` and undeclared
// keys are still documentation rather than behaviour, so defaulting
// never happens and undeclared keys are forwarded verbatim. Those are
// expected to be inverted as the corresponding enforcement lands; a
// failure there after such a change is the intended signal, not a
// regression.

// TestExecSchema_RequiredInputsEnforced covers the enforced half of the
// manifest input schema: the fixture's `send` action declares three
// inputs with `required: true`, and invoking it with none of them is
// rejected before the script is spawned. The diagnostic names all three
// missing inputs at once rather than failing on the first.
func TestExecSchema_RequiredInputsEnforced(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-required",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
	)
	if err == nil {
		t.Fatalf("expected exec to fail with missing required inputs\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}

	// Every missing input is named in one diagnostic.
	for _, want := range []string{"to", "subject", "body"} {
		if !strings.Contains(stderr, "'"+want+"'") {
			t.Errorf("stderr does not name missing input %q:\n%s",
				want, stderr)
		}
	}
	if !strings.Contains(stderr, "missing required input") {
		t.Errorf("stderr lacks the required-input diagnostic:\n%s", stderr)
	}

	// The rejection happens before exec, so the script never ran.
	if strings.Contains(stdout, "ACTION send") {
		t.Fatalf("stub script ran despite rejection; stdout:\n%s", stdout)
	}
	env := parseFixtureEnv(stdout)
	if len(env) != 0 {
		t.Errorf("script environment observed after rejection; got keys %v",
			fixtureEnvKeys(env))
	}
}

// TestExecSchema_RequiredInputsSatisfied is the companion of the
// rejection case: supplying every declared-required input lets the same
// action through to the script unchanged.
func TestExecSchema_RequiredInputsSatisfied(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-required-ok",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Hello",
		"--input", "body=Message body",
	)
	if err != nil {
		t.Fatalf("exec with all required inputs: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	if !strings.Contains(stdout, "ACTION send") {
		t.Fatalf("stub script did not run; stdout:\n%s\nstderr:\n%s",
			stdout, stderr)
	}

	env := parseFixtureEnv(stdout)
	if got := env["APS_EMAIL_FROM"]; got != "ops@example.com" {
		t.Errorf("env APS_EMAIL_FROM = %q, want %q; got keys %v",
			got, "ops@example.com", fixtureEnvKeys(env))
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
