package org

import (
	"testing"

	"hop.top/aps/internal/core"
)

func findingsOfKind(fs []Finding, kind FindingKind) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

func TestValidate_CleanTree(t *testing.T) {
	g := Build(treeProfiles())
	if fs := g.Validate(); len(fs) != 0 {
		t.Errorf("Validate() on clean tree = %+v, want none", fs)
	}
}

func TestValidate_Empty(t *testing.T) {
	if fs := Build(nil).Validate(); len(fs) != 0 {
		t.Errorf("Validate() on empty graph = %+v, want none", fs)
	}
}

func TestValidate_Cycle(t *testing.T) {
	g := Build([]core.Profile{
		prof("root", ""),
		prof("b", "c"),
		prof("c", "b"),
	})
	fs := findingsOfKind(g.Validate(), FindingCycle)
	if len(fs) != 1 {
		t.Fatalf("cycle findings = %+v, want exactly 1", fs)
	}
	// Full cycle path, canonicalized to start at the smallest id.
	if want := []string{"b", "c"}; !equalIDs(fs[0].ProfileIDs, want) {
		t.Errorf("cycle ProfileIDs = %v, want %v", fs[0].ProfileIDs, want)
	}
	if fs[0].Message == "" {
		t.Error("cycle finding has empty message")
	}
}

func TestValidate_CycleReportedOnce(t *testing.T) {
	// 3-node cycle plus a tail feeding into it: one cycle finding only.
	g := Build([]core.Profile{
		prof("x", "y"),
		prof("y", "z"),
		prof("z", "x"),
		prof("tail", "x"),
	})
	fs := findingsOfKind(g.Validate(), FindingCycle)
	if len(fs) != 1 {
		t.Fatalf("cycle findings = %+v, want exactly 1", fs)
	}
	if want := []string{"x", "y", "z"}; !equalIDs(fs[0].ProfileIDs, want) {
		t.Errorf("cycle ProfileIDs = %v, want %v", fs[0].ProfileIDs, want)
	}
}

func TestValidate_SelfReference(t *testing.T) {
	g := Build([]core.Profile{prof("ouro", "ouro")})
	fs := g.Validate()
	self := findingsOfKind(fs, FindingSelfReference)
	if len(self) != 1 {
		t.Fatalf("self-reference findings = %+v, want exactly 1", fs)
	}
	if want := []string{"ouro"}; !equalIDs(self[0].ProfileIDs, want) {
		t.Errorf("self-reference ProfileIDs = %v, want %v", self[0].ProfileIDs, want)
	}
	// A self loop is reported as self-reference, not additionally as a cycle.
	if cycles := findingsOfKind(fs, FindingCycle); len(cycles) != 0 {
		t.Errorf("self loop also reported as cycle: %+v", cycles)
	}
}

func TestValidate_DanglingReportsTo(t *testing.T) {
	g := Build([]core.Profile{prof("a", "missing")})
	fs := findingsOfKind(g.Validate(), FindingDanglingReportsTo)
	if len(fs) != 1 {
		t.Fatalf("dangling findings = %+v, want exactly 1", fs)
	}
	if want := []string{"a"}; !equalIDs(fs[0].ProfileIDs, want) {
		t.Errorf("dangling ProfileIDs = %v, want %v", fs[0].ProfileIDs, want)
	}
}

func TestValidate_UnknownType(t *testing.T) {
	g := Build([]core.Profile{
		{ID: "ok-agent", Type: core.ProfileTypeAgent},
		{ID: "ok-human", Type: core.ProfileTypeHuman},
		{ID: "ok-empty"},
		{ID: "typoed", Type: "person"},
	})
	fs := findingsOfKind(g.Validate(), FindingUnknownType)
	if len(fs) != 1 {
		t.Fatalf("unknown-type findings = %+v, want exactly 1", fs)
	}
	if want := []string{"typoed"}; !equalIDs(fs[0].ProfileIDs, want) {
		t.Errorf("unknown-type ProfileIDs = %v, want %v", fs[0].ProfileIDs, want)
	}
}

func TestValidate_AllKindsTogetherDeterministic(t *testing.T) {
	profiles := []core.Profile{
		prof("root", ""),
		prof("b", "c"),
		prof("c", "b"),
		prof("dangler", "missing"),
		prof("ouro", "ouro"),
		{ID: "typoed", Type: "person"},
	}
	perms := [][]int{
		{0, 1, 2, 3, 4, 5},
		{5, 4, 3, 2, 1, 0},
		{3, 0, 5, 2, 4, 1},
	}
	var want []Finding
	for i, perm := range perms {
		shuffled := make([]core.Profile, len(profiles))
		for j, k := range perm {
			shuffled[j] = profiles[k]
		}
		fs := Build(shuffled).Validate()
		if i == 0 {
			want = fs
			for _, kind := range []FindingKind{FindingCycle, FindingDanglingReportsTo, FindingSelfReference, FindingUnknownType} {
				if n := len(findingsOfKind(fs, kind)); n != 1 {
					t.Errorf("findings of kind %q = %d, want 1 (all: %+v)", kind, n, fs)
				}
			}
			continue
		}
		if len(fs) != len(want) {
			t.Fatalf("perm %d: %d findings, want %d", i, len(fs), len(want))
		}
		for j := range fs {
			if fs[j].Kind != want[j].Kind || !equalIDs(fs[j].ProfileIDs, want[j].ProfileIDs) {
				t.Errorf("perm %d: finding[%d] = %+v, want %+v", i, j, fs[j], want[j])
			}
		}
	}
}
