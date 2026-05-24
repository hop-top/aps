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
// added across ~169 leaves. The signature-validator pre-flight, on
// the other hand, is INDEPENDENT of EnforceValidate (kit cli.go:784)
// and is now configured at strictness=reject (T-0653) — meaning any
// non-empty ValidateSignature() report turns into a typed
// *kitcli.ValidationError out of Execute() via ValidationFailureMode.
// This test exists as the leaf-level regression net: it walks the
// signature checks directly and asserts each one is empty. A diff
// that re-introduces a violation fails here loudly, ahead of CI
// catching it through the broader Execute() path.
//
// Phases (each next-tightening task flips exactly one knob):
//
//   - T-0648: SignatureStrictness silent → warn (no test change yet,
//     warnings start surfacing through slog at boot).
//   - T-0653: SignatureStrictness warn → reject — this test pivots
//     from "ratchet down per-check ceilings" to "assert each of the
//     four signature checks is empty." All four are at 0 today;
//     keeping them at 0 is now the contract.
//   - T-0657: ValidationFailureMode → Error AND DisableValidate → false
//     AND EnforceValidate → true (Layer-A annotation buckets start
//     firing; this test pivots to AssertCLI-style coverage).
//   - T-0662: kitconformance.AssertCLI(t, root) is the only assertion;
//     all knob plumbing in this file is deleted.
//
// Kit API references (module cache, hop.top/kit@v0.4.0-alpha.4):
//
//   - go/console/cli/validate_signature.go  Root.ValidateSignature
//   - go/console/cli/cli.go:144              SignatureStrictnessReject
//   - go/console/cli/cli.go:784              SignatureStrictness gate
//   - go/conformance/conformance.go:72       kitconformance.AssertCLI
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

	// Bucket violations by check id so we can report per-check below.
	pathsByCheck := map[string][]string{}
	for _, v := range report.Violations {
		pathsByCheck[v.Check] = append(pathsByCheck[v.Check], v.Path)
	}

	// T-0653 contract: every signature check is zero-tolerance. Any
	// non-empty bucket is a regression that would also trip the
	// reject-mode dispatch out of root.Execute() — we surface it here
	// so the leaf adding the violation gets named directly in the
	// failure message.
	for _, check := range []string{
		kitcli.SignatureCheckReservedName,
		kitcli.SignatureCheckPassthrough,
		kitcli.SignatureCheckLocalGlobals,
		kitcli.SignatureCheckDepthHierarchical,
	} {
		paths := pathsByCheck[check]
		if len(paths) == 0 {
			continue
		}
		sort.Strings(paths)
		t.Errorf("signature check %q expected 0 violations, got %d at:\n  %s",
			check, len(paths), strings.Join(paths, "\n  "))
	}
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
