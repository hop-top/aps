package core

import (
	"strings"
	"testing"
)

// aliasAdaptersByType derives alias -> adapter pairs from serviceTypeAliases
// for one canonical type, so the pins below track the live alias map.
func aliasAdaptersByType(t *testing.T, canonicalType string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for alias, target := range serviceTypeAliases {
		typ, adapter, ok := strings.Cut(target, " ")
		if !ok {
			t.Fatalf("alias %q target %q is not %q-shaped", alias, target, "type adapter")
		}
		if typ != canonicalType {
			continue
		}
		out[alias] = adapter
	}
	return out
}

func TestTicketAdapterDocMetaPinnedToTicketAliases(t *testing.T) {
	ticketAliases := aliasAdaptersByType(t, "ticket")
	if len(ticketAliases) == 0 {
		t.Fatal("no ticket aliases derived from serviceTypeAliases")
	}

	for alias := range ticketAliases {
		meta, ok := ticketAdapterDocMeta[alias]
		if !ok {
			t.Errorf("ticket alias %q has no ticketAdapterDocMeta row", alias)
			continue
		}
		if strings.TrimSpace(meta.Inbound) == "" {
			t.Errorf("ticket alias %q has empty Inbound judgment", alias)
		}
		if strings.TrimSpace(meta.Maturity) == "" {
			t.Errorf("ticket alias %q has empty Maturity judgment", alias)
		}
	}

	for alias := range ticketAdapterDocMeta {
		if _, ok := ticketAliases[alias]; !ok {
			t.Errorf("stale ticketAdapterDocMeta row %q: not a ticket alias", alias)
		}
	}
}

func TestTicketAliasDocNotesPartitionTicketAliases(t *testing.T) {
	ticketAliases := aliasAdaptersByType(t, "ticket")

	for alias := range ticketAliases {
		_, noted := ticketAliasDocNotes[alias]
		allowed := ticketAliasNoDocNote[alias]
		if noted == allowed {
			t.Errorf("ticket alias %q must be in exactly one of ticketAliasDocNotes / ticketAliasNoDocNote (noted=%v, allowlisted=%v)", alias, noted, allowed)
		}
	}

	for alias, note := range ticketAliasDocNotes {
		if _, ok := ticketAliases[alias]; !ok {
			t.Errorf("stale ticketAliasDocNotes row %q: not a ticket alias", alias)
		}
		if strings.TrimSpace(note) == "" {
			t.Errorf("ticket alias %q has empty doc note; move it to ticketAliasNoDocNote", alias)
		}
	}
	for alias := range ticketAliasNoDocNote {
		if _, ok := ticketAliases[alias]; !ok {
			t.Errorf("stale ticketAliasNoDocNote row %q: not a ticket alias", alias)
		}
	}
}

func TestMessageAdapterDocNotesPartitionAdapterCatalogue(t *testing.T) {
	if len(knownMessageAdapters) == 0 {
		t.Fatal("knownMessageAdapters is empty")
	}

	for adapter := range knownMessageAdapters {
		_, noted := messageAdapterDocNotes[adapter]
		allowed := messageAdapterNoDocNote[adapter]
		if noted == allowed {
			t.Errorf("message adapter %q must be in exactly one of messageAdapterDocNotes / messageAdapterNoDocNote (noted=%v, allowlisted=%v)", adapter, noted, allowed)
		}
	}

	for adapter, note := range messageAdapterDocNotes {
		if !knownMessageAdapters[adapter] {
			t.Errorf("stale messageAdapterDocNotes row %q: not a known message adapter", adapter)
		}
		if strings.TrimSpace(note) == "" {
			t.Errorf("message adapter %q has empty doc note; move it to messageAdapterNoDocNote", adapter)
		}
	}
	for adapter := range messageAdapterNoDocNote {
		if !knownMessageAdapters[adapter] {
			t.Errorf("stale messageAdapterNoDocNote row %q: not a known message adapter", adapter)
		}
	}
}

func TestMessageAliasAdaptersAreInCatalogue(t *testing.T) {
	messageAliases := aliasAdaptersByType(t, "message")
	if len(messageAliases) == 0 {
		t.Fatal("no message aliases derived from serviceTypeAliases")
	}
	for alias, adapter := range messageAliases {
		if !knownMessageAdapters[adapter] {
			t.Errorf("message alias %q resolves adapter %q not present in knownMessageAdapters", alias, adapter)
		}
	}
}

func TestMessageAdapterCatalogueSortedAndComplete(t *testing.T) {
	catalogue := MessageAdapterCatalogue()
	if len(catalogue) != len(knownMessageAdapters) {
		t.Fatalf("catalogue has %d entries, knownMessageAdapters has %d", len(catalogue), len(knownMessageAdapters))
	}
	for i, adapter := range catalogue {
		if !knownMessageAdapters[adapter] {
			t.Errorf("catalogue entry %q not in knownMessageAdapters", adapter)
		}
		if i > 0 && catalogue[i-1] >= adapter {
			t.Errorf("catalogue not strictly sorted: %q before %q", catalogue[i-1], adapter)
		}
	}
}
