package bundle

import (
	"testing"

	"hop.top/aps/internal/cli/listing"
)

// rowsFor builds rows directly without touching the loader so we can
// drive the predicate composition under test in isolation.
func rowsFor() []bundleSummaryRow {
	return []bundleSummaryRow{
		{Name: "alpha", Source: "built-in", Tags: []string{"core", "shared"}},
		{Name: "beta", Source: "user", Tags: []string{"shared"}},
		{Name: "gamma", Source: "user (overrides built-in)", Tags: nil},
	}
}

// TestSourcePredicate_BuiltinOnly keeps only built-in rows.
func TestSourcePredicate_BuiltinOnly(t *testing.T) {
	got := listing.Filter(rowsFor(), sourcePredicate(true, false))
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("expected [alpha]; got %+v", got)
	}
}

// TestSourcePredicate_UserOnly drops the built-in row.
func TestSourcePredicate_UserOnly(t *testing.T) {
	got := listing.Filter(rowsFor(), sourcePredicate(false, true))
	names := []string{}
	for _, r := range got {
		names = append(names, r.Name)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 user rows; got %v", names)
	}
	for _, r := range got {
		if r.Source == "built-in" {
			t.Fatalf("--user returned built-in row: %+v", r)
		}
	}
}

// TestSourcePredicate_Default returns nil so listing.All treats it as
// match-all.
func TestSourcePredicate_Default(t *testing.T) {
	if p := sourcePredicate(false, false); p != nil {
		t.Fatalf("expected nil predicate when neither flag set; got %T", p)
	}
}

// TestTagFilter_MatchSlice validates listing.MatchSlice composition on
// our row type — flag value must appear in row.Tags.
func TestTagFilter_MatchSlice(t *testing.T) {
	pred := listing.MatchSlice(
		func(r bundleSummaryRow) []string { return r.Tags }, "core")
	got := listing.Filter(rowsFor(), pred)
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("expected [alpha]; got %+v", got)
	}
}

// TestTagFilter_EmptyMatchAll proves an unset --tag drops to match-all.
func TestTagFilter_EmptyMatchAll(t *testing.T) {
	pred := listing.MatchSlice(
		func(r bundleSummaryRow) []string { return r.Tags }, "")
	got := listing.Filter(rowsFor(), pred)
	if len(got) != len(rowsFor()) {
		t.Fatalf("empty want should match every row; got %d/%d",
			len(got), len(rowsFor()))
	}
}

// TestComposed combines tag + user-only filters under listing.All.
func TestComposed(t *testing.T) {
	pred := listing.All(
		listing.MatchSlice(
			func(r bundleSummaryRow) []string { return r.Tags }, "shared"),
		sourcePredicate(false, true),
	)
	got := listing.Filter(rowsFor(), pred)
	if len(got) != 1 || got[0].Name != "beta" {
		t.Fatalf("expected [beta]; got %+v", got)
	}
}
