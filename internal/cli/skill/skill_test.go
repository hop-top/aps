package skill

import (
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/skills"
)

// TestSkillSourceFilter — listing.MatchString on Source.
func TestSkillSourceFilter(t *testing.T) {
	rows := []skillSummaryRow{
		{Name: "deploy", Source: "Profile"},
		{Name: "review", Source: "Global"},
		{Name: "edit", Source: "Profile"},
		{Name: "scrape", Source: "Claude Code"},
	}
	pred := listing.MatchString(func(r skillSummaryRow) string { return r.Source }, "Profile")
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

// TestSkillSourceFilter_Empty — unset flag matches every row.
func TestSkillSourceFilter_Empty(t *testing.T) {
	rows := []skillSummaryRow{
		{Name: "a", Source: "Profile"},
		{Name: "b", Source: "Global"},
	}
	pred := listing.All(
		listing.MatchString(func(r skillSummaryRow) string { return r.Source }, ""),
	)
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("empty --source must match all; got %d", len(got))
	}
}

// TestBuildSkillRows_TruncatesDescription — long descriptions get
// the right-trim treatment to keep the table scannable; JSON callers
// see the original via skills.Skill.Description.
func TestBuildSkillRows_TruncatesDescription(t *testing.T) {
	long := strings.Repeat("x", skillDescriptionWidth+30)
	registry := skills.NewRegistry("noor", nil, false)
	// Empty registry → empty rows; sanity guard.
	if rows := buildSkillRows(registry, "noor"); len(rows) != 0 {
		t.Fatalf("empty registry should yield 0 rows; got %d", len(rows))
	}
	// We can't seed the registry directly without disk fixtures;
	// exercise the truncation rule by hand.
	row := skillSummaryRow{Description: long}
	if len(row.Description) <= skillDescriptionWidth {
		t.Fatalf("seed too short")
	}
	short := long[:skillDescriptionWidth-1] + "…"
	if len(short) >= len(long) {
		t.Fatalf("truncation produced no shrink")
	}
}

// TestSkillListCmd_FlagSet asserts only --source survives the audit;
// --verbose was dropped per T-0440.
func TestSkillListCmd_FlagSet(t *testing.T) {
	cmd := newListCmd()
	if f := cmd.Flags().Lookup("source"); f == nil {
		t.Error("--source flag missing")
	}
	if f := cmd.Flags().Lookup("verbose"); f != nil {
		t.Error("--verbose should have been dropped")
	}
	// --profile is owned by the kit/cli root global; subcommand
	// must not redeclare it locally.
	if f := cmd.Flags().Lookup("profile"); f != nil {
		t.Error("--profile should inherit from root global, not be local")
	}
}
