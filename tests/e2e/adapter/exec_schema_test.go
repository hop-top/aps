package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for the manifest `input` schema as the exec runtime honours
// it.
//
// `aps adapter exec` resolves an action's script from the manifest and
// forwards `--input` pairs as <PREFIX>_<KEY> env vars. The schema is
// now enforced on three axes: `required: true` inputs are checked
// against the caller-supplied map before the script is spawned,
// declared `default:` values fill in omitted inputs, and keys the
// action does not declare are forwarded but warned about on stderr
// (see TestExecSchema_UndeclaredInputsPassThroughWithWarning) —
// scripts may legitimately read vars the manifest does not enumerate,
// so an unknown key is a diagnostic rather than a rejection.
//
// Required-checking deliberately runs before defaults are merged, so a
// declared default never satisfies a required marker.

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

// TestExecSchema_DeclaredDefaultsApplied covers manifest-declared
// defaulting: the fixture's `list` action declares `limit` (default 10)
// and `folder` (default INBOX), and invoking `list` without them
// delivers both to the script as FIXTURE_LIMIT / FIXTURE_FOLDER. An
// input declaring no default (`query`) stays absent.
func TestExecSchema_DeclaredDefaultsApplied(t *testing.T) {
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

	for key, want := range map[string]string{
		"FIXTURE_LIMIT":  "10",
		"FIXTURE_FOLDER": "INBOX",
	} {
		got, ok := env[key]
		if !ok {
			t.Errorf("env %s absent; the manifest declares a default for it, got keys %v",
				key, fixtureEnvKeys(env))
			continue
		}
		if got != want {
			t.Errorf("env %s = %q, want declared default %q", key, got, want)
		}
	}

	// An input declaring no default gains no value out of thin air.
	if got, ok := env["FIXTURE_QUERY"]; ok {
		t.Errorf("env FIXTURE_QUERY unexpectedly present (=%q); the input declares no default, got keys %v",
			got, fixtureEnvKeys(env))
	}
}

// TestExecSchema_SuppliedInputOverridesDefault covers the precedence
// half of defaulting: a caller-supplied `--input` wins over the
// manifest's declared default, while the untouched sibling input still
// receives its default.
func TestExecSchema_SuppliedInputOverridesDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name: "schema-defaults-override",
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "list",
		"--from", "ops@example.com",
		"--input", "limit=50",
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

	if got := env["FIXTURE_LIMIT"]; got != "50" {
		t.Errorf("env FIXTURE_LIMIT = %q, want caller-supplied %q (declared default 10 must not win); got keys %v",
			got, "50", fixtureEnvKeys(env))
	}
	if got := env["FIXTURE_FOLDER"]; got != "INBOX" {
		t.Errorf("env FIXTURE_FOLDER = %q, want declared default %q; got keys %v",
			got, "INBOX", fixtureEnvKeys(env))
	}
}

// TestExecSchema_UndeclaredInputsPassThroughWithWarning covers the
// permissive-with-a-warning contract for undeclared inputs: an `--input`
// key the action does not declare is still forwarded to the script as a
// prefixed env var — backend scripts may read vars their manifest does
// not enumerate — but the runtime now names it on stderr so a typo is
// visible to the operator. The action still runs and the exit code is
// unchanged; this is a diagnostic, not a rejection.
func TestExecSchema_UndeclaredInputsPassThroughWithWarning(t *testing.T) {
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

	// Pass-through is no longer silent: the undeclared key is named on
	// stderr, keeping stdout free of diagnostics.
	diagnostic := diagnosticStderr(stderr)
	if !strings.Contains(diagnostic, "not-in-manifest") {
		t.Errorf("stderr = %q, want it to name the undeclared input key",
			diagnostic)
	}
	if !strings.Contains(diagnostic, "warn:") {
		t.Errorf("stderr = %q, want an undeclared-input warning", diagnostic)
	}
	if strings.Contains(stdout, "not-in-manifest") {
		t.Errorf("warning leaked onto stdout, which must stay machine-parseable:\n%s",
			stdout)
	}
}
