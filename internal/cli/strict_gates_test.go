package cli

import (
	"sort"
	"strings"
	"testing"

	kitcli "hop.top/kit/go/console/cli"
)

// TestRootValidate_StrictGatesPass is the regression net for kit 0.4's
// strict-validation conformance work (track aps-kit-12fcc-conformance,
// tasks T-0648 → T-0653 → T-0657 → T-0662).
//
// Production code sets cli.Config.DisableValidate=true on the aps root
// (see root.go) to bypass kit's EnforceValidate pre-flight while the
// missing kit/side-effect + kit/idempotent annotations are still being
// added across ~169 leaves. This test runs validation REGARDLESS of
// that production opt-out so it can act as the tightening dial: each
// companion task flips one knob below, the test catches drift, and the
// adopter contract converges on the kit 0.4 default.
//
// Phases (each next-tightening task flips exactly one knob):
//
//   - T-0648: SignatureStrictness silent → warn (no test change yet,
//     warnings start surfacing through slog at boot).
//   - T-0653: SignatureStrictness warn → reject (this test stops
//     tolerating SignatureCheckLocalGlobals + SignatureCheckDepth-
//     Hierarchical violations; the high-water mark below drops to 0
//     and the entire ValidateSignature() must be empty).
//   - T-0657: ValidationFailureMode → Error AND DisableValidate → false
//     AND EnforceValidate → true (Layer-A annotation buckets start
//     firing; this test pivots to AssertCLI-style coverage).
//   - T-0662: kitconformance.AssertCLI(t, root) is the only assertion;
//     all knob plumbing in this file is deleted.
//
// Today (T-0658, annotation coverage ≈ 0%) the test asserts:
//
//   - ValidateSignature() runs to completion (kit's signature walk is
//     wired correctly, no panic, no nil deref).
//   - reserved-name check is clean: no child of a parent declaring
//     kit/reserves-children shadows a reserved name. This is the only
//     signature check that is genuinely zero today and stays zero.
//   - passthrough check is clean: every leaf using cobra.ArbitraryArgs
//     declares kit/passthrough. (Warnings, not errors — but we hold
//     the line.)
//   - local-globals and depth-hierarchical errors do not exceed their
//     current high-water mark. Lowering these counts is the day-job of
//     the conformance track; raising them is a regression and must
//     fail loudly here.
//
// Kit API references (module cache, hop.top/kit@v0.4.0-alpha.3):
//
//   - go/console/cli/validate_signature.go:104  Root.ValidateSignature
//   - go/console/cli/cli.go:825                 Root.Validate
//   - go/conformance/conformance.go:72          kitconformance.AssertCLI
//     (deferred to T-0662; today's tree fails AssertCLI because the
//     shipped side-effect/idempotency arms also run.)
func TestRootValidate_StrictGatesPass(t *testing.T) {
	if root == nil || root.Cmd == nil {
		t.Fatal("aps root command tree is nil")
	}

	report := root.ValidateSignature()
	if report == nil {
		t.Fatal("ValidateSignature returned nil report")
	}

	// Bucket violations by check id so we can assert per-check below.
	counts := map[string]int{}
	pathsByCheck := map[string][]string{}
	for _, v := range report.Violations {
		counts[v.Check]++
		pathsByCheck[v.Check] = append(pathsByCheck[v.Check], v.Path)
	}

	// Fundamental shape rules that are zero today and MUST stay zero.
	// A non-zero count here is a real bug, not an annotation gap.
	for _, check := range []string{
		kitcli.SignatureCheckReservedName,
		kitcli.SignatureCheckPassthrough,
	} {
		if n := counts[check]; n != 0 {
			sort.Strings(pathsByCheck[check])
			t.Errorf("signature check %q expected 0 violations, got %d at:\n  %s",
				check, n, strings.Join(pathsByCheck[check], "\n  "))
		}
	}

	// High-water marks. These are the live regressions the conformance
	// track is unwinding. We freeze them here so PRs that add a fresh
	// leaf with --format/--profile shadow flags or a depth-3 chain
	// without kit/hierarchical fail this test immediately. Each time a
	// conformance PR lands and drops one of these counts, bump the
	// number down — never up. T-0653 flips both to 0.
	const (
		maxLocalGlobals      = 67 // T-0658 snapshot; T-0653 → 0
		maxDepthHierarchical = 44 // T-0658 snapshot; T-0653 → 0
	)
	enforceCeiling(t, kitcli.SignatureCheckLocalGlobals,
		counts[kitcli.SignatureCheckLocalGlobals], maxLocalGlobals,
		pathsByCheck[kitcli.SignatureCheckLocalGlobals])
	enforceCeiling(t, kitcli.SignatureCheckDepthHierarchical,
		counts[kitcli.SignatureCheckDepthHierarchical], maxDepthHierarchical,
		pathsByCheck[kitcli.SignatureCheckDepthHierarchical])
}

// enforceCeiling fails the test when got > limit. The limit is the
// current high-water mark; when conformance work brings the count
// down, the test author is expected to lower the constant in the
// caller. A got value BELOW the limit is fine (and is the whole
// point — we're ratcheting toward zero).
func enforceCeiling(t *testing.T, check string, got, limit int, paths []string) {
	t.Helper()
	if got <= limit {
		return
	}
	sort.Strings(paths)
	t.Errorf("signature check %q regressed: got %d violations, ceiling %d. "+
		"Either fix the new violation(s) below or, if this is intentional "+
		"churn during conformance work, raise the ceiling explicitly:\n  %s",
		check, got, limit, strings.Join(paths, "\n  "))
}

// Package-level pins: if kit drops or renames any of the symbols this
// test depends on in a future bump, the package fails to compile here
// and the upgrade PR is forced to address it.
var (
	_ = (*kitcli.Root)(nil).ValidateSignature
	_ = kitcli.SignatureCheckReservedName
	_ = kitcli.SignatureCheckLocalGlobals
	_ = kitcli.SignatureCheckDepthHierarchical
	_ = kitcli.SignatureCheckPassthrough
)
