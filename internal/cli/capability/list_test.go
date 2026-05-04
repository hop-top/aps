package capability

import (
	"testing"

	"hop.top/aps/internal/cli/listing"
)

func sampleRows() []capabilitySummaryRow {
	return []capabilitySummaryRow{
		{Name: "a2a", Source: "builtin", Tags: []string{"core"}, Profiles: []string{"alpha"}},
		{Name: "ibr", Source: "builtin", Tags: []string{"core", "ui"}, Profiles: nil},
		{Name: "mytool", Source: "external", Type: "managed", Tags: []string{"ui"}, Profiles: []string{"beta"}},
	}
}

// TestSourcePred_Builtin keeps only builtin rows.
func TestSourcePred_Builtin(t *testing.T) {
	got := listing.Filter(sampleRows(), capabilitySourcePred(true, false))
	if len(got) != 2 {
		t.Fatalf("expected 2 builtins; got %d", len(got))
	}
	for _, r := range got {
		if r.Source != "builtin" {
			t.Fatalf("non-builtin slipped through: %+v", r)
		}
	}
}

// TestSourcePred_External keeps only external rows.
func TestSourcePred_External(t *testing.T) {
	got := listing.Filter(sampleRows(), capabilitySourcePred(false, true))
	if len(got) != 1 || got[0].Name != "mytool" {
		t.Fatalf("expected [mytool]; got %+v", got)
	}
}

// TestSourcePred_None returns nil for match-all.
func TestSourcePred_None(t *testing.T) {
	if capabilitySourcePred(false, false) != nil {
		t.Fatalf("expected nil predicate")
	}
}

// TestTagFilter narrows by tag membership.
func TestTagFilter(t *testing.T) {
	pred := listing.MatchSlice(
		func(r capabilitySummaryRow) []string { return r.Tags }, "ui")
	got := listing.Filter(sampleRows(), pred)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows tagged ui; got %d", len(got))
	}
}

// TestEnabledOnFilter keeps rows whose Profiles contain the value.
func TestEnabledOnFilter(t *testing.T) {
	pred := listing.MatchSlice(
		func(r capabilitySummaryRow) []string { return r.Profiles }, "beta")
	got := listing.Filter(sampleRows(), pred)
	if len(got) != 1 || got[0].Name != "mytool" {
		t.Fatalf("expected [mytool]; got %+v", got)
	}
}

// TestComposed validates listing.All wiring of all four filter shapes.
func TestComposed(t *testing.T) {
	pred := listing.All(
		listing.MatchSlice(func(r capabilitySummaryRow) []string { return r.Tags }, "ui"),
		listing.MatchSlice(func(r capabilitySummaryRow) []string { return r.Profiles }, "beta"),
		capabilitySourcePred(false, true),
	)
	got := listing.Filter(sampleRows(), pred)
	if len(got) != 1 || got[0].Name != "mytool" {
		t.Fatalf("expected [mytool]; got %+v", got)
	}
}

// TestPatternRows shape — patterns differ structurally from
// capabilities (no source/type), so they ride a distinct row type.
func TestPatternRows(t *testing.T) {
	rows := buildPatternRows()
	if len(rows) == 0 {
		t.Fatalf("expected ≥1 pattern row")
	}
	for _, r := range rows {
		if r.Tool == "" || r.DefaultPath == "" {
			t.Fatalf("pattern row missing fields: %+v", r)
		}
	}
}
