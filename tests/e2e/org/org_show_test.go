package org_e2e

import (
	"strings"
	"testing"
)

// TestOrgShow_MidTree: a mid-hierarchy profile reports its chain up to
// root, its direct reports, and its transitive reports.
func TestOrgShow_MidTree(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "vp")
	if err != nil {
		t.Fatalf("org show vp: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"self", "manager", "direct-report", "transitive-report"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing relation %q:\n%s", want, stdout)
		}
	}
	for _, id := range []string{"vp", "ceo", "eng1", "eng2", "intern"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("stdout missing profile %q:\n%s", id, stdout)
		}
	}
	// vp's only channel is email; the CHANNELS column must carry it.
	if !strings.Contains(stdout, "email") {
		t.Errorf("stdout missing email channel:\n%s", stdout)
	}
}

// TestOrgShow_Root: a root profile has no manager rows.
func TestOrgShow_Root(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "ceo")
	if err != nil {
		t.Fatalf("org show ceo: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stdout, "manager") {
		t.Errorf("root profile must have no manager rows:\n%s", stdout)
	}
	for _, id := range []string{"vp", "eng1", "eng2", "intern"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("stdout missing report %q:\n%s", id, stdout)
		}
	}
}

// TestOrgShow_Leaf: a leaf profile has no report rows.
func TestOrgShow_Leaf(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "intern")
	if err != nil {
		t.Fatalf("org show intern: %v\nstderr: %s", err, stderr)
	}
	for _, rel := range []string{"direct-report", "transitive-report"} {
		if strings.Contains(stdout, rel) {
			t.Errorf("leaf profile must have no %s rows:\n%s", rel, stdout)
		}
	}
	// Chain: intern -> eng1 -> vp -> ceo.
	for _, id := range []string{"intern", "eng1", "vp", "ceo"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("stdout missing chain member %q:\n%s", id, stdout)
		}
	}
}

// TestOrgShow_UnknownID: an unknown profile id is a clear error.
func TestOrgShow_UnknownID(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "nobody")
	if err == nil {
		t.Fatalf("org show nobody: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, "nobody") {
		t.Errorf("stderr missing unknown id:\n%s", stderr)
	}
}

// TestOrgShow_DepthLimit: --depth 1 stops transitive expansion at
// direct reports.
func TestOrgShow_DepthLimit(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "vp", "--depth", "1")
	if err != nil {
		t.Fatalf("org show vp --depth 1: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stdout, "intern") {
		t.Errorf("--depth 1 must not include depth-2 report intern:\n%s", stdout)
	}
	for _, id := range []string{"eng1", "eng2"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("stdout missing direct report %q:\n%s", id, stdout)
		}
	}
}

// TestOrgShow_CycleErrorsNotHangs: cyclic reports_to data terminates
// with a partial chain plus a clear error, never an infinite loop.
func TestOrgShow_CycleErrorsNotHangs(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedCycle(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "show", "loopa")
	if err == nil {
		t.Fatalf("org show on cyclic data: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, "cycle") {
		t.Errorf("stderr missing cycle error:\n%s", stderr)
	}
	// Partial chain must still be printed.
	if !strings.Contains(stdout, "loopa") || !strings.Contains(stdout, "loopb") {
		t.Errorf("stdout missing partial chain:\n%s", stdout)
	}
}
