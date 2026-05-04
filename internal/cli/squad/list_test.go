package squad

import (
	"bytes"
	"strings"
	"testing"

	coresquad "hop.top/aps/internal/core/squad"
)

// resetManager swaps defaultManager for an empty one for the duration
// of the test. Squads are in-memory only so we share state with sibling
// commands; this keeps tests hermetic.
func resetManager(t *testing.T) {
	t.Helper()
	prev := defaultManager
	defaultManager = coresquad.NewManager()
	t.Cleanup(func() { defaultManager = prev })
}

func mustCreate(t *testing.T, s coresquad.Squad) {
	t.Helper()
	if err := defaultManager.Create(s); err != nil {
		t.Fatalf("create squad %s: %v", s.ID, err)
	}
}

func runListWith(t *testing.T, args ...string) string {
	t.Helper()
	cmd := newListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	// Capture stdout via hijacking os.Stdout would be invasive; the
	// list command writes directly to os.Stdout via listing.RenderList,
	// so we instead call the predicate path through Execute and
	// inspect via a sibling helper. For the listing path, redirect
	// stdout via a pipe.
	old, w, restore := captureStdout(t)
	defer restore()
	_ = old
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	w.Close()
	return readPipe(t)
}

func TestSquadList_FilterMember(t *testing.T) {
	resetManager(t)
	mustCreate(t, coresquad.Squad{
		ID: "core", Name: "Core", Type: coresquad.SquadTypeStream,
		Domain: "platform", Members: []string{"alice", "bob"},
	})
	mustCreate(t, coresquad.Squad{
		ID: "ops", Name: "Ops", Type: coresquad.SquadTypePlatform,
		Domain: "infra", Members: []string{"carol"},
	})

	out := runListWith(t, "--member", "alice")
	if !strings.Contains(out, "core") {
		t.Errorf("expected core (member=alice): %s", out)
	}
	if strings.Contains(out, "ops") {
		t.Errorf("did not expect ops (no alice): %s", out)
	}

	// Zero-match
	out = runListWith(t, "--member", "no-such")
	for _, id := range []string{"core", "ops"} {
		if strings.Contains(out, id) {
			t.Errorf("expected no rows for bogus member, found %q in: %s", id, out)
		}
	}
}

func TestSquadList_FilterRole(t *testing.T) {
	resetManager(t)
	mustCreate(t, coresquad.Squad{
		ID: "core", Name: "Core", Type: coresquad.SquadTypeStream,
		Domain: "platform", Members: []string{"alice"},
	})
	mustCreate(t, coresquad.Squad{
		ID: "ops", Name: "Ops", Type: coresquad.SquadTypePlatform,
		Domain: "infra", Members: []string{"carol"},
	})

	out := runListWith(t, "--role", "stream-aligned")
	if !strings.Contains(out, "core") {
		t.Errorf("expected core (type=stream-aligned): %s", out)
	}
	if strings.Contains(out, "ops") {
		t.Errorf("did not expect ops (type=platform): %s", out)
	}
}

func TestSquadList_NoFilters(t *testing.T) {
	resetManager(t)
	mustCreate(t, coresquad.Squad{
		ID: "core", Name: "Core", Type: coresquad.SquadTypeStream,
		Domain: "platform", Members: []string{"alice"},
	})
	mustCreate(t, coresquad.Squad{
		ID: "ops", Name: "Ops", Type: coresquad.SquadTypePlatform,
		Domain: "infra", Members: []string{"bob"},
	})

	out := runListWith(t)
	for _, id := range []string{"core", "ops"} {
		if !strings.Contains(out, id) {
			t.Errorf("expected %s in unfiltered output: %s", id, out)
		}
	}
}

func TestSquadList_Empty(t *testing.T) {
	resetManager(t)
	out := runListWith(t)
	// Empty registry should not panic and should produce no
	// squad-id rows.
	for _, id := range []string{"core", "ops"} {
		if strings.Contains(out, id) {
			t.Errorf("unexpected squad id in empty output: %s", out)
		}
	}
}

func TestSquadToSummaryRow_TruncatesMembers(t *testing.T) {
	row := squadToSummaryRow(coresquad.Squad{
		ID: "big", Name: "Big", Type: coresquad.SquadTypeStream,
		Members: []string{"a", "b", "c", "d", "e"},
	})
	if row.MemberCount != 5 {
		t.Errorf("MemberCount=%d, want 5", row.MemberCount)
	}
	if !strings.Contains(row.Members, "+2") {
		t.Errorf("Members %q should include +2 suffix for truncation", row.Members)
	}
}
