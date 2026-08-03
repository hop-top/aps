package adapter_e2e

import (
	"strings"
	"testing"
)

// End-to-end coverage of the From-address resolution `aps adapter exec`
// performs before dispatching an action. Priority is --from first, then
// the kit-managed root --profile global; every other combination is an
// error the user must see.
//
// The fixture stub echoes its environment, so the resolved address is
// asserted on APS_EMAIL_FROM rather than inferred from exit status.

// createProfile registers a profile in the isolated HOME. An empty
// email omits --email entirely, which (stdin not being a TTY under
// `go test`) leaves profile.email unset instead of prompting.
func createProfile(t *testing.T, home, id, email string) {
	t.Helper()

	args := []string{"profile", "create", id}
	if email != "" {
		args = append(args, "--email", email)
	}

	stdout, stderr, err := runAPS(t, home, args...)
	if err != nil {
		t.Fatalf("profile create %s: %v\nstdout: %s\nstderr: %s",
			id, err, stdout, stderr)
	}
}

// execFixtureSend runs the fixture's `send` action with the required
// inputs already supplied, so callers only vary the resolution flags.
func execFixtureSend(
	t *testing.T,
	home, name string,
	flags ...string,
) (string, string, error) {
	t.Helper()

	args := make([]string, 0, 4+len(flags)+6)
	args = append(args, "adapter", "exec", name, "send")
	args = append(args, flags...)
	args = append(args,
		"--input", "to=user@example.com",
		"--input", "subject=Hello",
		"--input", "body=Message body",
	)
	return runAPS(t, home, args...)
}

// assertFromReachedScript asserts the stub script received want as
// APS_EMAIL_FROM, proving the resolved address survived the whole
// dispatch path and not just the flag parse.
func assertFromReachedScript(t *testing.T, stdout, want string) {
	t.Helper()

	env := parseFixtureEnv(stdout)
	got, ok := env["APS_EMAIL_FROM"]
	if !ok {
		t.Fatalf("APS_EMAIL_FROM not received by script; got keys %v\nstdout: %s",
			fixtureEnvKeys(env), stdout)
	}
	if got != want {
		t.Errorf("APS_EMAIL_FROM = %q, want %q", got, want)
	}
}

// TestExecProfile_FromFlagWins asserts --from short-circuits profile
// lookup: the profile carries a different email, and the flag's value
// is what reaches the script.
func TestExecProfile_FromFlagWins(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	createProfile(t, home, "noor", "profile@example.com")
	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name,
		"--from", "explicit@example.com",
		"--profile", "noor",
	)
	if err != nil {
		t.Fatalf("exec with --from and --profile: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	assertFromReachedScript(t, stdout, "explicit@example.com")
}

// TestExecProfile_FromFlagWinsOverMissingProfile asserts the --from
// short-circuit happens before the profile is loaded at all: a profile
// id that does not exist never surfaces an error when --from is set.
func TestExecProfile_FromFlagWinsOverMissingProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name,
		"--profile", "does-not-exist",
		"--from", "explicit@example.com",
	)
	if err != nil {
		t.Fatalf("exec with --from and unknown profile: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	assertFromReachedScript(t, stdout, "explicit@example.com")
}

// TestExecProfile_ResolvesFromProfileBeforeSubcommand asserts the root
// global --profile supplies the address when placed ahead of the
// subcommand path.
func TestExecProfile_ResolvesFromProfileBeforeSubcommand(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	createProfile(t, home, "noor", "noor@example.com")
	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"--profile", "noor",
		"adapter", "exec", name, "send",
		"--input", "to=user@example.com",
		"--input", "subject=Hello",
		"--input", "body=Message body",
	)
	if err != nil {
		t.Fatalf("exec with leading --profile: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	assertFromReachedScript(t, stdout, "noor@example.com")
}

// TestExecProfile_ResolvesFromProfileAfterSubcommand asserts the same
// resolution when --profile trails the subcommand. --profile is a
// tool-level global, not an exec-local flag, so cobra must accept it
// in either position.
func TestExecProfile_ResolvesFromProfileAfterSubcommand(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	createProfile(t, home, "noor", "noor@example.com")
	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name, "--profile", "noor")
	if err != nil {
		t.Fatalf("exec with trailing --profile: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	assertFromReachedScript(t, stdout, "noor@example.com")
}

// TestExecProfile_NeitherFlagIsError asserts exec refuses to guess an
// address when neither --from nor --profile is given.
func TestExecProfile_NeitherFlagIsError(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name)
	if err == nil {
		t.Fatalf("expected error with neither --from nor --profile\nstdout: %s",
			stdout)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "--from or --profile is required for exec") {
		t.Errorf("expected required-flag error, got:\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}
	if strings.Contains(combined, "ACTION send") {
		t.Errorf("action ran despite unresolved From address:\n%s", combined)
	}
}

// TestExecProfile_UnloadableProfileIsError asserts a --profile naming a
// profile that cannot be loaded surfaces the load failure, quoting the
// profile id, rather than falling back to an empty address.
func TestExecProfile_UnloadableProfileIsError(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name,
		"--profile", "ghost",
	)
	if err == nil {
		t.Fatalf("expected error for unloadable profile\nstdout: %s", stdout)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, `load profile "ghost"`) {
		t.Errorf("expected load-profile error naming the id, got:\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}
	if strings.Contains(combined, "ACTION send") {
		t.Errorf("action ran despite unloadable profile:\n%s", combined)
	}
}

// TestExecProfile_ProfileWithoutEmailIsError asserts a profile that
// loads but carries no email produces a remediation error naming the
// profile and pointing at both --from and `aps profile create --email`.
func TestExecProfile_ProfileWithoutEmailIsError(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	createProfile(t, home, "blank", "")
	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := execFixtureSend(t, home, name,
		"--profile", "blank",
	)
	if err == nil {
		t.Fatalf("expected error for profile without email\nstdout: %s", stdout)
	}
	combined := stdout + stderr
	for _, want := range []string{
		`profile "blank" has no email`,
		"use --from",
		"aps profile create blank --email",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("expected %q in error output, got:\nstdout: %s\nstderr: %s",
				want, stdout, stderr)
		}
	}
	if strings.Contains(combined, "ACTION send") {
		t.Errorf("action ran despite profile without email:\n%s", combined)
	}
}
