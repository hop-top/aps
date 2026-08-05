package org

import (
	"errors"
	"testing"

	"hop.top/aps/internal/core"
)

// prof builds a minimal profile for graph tests.
func prof(id, reportsTo string) core.Profile {
	return core.Profile{ID: id, DisplayName: id, ReportsTo: reportsTo}
}

func ids(profiles []core.Profile) []string {
	out := make([]string, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p.ID)
	}
	return out
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// tree: ceo <- vp <- {eng1, eng2}; ceo <- cfo
func treeProfiles() []core.Profile {
	return []core.Profile{
		prof("ceo", ""),
		prof("vp", "ceo"),
		prof("eng1", "vp"),
		prof("eng2", "vp"),
		prof("cfo", "ceo"),
	}
}

func TestBuild_EmptyAndNil(t *testing.T) {
	for _, profiles := range [][]core.Profile{nil, {}} {
		g := Build(profiles)
		if g == nil {
			t.Fatal("Build() returned nil graph")
		}
		if roots := g.Roots(); len(roots) != 0 {
			t.Errorf("Roots() on empty graph = %v, want empty", ids(roots))
		}
		if _, ok := g.Manager("ghost"); ok {
			t.Error("Manager() on empty graph reported ok")
		}
	}
}

func TestBuild_DuplicateIDsLastWins(t *testing.T) {
	g := Build([]core.Profile{
		{ID: "dup", DisplayName: "first", ReportsTo: ""},
		{ID: "dup", DisplayName: "second", ReportsTo: ""},
	})
	roots := g.Roots()
	if len(roots) != 1 {
		t.Fatalf("Roots() = %v, want single deduplicated root", ids(roots))
	}
	if roots[0].DisplayName != "second" {
		t.Errorf("duplicate id resolution: DisplayName = %q, want %q (last wins)", roots[0].DisplayName, "second")
	}
}

func TestManager(t *testing.T) {
	g := Build(treeProfiles())
	cases := []struct {
		id     string
		want   string
		wantOK bool
	}{
		{"vp", "ceo", true},
		{"eng1", "vp", true},
		{"ceo", "", false},   // root has no manager
		{"ghost", "", false}, // unknown id
		{"", "", false},      // empty id
	}
	for _, c := range cases {
		got, ok := g.Manager(c.id)
		if ok != c.wantOK || (ok && got.ID != c.want) {
			t.Errorf("Manager(%q) = (%q, %v), want (%q, %v)", c.id, got.ID, ok, c.want, c.wantOK)
		}
	}
}

func TestManager_DanglingReportsTo(t *testing.T) {
	g := Build([]core.Profile{prof("a", "missing")})
	if _, ok := g.Manager("a"); ok {
		t.Error("Manager() with dangling reports_to reported ok")
	}
}

func TestChain_Tree(t *testing.T) {
	g := Build(treeProfiles())
	chain, err := g.Chain("eng1")
	if err != nil {
		t.Fatalf("Chain(eng1) error: %v", err)
	}
	if want := []string{"eng1", "vp", "ceo"}; !equalIDs(ids(chain), want) {
		t.Errorf("Chain(eng1) = %v, want %v", ids(chain), want)
	}

	chain, err = g.Chain("ceo")
	if err != nil {
		t.Fatalf("Chain(ceo) error: %v", err)
	}
	if want := []string{"ceo"}; !equalIDs(ids(chain), want) {
		t.Errorf("Chain(ceo) = %v, want %v", ids(chain), want)
	}
}

func TestChain_UnknownID(t *testing.T) {
	g := Build(treeProfiles())
	chain, err := g.Chain("ghost")
	if err != nil {
		t.Fatalf("Chain(ghost) error: %v", err)
	}
	if len(chain) != 0 {
		t.Errorf("Chain(ghost) = %v, want empty", ids(chain))
	}
}

func TestChain_DanglingReportsTo(t *testing.T) {
	// Dangling manager: chain stops at the last known profile, no error.
	g := Build([]core.Profile{prof("a", "missing")})
	chain, err := g.Chain("a")
	if err != nil {
		t.Fatalf("Chain(a) error: %v", err)
	}
	if want := []string{"a"}; !equalIDs(ids(chain), want) {
		t.Errorf("Chain(a) = %v, want %v", ids(chain), want)
	}
}

func TestChain_TwoNodeCycle(t *testing.T) {
	g := Build([]core.Profile{prof("a", "b"), prof("b", "a")})
	chain, err := g.Chain("a")
	if err == nil {
		t.Fatal("Chain() on 2-node cycle: want cycle error, got nil")
	}
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("Chain() error = %T, want *CycleError", err)
	}
	if want := []string{"a", "b"}; !equalIDs(cycleErr.Path, want) {
		t.Errorf("CycleError.Path = %v, want %v", cycleErr.Path, want)
	}
	// Partial chain: each node visited once, in walk order.
	if want := []string{"a", "b"}; !equalIDs(ids(chain), want) {
		t.Errorf("Chain() partial = %v, want %v", ids(chain), want)
	}
}

func TestChain_SelfCycle(t *testing.T) {
	g := Build([]core.Profile{prof("a", "a")})
	chain, err := g.Chain("a")
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("Chain() on self cycle: error = %v, want *CycleError", err)
	}
	if want := []string{"a"}; !equalIDs(cycleErr.Path, want) {
		t.Errorf("CycleError.Path = %v, want %v", cycleErr.Path, want)
	}
	if want := []string{"a"}; !equalIDs(ids(chain), want) {
		t.Errorf("Chain() partial = %v, want %v", ids(chain), want)
	}
}

