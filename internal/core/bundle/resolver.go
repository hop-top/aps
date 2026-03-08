package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ProfileContext holds runtime information about the profile being resolved.
type ProfileContext struct {
	ID          string
	DisplayName string
	Email       string
	ConfigDir   string
	DataDir     string
	Runtime     string      // "claude" | "gemini" | "codex" | "" (detected from env)
	Scope       BundleScope // profile's own scope (union-merged with bundle scope)
}

// ResolvedBundle is the fully evaluated result of applying a Bundle to a profile context.
type ResolvedBundle struct {
	Bundle           Bundle
	Scope            BundleScope   // merged scope (union of bundle + profile)
	Env              map[string]string
	AlwaysServices   []ServiceEntry
	OnDemandServices []ServiceEntry
	BinaryResults    []BinaryResult
	Errors           []error
	Warnings         []string
}

// BinaryResult holds the per-binary evaluation outcome.
type BinaryResult struct {
	Binary    string
	Found     bool
	Skipped   bool   // missing:skip policy triggered
	Blocked   bool   // blocked:true
	Command   string // resolved command with template vars expanded
	DenyFlags []string
	Message   string
}

// Resolve fully evaluates bundle b in the given profile context.
//
// Steps:
//  1. Resolve inheritance (T-0044)
//  2. Merge scope — bundle union profile (T-0045)
//  3. Evaluate binary requirements (T-0046)
//  4. Resolve commands with template expansion (T-0047)
//  5. Enforce deny_flags policy (T-0048)
//  6. Classify services (T-0049)
//  7. Detect runtime and layer runtime_overrides env (T-0050)
//  8. Expand template vars in env values (T-0051)
func Resolve(b Bundle, reg *Registry, ctx ProfileContext) (*ResolvedBundle, error) {
	// T-0044 — Bundle inheritance
	effective, err := resolveInheritance(b, reg)
	if err != nil {
		return nil, err
	}

	rb := &ResolvedBundle{
		Bundle: effective,
	}

	// T-0045 — Scope union merge
	rb.Scope = unionScope(effective.Scope, ctx.Scope)

	// T-0046 / T-0047 / T-0048 — Binary requirement evaluation
	for _, req := range effective.Requires {
		result := evaluateBinary(req, ctx, rb)
		rb.BinaryResults = append(rb.BinaryResults, result)
	}

	// T-0049 — Service classification
	for _, svc := range effective.Services {
		switch svc.Start {
		case "always":
			rb.AlwaysServices = append(rb.AlwaysServices, svc)
		case "on-demand":
			rb.OnDemandServices = append(rb.OnDemandServices, svc)
		}
	}

	// T-0050 — Runtime detection and override env merge
	runtime := detectRuntime(ctx)

	// T-0051 — Env var injection with template expansion
	rb.Env = buildEnv(effective, runtime, ctx)

	return rb, nil
}

// resolveInheritance merges parent bundle fields into child where child fields are zero/empty.
// Inheritance is one level only (enforced by Validate).
func resolveInheritance(b Bundle, reg *Registry) (Bundle, error) {
	if b.Extends == "" {
		return b, nil
	}

	parent, err := reg.Get(b.Extends)
	if err != nil {
		return Bundle{}, fmt.Errorf("bundle %q: cannot resolve parent %q: %w", b.Name, b.Extends, err)
	}

	// Start from parent, then overlay child fields where child has non-zero values.
	merged := *parent

	// Always keep child identity fields.
	merged.Name = b.Name
	merged.Description = b.Description
	merged.Version = b.Version
	merged.Extends = b.Extends

	// Capabilities: child wins if set, else inherit.
	if len(b.Capabilities) > 0 {
		merged.Capabilities = b.Capabilities
	}

	// Scope: child wins per-field if set.
	merged.Scope = mergeInheritedScope(parent.Scope, b.Scope)

	// Env: parent env first, then child overwrites individual keys.
	merged.Env = mergeMaps(parent.Env, b.Env)

	// Services: child wins if set.
	if len(b.Services) > 0 {
		merged.Services = b.Services
	}

	// Requires: child wins if set.
	if len(b.Requires) > 0 {
		merged.Requires = b.Requires
	}

	// RuntimeOverrides: child wins per runtime key.
	merged.RuntimeOverrides = mergeRuntimeOverrides(parent.RuntimeOverrides, b.RuntimeOverrides)

	return merged, nil
}

// mergeInheritedScope overlays child scope fields onto parent — child wins per-field only when set.
func mergeInheritedScope(parent, child BundleScope) BundleScope {
	result := BundleScope{
		Operations:   parent.Operations,
		FilePatterns: parent.FilePatterns,
		Networks:     parent.Networks,
	}
	if len(child.Operations) > 0 {
		result.Operations = child.Operations
	}
	if len(child.FilePatterns) > 0 {
		result.FilePatterns = child.FilePatterns
	}
	if len(child.Networks) > 0 {
		result.Networks = child.Networks
	}
	return result
}

