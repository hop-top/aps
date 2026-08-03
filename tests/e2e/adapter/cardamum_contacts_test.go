package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/contacts/backends/cardamum/*.sh.
//
// Every cardamum action shares the same shape: resolve an optional
// account into a flag, resolve the addressbook through a two-level
// default (CONTACT_ADDRESSBOOK, then CONTACTS_ADDRESSBOOK, then
// "default"), then call `cardamum cards <verb>`. The account flag and
// the addressbook fallback are therefore worth asserting once per
// action rather than once overall — a per-script regression in either
// is invisible from any other script's tests.
//
// add/update/note additionally build or rewrite a vCard, so those get
// stdin assertions on top of argv.

// cardamumScript names a cardamum action.
func cardamumScript(action string) backendScript {
	return backendScript{Adapter: "contacts", Backend: "cardamum", Action: action}
}

// cardsStub scripts stdout for the `cards read` call that update.sh
// and note.sh consume before writing back.
func cardsStub(vcard string) stubOptions {
	return stubOptions{
		Responses: []stubResponse{
			{Match: []string{"cards", "read"}, Stdout: vcard},
		},
		ReadsStdin: [][]string{
			{"cards", "create"},
			{"cards", "update"},
		},
	}
}

// sampleVCard is a representative stored card for read-modify-write
// actions.
const sampleVCard = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Ada Lovelace\n" +
	"EMAIL:ada@example.com\n" +
	"END:VCARD\n"

// TestCardamumAddressbookDefault covers the two-level fallback across
// every action that takes an addressbook: unset means "default".
func TestCardamumAddressbookDefault(t *testing.T) {
	requireBash(t)
	t.Parallel()

	cases := []struct {
		action   string
		env      map[string]string
		wantBook string // argv position varies, so assert by presence
	}{
		{action: "list", env: map[string]string{}},
		{action: "show", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "find", env: map[string]string{"CONTACT_QUERY": "ada"}},
		{action: "delete", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "add", env: map[string]string{"CONTACT_EMAIL": "a@b.c"}},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()

			invs, out, err := runBackendScriptMulti(t, cardamumScript(tc.action),
				tc.env, cardsStub(sampleVCard))
			if err != nil {
				t.Fatalf("%s.sh failed: %v\noutput: %s", tc.action, err, out)
			}
			if len(invs) == 0 {
				t.Fatalf("%s.sh did not invoke the backend\noutput: %s", tc.action, out)
			}
			if !argvContains(invs[0], "default") {
				t.Errorf("%s.sh did not fall back to addressbook 'default': %s",
					tc.action, invs[0].argvString())
			}
		})
	}
}

// TestCardamumAddressbookOverridePrecedence pins CONTACT_ADDRESSBOOK
// winning over CONTACTS_ADDRESSBOOK. The per-action input must beat
// the profile-level default, not the other way round.
func TestCardamumAddressbookOverridePrecedence(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("list"), map[string]string{
		"CONTACTS_ADDRESSBOOK": "profile-book",
		"CONTACT_ADDRESSBOOK":  "action-book",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "cards", "list", "action-book", "--json")
}

// TestCardamumProfileAddressbook covers CONTACTS_ADDRESSBOOK applying
// when the per-action override is absent.
func TestCardamumProfileAddressbook(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("list"), map[string]string{
		"CONTACTS_ADDRESSBOOK": "profile-book",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("list.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "cards", "list", "profile-book", "--json")
}

// TestCardamumAccountFlag covers the -a flag across every action.
// Each script builds ACCT_FLAG independently, so each is asserted.
func TestCardamumAccountFlag(t *testing.T) {
	requireBash(t)
	t.Parallel()

	cases := []struct {
		action string
		env    map[string]string
	}{
		{action: "list", env: map[string]string{}},
		{action: "show", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "find", env: map[string]string{"CONTACT_QUERY": "ada"}},
		{action: "delete", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "add", env: map[string]string{"CONTACT_EMAIL": "a@b.c"}},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()

			env := map[string]string{"CONTACTS_ACCOUNT": "work"}
			for k, v := range tc.env {
				env[k] = v
			}

			invs, out, err := runBackendScriptMulti(t, cardamumScript(tc.action),
				env, cardsStub(sampleVCard))
			if err != nil {
				t.Fatalf("%s.sh failed: %v\noutput: %s", tc.action, err, out)
			}
			if len(invs) == 0 {
				t.Fatalf("%s.sh did not invoke the backend\noutput: %s", tc.action, out)
			}

			if !argvHasPair(invs[0], "-a", "work") {
				t.Errorf("%s.sh did not pass -a work: %s",
					tc.action, invs[0].argvString())
			}
		})
	}
}

// TestCardamumAccountWithSpace is the word-splitting case, applied to
// every action. cardamum account names are user-chosen labels from its
// config, so spaces are permitted.
func TestCardamumAccountWithSpace(t *testing.T) {
	requireBash(t)
	t.Parallel()

	cases := []struct {
		action string
		env    map[string]string
	}{
		{action: "list", env: map[string]string{}},
		{action: "show", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "find", env: map[string]string{"CONTACT_QUERY": "ada"}},
		{action: "delete", env: map[string]string{"CONTACT_ID": "c1"}},
		{action: "add", env: map[string]string{"CONTACT_EMAIL": "a@b.c"}},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()

			env := map[string]string{"CONTACTS_ACCOUNT": "work account"}
			for k, v := range tc.env {
				env[k] = v
			}

			invs, out, err := runBackendScriptMulti(t, cardamumScript(tc.action),
				env, cardsStub(sampleVCard))
			if err != nil {
				t.Fatalf("%s.sh failed: %v\noutput: %s", tc.action, err, out)
			}
			if len(invs) == 0 {
				t.Fatalf("%s.sh did not invoke the backend\noutput: %s", tc.action, out)
			}

			if !argvHasPair(invs[0], "-a", "work account") {
				t.Errorf("%s.sh word-split the account name: %s",
					tc.action, invs[0].argvString())
			}
		})
	}
}

// TestCardamumAdd_VCardMinimal covers the vCard built from the
// required field alone. CONTACT_NAME defaults to the email, so FN and
// EMAIL both carry it.
func TestCardamumAdd_VCardMinimal(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("add"), map[string]string{
		"CONTACT_EMAIL": "ada@example.com",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("add.sh failed: %v\noutput: %s", err, out)
	}

	assertStdinLines(t, invs[0], []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"FN:ada@example.com",
		"EMAIL:ada@example.com",
		"END:VCARD",
	})
}

// TestCardamumAdd_VCardAllFields covers every optional field being
// appended in order, with END:VCARD staying last.
func TestCardamumAdd_VCardAllFields(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("add"), map[string]string{
		"CONTACT_EMAIL": "ada@example.com",
		"CONTACT_NAME":  "Ada Lovelace",
		"CONTACT_ORG":   "Analytical Engines Ltd",
		"CONTACT_PHONE": "+1-555-0100",
		"CONTACT_NOTE":  "Met at conference",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("add.sh failed: %v\noutput: %s", err, out)
	}

	assertStdinLines(t, invs[0], []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"FN:Ada Lovelace",
		"EMAIL:ada@example.com",
		"ORG:Analytical Engines Ltd",
		"TEL:+1-555-0100",
		"NOTE:Met at conference",
		"END:VCARD",
	})
}

// TestCardamumAdd_MissingEmail covers the ${CONTACT_EMAIL:?} guard.
func TestCardamumAdd_MissingEmail(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("add"),
		map[string]string{}, cardsStub(sampleVCard))
	if err == nil {
		t.Fatalf("expected failure with CONTACT_EMAIL unset, got success:\n%s",
			invocationsString(invs))
	}
	if len(invs) != 0 {
		t.Errorf("backend invoked despite missing CONTACT_EMAIL:\n%s",
			invocationsString(invs))
	}
	if !strings.Contains(out, "CONTACT_EMAIL") {
		t.Errorf("diagnostic does not name CONTACT_EMAIL:\n%s", out)
	}
}

// TestCardamumUpdate_ReadModifyWrite covers the two-call flow: read
// the stored card, rewrite one field with sed, write it back.
func TestCardamumUpdate_ReadModifyWrite(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("update"), map[string]string{
		"CONTACT_ID":   "c1",
		"CONTACT_NAME": "Ada King",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("update.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[0], "cards", "read", "default", "c1")
	assertArgv(t, invs[1], "cards", "update", "default", "c1")

	assertStdinLines(t, invs[1], []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"FN:Ada King",
		"EMAIL:ada@example.com",
		"END:VCARD",
	})
}

// TestCardamumUpdate_SlashInValue is the sed-injection case.
//
// update.sh substitutes with s/^FN:.*/FN:$CONTACT_NAME/ — an
// unescaped value. A field containing "/" closes the replacement
// early and the rest is parsed as sed flags, so the command either
// errors or writes a truncated value.
//
// Slashes are ordinary in these fields: job titles ("Eng/Ops"),
// street addresses, and URLs in notes all carry them.
func TestCardamumUpdate_SlashInValue(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("update"), map[string]string{
		"CONTACT_ID":  "c1",
		"CONTACT_ORG": "Research/Development",
	}, cardsStub("BEGIN:VCARD\nVERSION:3.0\nFN:Ada\nORG:Old\nEND:VCARD\n"))
	if err != nil {
		t.Fatalf("update.sh failed on a value containing '/': %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	if !containsLine(invs[1], "ORG:Research/Development") {
		t.Errorf("slash in ORG was not written verbatim:\n%s", invs[1].Stdin)
	}
}