func TestDirectReports(t *testing.T) {
	g := Build(treeProfiles())
	cases := []struct {
		id   string
		want []string
	}{
		{"ceo", []string{"cfo", "vp"}}, // sorted by id
		{"vp", []string{"eng1", "eng2"}},
		{"eng1", nil},
		{"ghost", nil},
	}
	for _, c := range cases {
		got := ids(g.DirectReports(c.id))
		if !equalIDs(got, c.want) {
			t.Errorf("DirectReports(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

func TestDirectReports_TerminatesOnCycle(t *testing.T) {
	g := Build([]core.Profile{prof("a", "b"), prof("b", "a"), prof("self", "self")})
	if got := ids(g.DirectReports("a")); !equalIDs(got, []string{"b"}) {
		t.Errorf("DirectReports(a) on cyclic data = %v, want [b]", got)
	}
	if got := ids(g.DirectReports("self")); !equalIDs(got, []string{"self"}) {
		t.Errorf("DirectReports(self) = %v, want [self]", got)
	}
}

func TestTransitiveReports_Unlimited(t *testing.T) {
	g := Build(treeProfiles())
	for _, depth := range []int{0, -1} {
		got := ids(g.TransitiveReports("ceo", depth))
		if want := []string{"cfo", "eng1", "eng2", "vp"}; !equalIDs(got, want) {
			t.Errorf("TransitiveReports(ceo, %d) = %v, want %v", depth, got, want)
		}
	}
}

func TestTransitiveReports_DepthLimited(t *testing.T) {
	g := Build(treeProfiles())
	cases := []struct {
		depth int
		want  []string
	}{
		{1, []string{"cfo", "vp"}},
		{2, []string{"cfo", "eng1", "eng2", "vp"}},
		{99, []string{"cfo", "eng1", "eng2", "vp"}},
	}
	for _, c := range cases {
		got := ids(g.TransitiveReports("ceo", c.depth))
		if !equalIDs(got, c.want) {
			t.Errorf("TransitiveReports(ceo, %d) = %v, want %v", c.depth, got, c.want)
		}
	}
}

func TestTransitiveReports_UnknownID(t *testing.T) {
	g := Build(treeProfiles())
	if got := g.TransitiveReports("ghost", 0); len(got) != 0 {
		t.Errorf("TransitiveReports(ghost) = %v, want empty", ids(got))
	}
}

func TestTransitiveReports_TerminatesOnCycle(t *testing.T) {
	// a <-> b cycle with c hanging off b; self-loop on d.
	g := Build([]core.Profile{
		prof("a", "b"),
		prof("b", "a"),
		prof("c", "b"),
		prof("d", "d"),
	})
	if got := ids(g.TransitiveReports("a", 0)); !equalIDs(got, []string{"b", "c"}) {
		t.Errorf("TransitiveReports(a, 0) on cyclic data = %v, want [b c]", got)
	}
	if got := ids(g.TransitiveReports("d", 0)); len(got) != 0 {
		t.Errorf("TransitiveReports(d, 0) on self loop = %v, want empty (self excluded)", got)
	}
}

func TestRoots_Forest(t *testing.T) {
	g := Build([]core.Profile{
		prof("z-root", ""),
		prof("a-root", ""),
		prof("kid", "z-root"),
		prof("dangler", "missing"), // dangling target is NOT a root
	})
	if got := ids(g.Roots()); !equalIDs(got, []string{"a-root", "z-root"}) {
		t.Errorf("Roots() = %v, want [a-root z-root]", got)
	}
}

func TestGraph_Determinism(t *testing.T) {
	base := []core.Profile{
		prof("ceo", ""),
		prof("vp", "ceo"),
		prof("eng1", "vp"),
		prof("eng2", "vp"),
		prof("cfo", "ceo"),
		prof("root2", ""),
	}
	// Fixed permutations instead of rand: deterministic test, same coverage.
	perms := [][]int{
		{0, 1, 2, 3, 4, 5},
		{5, 4, 3, 2, 1, 0},
		{2, 5, 0, 4, 1, 3},
	}
	var wantRoots, wantReports, wantTransitive []string
	for i, perm := range perms {
		shuffled := make([]core.Profile, len(base))
		for j, k := range perm {
			shuffled[j] = base[k]
		}
		g := Build(shuffled)
		roots := ids(g.Roots())
		reports := ids(g.DirectReports("ceo"))
		transitive := ids(g.TransitiveReports("ceo", 0))
		if i == 0 {
			wantRoots, wantReports, wantTransitive = roots, reports, transitive
			continue
		}
		if !equalIDs(roots, wantRoots) {
			t.Errorf("perm %d: Roots() = %v, want %v", i, roots, wantRoots)
		}
		if !equalIDs(reports, wantReports) {
			t.Errorf("perm %d: DirectReports(ceo) = %v, want %v", i, reports, wantReports)
		}
		if !equalIDs(transitive, wantTransitive) {
			t.Errorf("perm %d: TransitiveReports(ceo) = %v, want %v", i, transitive, wantTransitive)
		}
	}
	if !equalIDs(wantRoots, []string{"ceo", "root2"}) {
		t.Errorf("Roots() = %v, want sorted [ceo root2]", wantRoots)
	}
	if !equalIDs(wantReports, []string{"cfo", "vp"}) {
		t.Errorf("DirectReports(ceo) = %v, want sorted [cfo vp]", wantReports)
	}
}