// mergeRuntimeOverrides merges two runtime override maps; child keys win.
func mergeRuntimeOverrides(parent, child map[string]RuntimeOverride) map[string]RuntimeOverride {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	out := make(map[string]RuntimeOverride, len(parent)+len(child))
	for k, v := range parent {
		out[k] = v
	}
	for k, v := range child {
		// Merge env within the same runtime key.
		if existing, ok := out[k]; ok {
			out[k] = RuntimeOverride{Env: mergeMaps(existing.Env, v.Env)}
		} else {
			out[k] = v
		}
	}
	return out
}

// mergeMaps merges two string maps; b wins over a for duplicate keys.
func mergeMaps(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// unionScope computes the union of two BundleScope values.
//
// Operations and Networks: deduplicated union.
// FilePatterns: deduplicated union; if "**" appears in either, result is ["**"].
func unionScope(a, b BundleScope) BundleScope {
	return BundleScope{
		Operations:   unionStrings(a.Operations, b.Operations),
		FilePatterns: unionFilePatterns(a.FilePatterns, b.FilePatterns),
		Networks:     unionStrings(a.Networks, b.Networks),
	}
}

// unionStrings returns a deduplicated union of two string slices.
func unionStrings(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	var out []string
	for _, s := range append(a, b...) {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// unionFilePatterns is like unionStrings but collapses to ["**"] when "**" is present.
func unionFilePatterns(a, b []string) []string {
	combined := unionStrings(a, b)
	for _, p := range combined {
		if p == "**" {
			return []string{"**"}
		}
	}
	return combined
}

// evaluateBinary evaluates a single BinaryRequirement against PATH and the profile context.
// It appends errors/warnings to rb and returns the BinaryResult.
func evaluateBinary(req BinaryRequirement, ctx ProfileContext, rb *ResolvedBundle) BinaryResult {
	result := BinaryResult{
		Binary:    req.Binary,
		DenyFlags: req.DenyFlags,
		Message:   req.Message,
	}

	// T-0047 — Resolve command with template variable substitution.
	result.Command = expandTemplateVars(req.Command, ctx)

	// T-0046 — Check PATH.
	_, lookErr := exec.LookPath(req.Binary)
	result.Found = lookErr == nil

	// blocked: true → always blocked regardless of presence.
	if req.Blocked {
		result.Blocked = true
		msg := req.Message
		if msg == "" {
			msg = fmt.Sprintf("binary %q is blocked", req.Binary)
		}
		rb.Errors = append(rb.Errors, fmt.Errorf("%s", msg))
		return result
	}

	// Apply missing policy when binary is not found.
	if !result.Found {
		switch req.Missing {
		case "skip":
			result.Skipped = true
		case "warn":
			msg := req.Message
			if msg == "" {
				msg = fmt.Sprintf("binary %q not found in PATH", req.Binary)
			}
			rb.Warnings = append(rb.Warnings, msg)
		case "error", "":
			// default to error when not found and no policy specified
			msg := req.Message
			if msg == "" {
				msg = fmt.Sprintf("required binary %q not found in PATH", req.Binary)
			}
			rb.Errors = append(rb.Errors, fmt.Errorf("%s", msg))
		}
		return result
	}

	// T-0048 — deny_flags enforcement (only relevant when binary is present).
	if len(req.DenyFlags) > 0 && req.DenyPolicy == "error" {
		rb.Errors = append(rb.Errors, fmt.Errorf(
			"binary %q has denied flags %v (deny_policy=error)", req.Binary, req.DenyFlags,
		))
	}
	// "strip" policy is metadata for the runtime shim — recorded in BinaryResult.DenyFlags and
	// available to callers; no further action needed here.

	return result
}

// expandTemplateVars substitutes ${PROFILE_*} placeholders in s.
func expandTemplateVars(s string, ctx ProfileContext) string {
	if s == "" {
		return s
	}
	replacer := strings.NewReplacer(
		"${PROFILE_ID}", ctx.ID,
		"${PROFILE_DISPLAY_NAME}", ctx.DisplayName,
		"${PROFILE_EMAIL}", ctx.Email,
		"${PROFILE_CONFIG_DIR}", ctx.ConfigDir,
		"${PROFILE_DATA_DIR}", ctx.DataDir,
	)
	return replacer.Replace(s)
}

// detectRuntime determines the active runtime from ProfileContext or environment variables.
func detectRuntime(ctx ProfileContext) string {
	if ctx.Runtime != "" {
		return ctx.Runtime
	}
	switch {
	case os.Getenv("CLAUDE_API_KEY") != "":
		return "claude"
	case os.Getenv("GEMINI_API_KEY") != "":
		return "gemini"
	case os.Getenv("OPENAI_API_KEY") != "":
		return "codex"
	}
	return ""
}

// buildEnv constructs the final env map: bundle env + runtime override env (runtime wins).
// All values have template vars expanded.
func buildEnv(b Bundle, runtime string, ctx ProfileContext) map[string]string {
	out := make(map[string]string, len(b.Env))

	// Base bundle env.
	for k, v := range b.Env {
		out[k] = expandTemplateVars(v, ctx)
	}

	// Runtime override env layers on top.
	if runtime != "" {
		if override, ok := b.RuntimeOverrides[runtime]; ok {
			for k, v := range override.Env {
				out[k] = expandTemplateVars(v, ctx)
			}
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