// TestCardamumUpdate_AmpersandInValue is the other sed metacharacter
// case: unescaped "&" in a replacement expands to the whole matched
// text, so "R&D" would render as "R<entire ORG line>D".
func TestCardamumUpdate_AmpersandInValue(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("update"), map[string]string{
		"CONTACT_ID":  "c1",
		"CONTACT_ORG": "R&D",
	}, cardsStub("BEGIN:VCARD\nVERSION:3.0\nFN:Ada\nORG:Old\nEND:VCARD\n"))
	if err != nil {
		t.Fatalf("update.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	if !containsLine(invs[1], "ORG:R&D") {
		t.Errorf("ampersand in ORG was expanded by sed rather than written literally:\n%s",
			invs[1].Stdin)
	}
}

// TestCardamumNote_AppendsToExisting covers note.sh's append branch:
// an existing NOTE line gains " | [timestamp] text" rather than being
// replaced.
func TestCardamumNote_AppendsToExisting(t *testing.T) {
	requireBash(t)
	t.Parallel()

	stored := "BEGIN:VCARD\nVERSION:3.0\nFN:Ada\nNOTE:first note\nEND:VCARD\n"

	invs, out, err := runBackendScriptMulti(t, cardamumScript("note"), map[string]string{
		"CONTACT_ID":   "c1",
		"CONTACT_TEXT": "second note",
	}, cardsStub(stored))
	if err != nil {
		t.Fatalf("note.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	var noteLine string
	for _, line := range invs[1].stdinLines() {
		if strings.HasPrefix(line, "NOTE:") {
			noteLine = line
		}
	}
	if !strings.Contains(noteLine, "first note") {
		t.Errorf("append dropped the existing note: %q", noteLine)
	}
	if !strings.Contains(noteLine, "second note") {
		t.Errorf("append did not add the new text: %q", noteLine)
	}
}

// TestCardamumNote_CreatesWhenAbsent covers the other branch: a card
// with no NOTE line gains one before END:VCARD.
func TestCardamumNote_CreatesWhenAbsent(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, cardamumScript("note"), map[string]string{
		"CONTACT_ID":   "c1",
		"CONTACT_TEXT": "first note",
	}, cardsStub(sampleVCard))
	if err != nil {
		t.Fatalf("note.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d:\n%s", len(invs), invocationsString(invs))
	}

	lines := invs[1].stdinLines()
	if len(lines) == 0 {
		t.Fatalf("note.sh wrote an empty card")
	}
	if got := lines[len(lines)-1]; got != "END:VCARD" {
		t.Errorf("END:VCARD is not last after inserting a note; got %q\n%s",
			got, invs[1].Stdin)
	}

	var found bool
	for _, line := range lines {
		if strings.HasPrefix(line, "NOTE:") && strings.Contains(line, "first note") {
			found = true
		}
	}
	if !found {
		t.Errorf("no NOTE line was inserted:\n%s", invs[1].Stdin)
	}
}

// TestCardamumFind_PassesQueryToGrep covers find.sh's grep stage: the
// query filters the backend's output rather than reaching argv.
func TestCardamumFind_PassesQueryToGrep(t *testing.T) {
	requireBash(t)
	t.Parallel()

	listing := "ada@example.com\nbob@example.com\ncarol@example.com\n"

	invs, out, err := runBackendScriptMulti(t, cardamumScript("find"), map[string]string{
		"CONTACT_QUERY": "bob",
	}, stubOptions{
		Responses: []stubResponse{
			{Match: []string{"cards", "list"}, Stdout: listing},
		},
	})
	if err != nil {
		t.Fatalf("find.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "cards", "list", "default", "--json")

	if !strings.Contains(out, "bob@example.com") {
		t.Errorf("matching contact missing from output:\n%s", out)
	}
	if strings.Contains(out, "ada@example.com") {
		t.Errorf("non-matching contact leaked into output:\n%s", out)
	}
}

// TestCardamumFind_NoMatchIsNotAnError covers grep's exit-1-on-no-match
// under `set -e`.
//
// An empty search result is an ordinary outcome, not a failure. If the
// script propagates grep's 1, the caller cannot distinguish "no
// contacts matched" from "the backend broke".
func TestCardamumFind_NoMatchIsNotAnError(t *testing.T) {
	requireBash(t)
	t.Parallel()

	listing := "ada@example.com\nbob@example.com\n"

	_, out, err := runBackendScriptMulti(t, cardamumScript("find"), map[string]string{
		"CONTACT_QUERY": "nobody",
	}, stubOptions{
		Responses: []stubResponse{
			{Match: []string{"cards", "list"}, Stdout: listing},
		},
	})
	if err != nil {
		t.Fatalf("find.sh treated an empty search result as a failure: %v\noutput: %s",
			err, out)
	}
}

// argvContains reports whether any argv element equals want.
func argvContains(inv backendInvocation, want string) bool {
	for _, a := range inv.Argv {
		if a == want {
			return true
		}
	}
	return false
}

// argvHasPair reports whether flag appears immediately followed by
// value — the shape a correctly-quoted two-element flag takes.
func argvHasPair(inv backendInvocation, flag, value string) bool {
	for i := 0; i+1 < len(inv.Argv); i++ {
		if inv.Argv[i] == flag && inv.Argv[i+1] == value {
			return true
		}
	}
	return false
}
