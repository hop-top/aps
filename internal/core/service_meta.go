package core

import "sort"

// This file carries the doc-facing judgment metadata for the service alias
// and adapter catalogues. Structural facts (alias -> canonical type and
// adapter, the known message adapter set) live in service.go; only the
// human judgment strings the generated doc tables render live here, so the
// docs regenerate from code instead of drifting by hand (see
// internal/tools/servicemd and `make docs-gen`).

// TicketAdapterDocMeta is the judgment row rendered for one ticket adapter
// alias in the docs/user/tickets.md adapter table.
type TicketAdapterDocMeta struct {
	// Inbound summarizes the payload the mounted route accepts.
	Inbound string
	// Maturity is the service maturity verdict for the adapter.
	Maturity string
}

//nolint:goconst // adapter catalogue; literals are the data
var ticketAdapterDocMeta = map[string]TicketAdapterDocMeta{
	"email": {
		Inbound:  "flat email JSON from a relay or poller",
		Maturity: "ready",
	},
	"github": {
		Inbound:  "route mounted; payloads rejected until a normalizer lands",
		Maturity: "component",
	},
	"gitlab": {
		Inbound:  "GitLab issue/MR/note webhook JSON",
		Maturity: "ready",
	},
	"jira": {
		Inbound:  "Jira issue/comment webhook JSON",
		Maturity: "ready",
	},
	"linear": {
		Inbound:  "Linear issue/comment webhook JSON",
		Maturity: "ready",
	},
}

// ticketAliasDocNotes holds the note appended to a ticket alias row in the
// quick-ref alias table. Aliases that deliberately carry no note are listed
// in ticketAliasNoDocNote.
var ticketAliasDocNotes = map[string]string{
	"email": "(mounted at `/services/<id>/ticket/email`; see [tickets](user/tickets.md))",
}

// ticketAliasNoDocNote is the explicit allowlist of ticket aliases that
// deliberately carry no quick-ref note.
var ticketAliasNoDocNote = map[string]bool{
	"github": true,
	"gitlab": true,
	"jira":   true,
	"linear": true,
}

// messageAdapterDocNotes holds the note appended to a message adapter row
// in the quick-ref alias table. Adapters that deliberately carry no note
// are listed in messageAdapterNoDocNote.
var messageAdapterDocNotes = map[string]string{
	"email": "-- pass `--type message --adapter email`",
}

// messageAdapterNoDocNote is the explicit allowlist of message adapters
// that deliberately carry no quick-ref note.
//
//nolint:goconst // adapter catalogue; literals are the data
var messageAdapterNoDocNote = map[string]bool{
	"discord":  true,
	"slack":    true,
	"sms":      true,
	"teams":    true,
	"telegram": true,
	"whatsapp": true,
}

// MessageAdapterCatalogue returns the known message adapter names in
// alphabetical order.
func MessageAdapterCatalogue() []string {
	out := make([]string, 0, len(knownMessageAdapters))
	for name := range knownMessageAdapters {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// TicketAdapterDocMetaFor returns the doc judgment row for a ticket adapter
// alias.
func TicketAdapterDocMetaFor(alias string) (TicketAdapterDocMeta, bool) {
	meta, ok := ticketAdapterDocMeta[alias]
	return meta, ok
}

// TicketAliasDocNote returns the note appended to a ticket alias row in the
// quick-ref alias table; empty when the alias carries none.
func TicketAliasDocNote(alias string) string {
	return ticketAliasDocNotes[alias]
}

// MessageAdapterDocNote returns the note appended to a message adapter row
// in the quick-ref alias table; empty when the adapter carries none.
func MessageAdapterDocNote(adapter string) string {
	return messageAdapterDocNotes[adapter]
}
