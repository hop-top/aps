package cli

import (
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
)

const sampleCardamumJSON = `[
  {"id":"alice","addressbook_id":"default","vcard":"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Alice Wonder\r\nORG:Wonderland Corp\r\nEMAIL;TYPE=WORK:alice@example.com\r\nTEL;TYPE=CELL,VOICE:+15551112222\r\nEND:VCARD\r\n"},
  {"id":"bob","addressbook_id":"default","vcard":"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Bob Builder\r\nORG:Wonderland Corp\r\nEND:VCARD\r\n"},
  {"id":"carol","addressbook_id":"work","vcard":"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Carol\r\nEMAIL:carol@solo.example\r\nEND:VCARD\r\n"}
]`

// TestVcardField_BasicLookup pulls FN/EMAIL/ORG/TEL out of a sample card.
func TestVcardField_BasicLookup(t *testing.T) {
	body := "BEGIN:VCARD\r\nFN:Alice\r\nEMAIL;TYPE=WORK:a@x.test\r\nTEL;TYPE=CELL:+1\r\nEND:VCARD"
	cases := map[string]string{
		"FN":    "Alice",
		"EMAIL": "a@x.test",
		"TEL":   "+1",
		"ORG":   "",
	}
	for k, want := range cases {
		if got := vcardField(body, k); got != want {
			t.Fatalf("vcardField(%q) = %q; want %q", k, got, want)
		}
	}
}

// TestVcardField_LineFolding handles RFC 6350 line-fold continuations.
func TestVcardField_LineFolding(t *testing.T) {
	body := "BEGIN:VCARD\r\nFN:Long\r\n Name\r\nEND:VCARD"
	if got := vcardField(body, "FN"); got != "LongName" {
		t.Fatalf("expected unfolded value; got %q", got)
	}
}

// TestContactRowsFromCardamum parses the array shape and lifts fields.
func TestContactRowsFromCardamum(t *testing.T) {
	rows, err := contactRowsFromCardamum(sampleCardamumJSON)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows; got %d", len(rows))
	}
	idx := func(id string) contactSummaryRow {
		for _, r := range rows {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("row %s not found", id)
		return contactSummaryRow{}
	}
	alice := idx("alice")
	if alice.Name != "Alice Wonder" || alice.Email != "alice@example.com" ||
		alice.Org != "Wonderland Corp" || alice.Phone != "+15551112222" ||
		alice.Addressbook != "default" {
		t.Fatalf("alice row mismatch: %+v", alice)
	}
	bob := idx("bob")
	if bob.Email != "" || bob.Org != "Wonderland Corp" {
		t.Fatalf("bob row mismatch: %+v", bob)
	}
}

// TestContactRowsFromCardamum_Empty handles the "" + "[]" cases.
func TestContactRowsFromCardamum_Empty(t *testing.T) {
	for _, in := range []string{"", "  \n", "[]"} {
		rows, err := contactRowsFromCardamum(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected 0 rows for %q; got %d", in, len(rows))
		}
	}
}

// TestContactRowsFromCardamum_BadJSON surfaces decode errors.
func TestContactRowsFromCardamum_BadJSON(t *testing.T) {
	_, err := contactRowsFromCardamum("not json")
	if err == nil || !strings.Contains(err.Error(), "parse cardamum json") {
		t.Fatalf("expected parse error; got %v", err)
	}
}

// TestContactFilters validates --org and --has-email predicate
// composition against the parsed sample.
func TestContactFilters(t *testing.T) {
	rows, _ := contactRowsFromCardamum(sampleCardamumJSON)

	orgPred := listing.MatchString(
		func(r contactSummaryRow) string { return r.Org },
		"Wonderland Corp")
	got := listing.Filter(rows, orgPred)
	if len(got) != 2 {
		t.Fatalf("--org Wonderland Corp expected 2; got %d", len(got))
	}

	hasEmail := listing.BoolFlag(
		true,
		func(r contactSummaryRow) bool { return r.Email != "" },
		true)
	got = listing.Filter(rows, hasEmail)
	if len(got) != 2 {
		t.Fatalf("--has-email expected 2; got %d", len(got))
	}

	noEmail := listing.BoolFlag(
		true,
		func(r contactSummaryRow) bool { return r.Email != "" },
		false)
	got = listing.Filter(rows, noEmail)
	if len(got) != 1 || got[0].ID != "bob" {
		t.Fatalf("--has-email=false expected [bob]; got %+v", got)
	}

	// Composed: org + has-email.
	combined := listing.All(orgPred, hasEmail)
	got = listing.Filter(rows, combined)
	if len(got) != 1 || got[0].ID != "alice" {
		t.Fatalf("composed expected [alice]; got %+v", got)
	}
}
