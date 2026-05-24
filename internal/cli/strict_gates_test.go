package cli

import (
	"testing"

	kitconformance "hop.top/kit/go/conformance"
)

// TestRootValidate_StrictGatesPass is the regression net for kit 0.4's
// strict-validation conformance work (track aps-kit-12fcc-conformance,
// tasks T-0648 → T-0653 → T-0657 → T-0662).
//
// Per the conformance plan's T-0662 phase, this test delegates the
// entire Layer-A + signature contract to kit's shipped helper
// kitconformance.AssertCLI. AssertCLI forces EnforceValidate=true for
// the duration of the check (the adopter's Config is restored after),
// runs root.Validate() against the live tree, and surfaces the
// returned *kitcli.ValidationError as t.Errorf with full per-bucket
// detail (missing kit/side-effect, missing Long:, depth-1 leaves
// without kit/top-level-verb, MaxTopLevelVerbs overflow, signature
// violations across all four checks, etc.).
//
// The production aps Config (see root.go) already runs with
// EnforceDestructiveToken, EnforceGuidance, EnforceDryRunRationale,
// SignatureStrictness=Reject, and ValidationFailureMode=Error — so
// AssertCLI's defaulted Options{} (which leaves the configurable
// gates at the adopter-provided values) exercises the same surface
// kit/cli walks at boot. A regression that re-introduces any
// violation fails here loudly, ahead of CI catching it through the
// broader Execute() dispatch.
//
// Kit API references (module cache, hop.top/kit@v0.4.0-alpha.4):
//
//   - go/conformance/conformance.go: kitconformance.AssertCLI
//   - go/console/cli/cli.go:        Config.EnforceValidate /
//     ValidationFailureMode / SignatureStrictness wiring
//   - go/console/cli/validate.go:   the Layer-A bucket walk
func TestRootValidate_StrictGatesPass(t *testing.T) {
	if root == nil || root.Cmd == nil {
		t.Fatal("aps root command tree is nil")
	}
	// AssertCLI already calls t.Errorf with per-bucket detail on failure;
	// the returned *kitcli.ValidationError is for callers who want to
	// inspect a specific bucket shape. We don't, so discard it.
	_ = kitconformance.AssertCLI(t, root)
}
