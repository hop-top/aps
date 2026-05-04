package listing

import (
	"testing"
)

type row struct {
	id    string
	tags  []string
	flag  bool
}

func id(r row) string         { return r.id }
func tags(r row) []string     { return r.tags }
func flag(r row) bool         { return r.flag }

func TestAll_NoArgs_MatchesEverything(t *testing.T) {
	p := All[row]()
	if !p(row{}) {
		t.Fatal("All() with no args should match every row")
	}
}

func TestAll_NilPredicates_MatchEverything(t *testing.T) {
	p := All[row](nil, nil)
	if !p(row{id: "x"}) {
		t.Fatal("All with all-nil predicates should match")
	}
}

func TestAll_AllMatch(t *testing.T) {
	p := All(
		MatchString(id, "noor"),
		MatchSlice(tags, "ops"),
	)
	if !p(row{id: "noor", tags: []string{"ops", "lead"}}) {
		t.Fatal("expected match: id=noor + tag=ops")
	}
}

func TestAll_OneMismatch(t *testing.T) {
	p := All(
		MatchString(id, "noor"),
		MatchSlice(tags, "missing"),
	)
	if p(row{id: "noor", tags: []string{"ops"}}) {
		t.Fatal("expected mismatch: tag missing")
	}
}

func TestAny_OneMatch(t *testing.T) {
	p := Any(
		MatchString(id, "absent"),
		MatchString(id, "noor"),
	)
	if !p(row{id: "noor"}) {
		t.Fatal("Any: expected match on second predicate")
	}
}

func TestAny_NoMatch(t *testing.T) {
	p := Any(
		MatchString(id, "absent"),
		MatchString(id, "missing"),
	)
	if p(row{id: "noor"}) {
		t.Fatal("Any: expected no match")
	}
}

func TestAny_NoArgs_MatchesNothing(t *testing.T) {
	p := Any[row]()
	if p(row{}) {
		t.Fatal("Any() with no args should match no row")
	}
}

func TestNot_InvertsMatch(t *testing.T) {
	p := Not(MatchString(id, "noor"))
	if p(row{id: "noor"}) {
		t.Fatal("Not should invert match")
	}
	if !p(row{id: "other"}) {
		t.Fatal("Not should match when inner does not")
	}
}

func TestNot_NilInner_MatchesNothing(t *testing.T) {
	// A nil predicate is "match-all" inside All; under Not it
	// inverts to match-none. Documented behavior.
	p := Not[row](nil)
	if p(row{}) {
		t.Fatal("Not(nil) should never match")
	}
}

func TestMatchString_EmptyWant_IsNil(t *testing.T) {
	p := MatchString(id, "")
	if p != nil {
		t.Fatal("MatchString with empty want should return nil (match-all)")
	}
}

func TestMatchSlice_EmptyWant_IsNil(t *testing.T) {
	p := MatchSlice(tags, "")
	if p != nil {
		t.Fatal("MatchSlice with empty want should return nil")
	}
}

func TestMatchSlice_Found(t *testing.T) {
	p := MatchSlice(tags, "ops")
	if !p(row{tags: []string{"ops", "lead"}}) {
		t.Fatal("expected match")
	}
	if p(row{tags: []string{"finance"}}) {
		t.Fatal("expected miss")
	}
}

func TestBoolFlag_NotChanged_IsNil(t *testing.T) {
	p := BoolFlag(false, flag, true)
	if p != nil {
		t.Fatal("BoolFlag with changed=false should return nil")
	}
}

func TestBoolFlag_Changed_True(t *testing.T) {
	p := BoolFlag(true, flag, true)
	if !p(row{flag: true}) {
		t.Fatal("expected match for flag=true want=true")
	}
	if p(row{flag: false}) {
		t.Fatal("expected miss for flag=false want=true")
	}
}

func TestBoolFlag_Changed_False(t *testing.T) {
	p := BoolFlag(true, flag, false)
	if !p(row{flag: false}) {
		t.Fatal("expected match for flag=false want=false")
	}
	if p(row{flag: true}) {
		t.Fatal("expected miss for flag=true want=false")
	}
}

func TestFilter_NilPredicate_ReturnsAll(t *testing.T) {
	rows := []row{{id: "a"}, {id: "b"}}
	got := Filter[row](rows, nil)
	if len(got) != 2 {
		t.Fatalf("nil predicate: expected all rows, got %d", len(got))
	}
}

func TestFilter_AppliesPredicate(t *testing.T) {
	rows := []row{{id: "a"}, {id: "b"}, {id: "a"}}
	got := Filter(rows, MatchString(id, "a"))
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
}

func TestFilter_EmptyInput_EmptyOutput(t *testing.T) {
	got := Filter[row](nil, MatchString(id, "anything"))
	if len(got) != 0 {
		t.Fatalf("expected empty output, got %d", len(got))
	}
}
