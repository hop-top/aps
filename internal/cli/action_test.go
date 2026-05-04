package cli

import (
	"testing"

	"hop.top/aps/internal/cli/listing"
)

// TestActionTypeFilter_Match — --type sh keeps only sh rows.
func TestActionTypeFilter_Match(t *testing.T) {
	rows := []actionSummaryRow{
		{ID: "deploy", Type: "sh"},
		{ID: "report", Type: "py"},
		{ID: "build", Type: "sh"},
		{ID: "lint", Type: "js"},
	}
	pred := listing.MatchString(func(r actionSummaryRow) string { return r.Type }, "sh")
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	for _, r := range got {
		if r.Type != "sh" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

// TestActionTypeFilter_Empty — unset --type returns every row.
func TestActionTypeFilter_Empty(t *testing.T) {
	rows := []actionSummaryRow{
		{ID: "a", Type: "sh"},
		{ID: "b", Type: "py"},
	}
	pred := listing.All(
		listing.MatchString(func(r actionSummaryRow) string { return r.Type }, ""),
	)
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("empty --type should match all; got len=%d", len(got))
	}
}

// TestActionListCmd_HasTypeFlag asserts the cobra wiring registers
// --type so callers can rely on it without inspecting source.
func TestActionListCmd_HasTypeFlag(t *testing.T) {
	if f := actionListCmd.Flags().Lookup("type"); f == nil {
		t.Fatal("aps action list should have --type flag")
	}
}
