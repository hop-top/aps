package adapter_e2e

import (
	"strings"
	"testing"
)

// End-to-end coverage of the `aps adapter exec` argument and dispatch
// surface: arity enforcement, adapter/action resolution, and the
// strategy gate. Assertions target substrings of the real error text
// emitted by the compiled binary, so a wrong-but-non-nil error cannot
// satisfy them.

// nonScriptAdapterManifest renders a manifest for an adapter whose
// strategy is not script, so exec must refuse to dispatch it.
func nonScriptAdapterManifest(name string, strategy string) string {
	return "api_version: v1\n" +
		"kind: adapter\n" +
		"name: " + name + "\n" +
		"type: protocol\n" +
		"strategy: " + strategy + "\n" +
		"description: Non-script adapter for exec strategy-gate tests\n" +
		"config: {}\n"
}

// requireExecFailure asserts the command failed and that its combined
// output carries every expected fragment.
func requireExecFailure(
	t *testing.T,
	stdout, stderr string,
	err error,
	wantFragments ...string,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected failure, got success\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}
	combined := stdout + stderr
	for _, want := range wantFragments {
		if !strings.Contains(combined, want) {
			t.Errorf("output missing %q\nstdout: %s\nstderr: %s",
				want, stdout, stderr)
		}
	}
}

// TestExecDispatch_HappyPath is the baseline: a known adapter plus a
// known action returns that action's stdout.
func TestExecDispatch_HappyPath(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "read",
		"--from", "ops@example.com",
		"--input", "id=7131",
	)
	if err != nil {
		t.Fatalf("exec read: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "ACTION read") {
		t.Errorf("stdout missing action marker:\n%s", stdout)
	}

	env := parseFixtureEnv(stdout)
	if got := env["FIXTURE_ID"]; got != "7131" {
		t.Errorf("env FIXTURE_ID = %q, want %q; got keys %v",
			got, "7131", fixtureEnvKeys(env))
	}
	if got := env["APS_EMAIL_FROM"]; got != "ops@example.com" {
		t.Errorf("env APS_EMAIL_FROM = %q, want %q; got keys %v",
			got, "ops@example.com", fixtureEnvKeys(env))
	}
}

// TestExecDispatch_WrongArity covers cobra.ExactArgs(2): zero, one, and
// three positional args must each fail with the arity diagnostic.
func TestExecDispatch_WrongArity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		args     []string
		received string
	}{
		{name: "no args", args: nil, received: "received 0"},
		{
			name:     "adapter only",
			args:     []string{execFixtureName},
			received: "received 1",
		},
		{
			name:     "extra arg",
			args:     []string{execFixtureName, "send", "extra"},
			received: "received 3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()
			writeExecFixtureAdapter(t, home, execFixtureOptions{})

			args := append([]string{"adapter", "exec"}, tc.args...)
			args = append(args, "--from", "ops@example.com")
			stdout, stderr, err := runAPS(t, home, args...)

			requireExecFailure(t, stdout, stderr, err,
				"accepts 2 arg(s)", tc.received)
		})
	}
}

// TestExecDispatch_UnknownAdapter asserts the registry lookup failure
// names the missing adapter.
func TestExecDispatch_UnknownAdapter(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", "ghost", "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
	)

	requireExecFailure(t, stdout, stderr, err,
		"ghost", "adapter not found")
}

// TestExecDispatch_UnknownAction asserts a known adapter with an
// unknown action fails at manifest action resolution, naming the
// action rather than the adapter.
func TestExecDispatch_UnknownAction(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "archive",
		"--from", "ops@example.com",
	)

	requireExecFailure(t, stdout, stderr, err,
		`action "archive" not found in manifest`)
}

// TestExecDispatch_NonScriptStrategy asserts the strategy gate rejects
// adapters that do not use the script strategy, and that the message
// names the offending strategy.
func TestExecDispatch_NonScriptStrategy(t *testing.T) {
	t.Parallel()

	for _, strategy := range []string{"subprocess", "builtin"} {
		t.Run(strategy, func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()

			const name = "nonscript"
			writeGlobalAdapter(t, home, name,
				nonScriptAdapterManifest(name, strategy))

			stdout, stderr, err := runAPS(t, home,
				"adapter", "exec", name, "send",
				"--from", "ops@example.com",
			)

			requireExecFailure(t, stdout, stderr, err,
				`adapter "nonscript" uses `+strategy+
					" strategy; exec requires script")
		})
	}
}
