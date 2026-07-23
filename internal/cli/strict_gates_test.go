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

// TestRootValidateSignature_NoViolations closes the gap AssertCLI
// leaves open: kit runs TWO independent validators at boot, and
// AssertCLI only exercises one of them.
//
//   - Root.Validate() — the Layer-A walk (annotations, Short/Long,
//     shape, configurable gates). AssertCLI calls this with
//     EnforceValidate forced on. Signature checks are NOT part of
//     Validate(); see kit cli.go (collectShippedValidation +
//     collectLayerAValidation only).
//   - Root.ValidateSignature() — the four signature checks
//     (local-globals, reserved-name, depth-hierarchical,
//     passthrough). Execute() runs this in a SEPARATE gate keyed on
//     Config.SignatureStrictness (kit cli.go, dispatchSignatureReport
//     after the EnforceValidate block), independent of
//     EnforceValidate.
//
// Because aps runs with SignatureStrictness=Reject, a leaf that
// redefines a global flag (e.g. a local --format shadowing kit's
// persistent output-mode --format) aborts EVERY invocation at
// startup — including --help — while AssertCLI stays green. This
// test invokes the exact walk Execute() dispatches on and fails on
// ANY violation: reject mode keys on report.HasViolations() without
// filtering severity (kit dispatchSignatureReport), so even a
// warning-severity passthrough entry aborts startup.
func TestRootValidateSignature_NoViolations(t *testing.T) {
	if root == nil || root.Cmd == nil {
		t.Fatal("aps root command tree is nil")
	}
	report := root.ValidateSignature()
	for _, v := range report.Violations {
		t.Errorf("signature violation: %s [%s/%s] %s", v.Path, v.Check, v.Severity, v.Detail)
	}
}
