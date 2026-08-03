package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/email/backends/himalaya/read.sh.
//
// read.sh is the simplest email action: one required input (the
// envelope ID) and one conditional flag. The ID is a positional
// argument, so its placement relative to the account flag matters —
// himalaya must see the ID as the message selector, not as a value
// belonging to -a.

// readScript resolves read.sh from the adapters tree.
func readScript(t *testing.T) backendScript {
	t.Helper()
	return backendScript{Adapter: "email", Backend: "himalaya", Action: "read"}
}

// TestHimalayaRead_NoAccount is the baseline: ID positional, no -a.
func TestHimalayaRead_NoAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{
		"EMAIL_ID": "7131",
	})
	if err != nil {
		t.Fatalf("read.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "message", "read", "7131")
}

// TestHimalayaRead_WithAccount covers the account flag trailing the
// positional ID.
func TestHimalayaRead_WithAccount(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{
		"EMAIL_ID":          "7131",
		"APS_EMAIL_ACCOUNT": "work",
	})
	if err != nil {
		t.Fatalf("read.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "message", "read", "7131", "-a", "work")
}

// TestHimalayaRead_EmptyAccountOmitsFlag pins set-but-empty as "no
// account" rather than an account named "".
func TestHimalayaRead_EmptyAccountOmitsFlag(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{
		"EMAIL_ID":          "7131",
		"APS_EMAIL_ACCOUNT": "",
	})
	if err != nil {
		t.Fatalf("read.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "message", "read", "7131")
}

// TestHimalayaRead_MissingID covers the ${EMAIL_ID:?} guard: the
// backend must not be invoked at all when the ID is absent, since
// `himalaya message read` with no selector would act on whatever it
// considers current.
func TestHimalayaRead_MissingID(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{})
	if err == nil {
		t.Fatalf("expected failure with EMAIL_ID unset, got success\nargv: %s",
			inv.argvString())
	}
	if len(inv.Argv) != 0 {
		t.Errorf("backend invoked despite missing EMAIL_ID: %s", inv.argvString())
	}
	if !strings.Contains(out, "EMAIL_ID") {
		t.Errorf("diagnostic does not name EMAIL_ID:\n%s", out)
	}
}

// TestHimalayaRead_AccountWithSpace is the word-splitting case for
// read.sh, matching the send.sh bug: an account name containing a
// space must stay one argv element.
func TestHimalayaRead_AccountWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{
		"EMAIL_ID":          "7131",
		"APS_EMAIL_ACCOUNT": "work account",
	})
	if err != nil {
		t.Fatalf("read.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "message", "read", "7131", "-a", "work account")
}

// TestHimalayaRead_IDWithSpace guards the positional argument itself.
// "$ID" is already quoted in the script; this pins that, so a future
// edit dropping the quotes fails here rather than in production.
func TestHimalayaRead_IDWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	inv, out, err := runBackendScript(t, readScript(t), map[string]string{
		"EMAIL_ID": "7131 7132",
	})
	if err != nil {
		t.Fatalf("read.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, inv, "message", "read", "7131 7132")
}
