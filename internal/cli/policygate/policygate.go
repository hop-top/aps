// Package policygate exposes a synchronous pre_persisted publisher used
// by aps CLI delete handlers (T-1292) to consult kit/runtime/policy
// vetoes before mutating state.
//
// Why a separate package: aps subcommand subpackages (cli/session,
// cli/workspace, …) cannot import internal/cli (the parent registers
// them, so an import would form a cycle). policygate is a thin seam
// the parent cli package wires at init via SetBus, and subpackages
// reach into via Publish.
//
// Contract:
//
//   - SetBus(b) is called once from internal/cli (bus.go init) once
//     the process bus is constructed; nil-safe.
//   - Publish(ctx, kind, op, id) fires kit.runtime.entity.pre_persisted
//     synchronously. The kit/runtime/policy engine subscribes to that
//     topic and vetoes via *policy.PolicyDeniedError when a rule denies.
//   - The kind string is stuffed into ctx via clinote.WithContextAttrs
//     before Publish so policy CEL can read context.request_attrs.kind.
//   - When the bus is nil (no APS_BUS_TOKEN, tests bypassing init),
//     Publish is a no-op and returns nil. Adopters who want enforcement
//     in tests construct a bus + wire policy themselves.
package policygate

import (
	"context"
	"errors"

	"hop.top/kit/go/console/output"
	"hop.top/kit/go/runtime/bus"
	"hop.top/kit/go/runtime/domain"
	"hop.top/kit/go/runtime/policy"
)

// processBus is the kit/runtime/bus shared with the rest of the CLI.
// nil means publishes short-circuit — see package doc.
var processBus bus.Bus

// SetBus wires the process bus. Idempotent; nil disables enforcement.
func SetBus(b bus.Bus) { processBus = b }

// PublishDeletePrePersisted fires kit.runtime.entity.pre_persisted with
// Op=delete on the process bus. ctx is enriched in-place with the kind
// discriminator under policy.ContextAttrsKey so CEL rules can read
// context.request_attrs.kind. Returns the bus's Publish error which
// surfaces *policy.PolicyDeniedError when a rule denies; callers map
// that to exit code 4 via internal/cli/exit (PolicyDeniedError unwraps
// to domain.ErrConflict).
//
// kind is the entity discriminator the bundled rules key on:
//   - "session"           — collaboration sessions and process registry sessions
//   - "workspace_context" — workspace shared context variables
//
// Adopter-authored rules can extend the taxonomy without code changes
// by gating on different context.request_attrs.kind values.
func PublishDeletePrePersisted(ctx context.Context, kind, id string) (context.Context, error) {
	return PublishDeletePrePersistedWithAttrs(ctx, kind, id, nil)
}

// PublishDeletePrePersistedWithAttrs is the workspace-aware variant of
// PublishDeletePrePersisted. extra is merged into request_attrs alongside
// the kind discriminator; nil is equivalent to PublishDeletePrePersisted.
//
// The conventional key is "workspace_id" (T-1308) — the aps principal
// resolver reads it to look up the calling profile's workspace role for
// principal.role. Other keys can be added without code changes; CEL
// rules read context.request_attrs.<key>.
func PublishDeletePrePersistedWithAttrs(ctx context.Context, kind, id string, extra map[string]any) (context.Context, error) {
	ctx = withRequestAttrs(ctx, kind, extra)
	if processBus == nil {
		return ctx, nil
	}
	payload := domain.PreEntityPayload{
		Op:       domain.OpDelete,
		Phase:    domain.PhasePrePersisted,
		EntityID: id,
	}
	ev := bus.NewEvent("kit.runtime.entity.pre_persisted", "aps.cli", payload)
	if err := processBus.Publish(ctx, ev); err != nil {
		return ctx, asCLIError(err)
	}
	return ctx, nil
}

// asCLIError converts a policy veto into an *output.Error envelope
// (Code=CONFLICT, ExitCode=4) so kit's RunE wrapper passes it through
// to internal/cli/exit.Code unchanged. Without this shim, kit's
// toCLIError wraps any error not implementing asCLIError as
// CodeGeneric/ExitCode=1, which would lose the 4 mapping. Other errors
// pass through verbatim — only PolicyDeniedError + ErrConflict get
// the structured envelope.
func asCLIError(err error) error {
	if err == nil {
		return nil
	}
	var pde *policy.PolicyDeniedError
	if errors.As(err, &pde) {
		return &output.Error{
			Code:     output.CodeConflict,
			Message:  pde.Error(),
			ExitCode: 4,
		}
	}
	if errors.Is(err, domain.ErrConflict) {
		return &output.Error{
			Code:     output.CodeConflict,
			Message:  err.Error(),
			ExitCode: 4,
		}
	}
	return err
}

// withRequestAttrs merges {kind: <kind>, ...extra} into ctx's
// request_attrs map so CEL rules can read context.request_attrs.<key>.
// Preserves any pre-existing note (set by clinote.WithContext) and any
// other attrs already on ctx. extra takes precedence over existing
// request_attrs entries with the same key, and kind always wins over an
// explicit "kind" in extra so callers can't accidentally shadow the
// discriminator.
//
// Used by PublishDeletePrePersistedWithAttrs (T-1308) for bulk merges
// like (kind=session, workspace_id=...). Single-attr callers use
// withRequestAttr / withKind / WithContextVariableAttrs (T-1309).
func withRequestAttrs(ctx context.Context, kind string, extra map[string]any) context.Context {
	merged := copyMergedAttrs(ctx, len(extra)+1)
	attrs := merged["request_attrs"].(map[string]any)
	for k, v := range extra {
		attrs[k] = v
	}
	attrs["kind"] = kind
	return context.WithValue(ctx, policy.ContextAttrsKey, merged)
}

// WithContextVariableAttrs (T-1309) merges the visibility attribute
// into ctx's request_attrs map so CEL rules can read
// context.request_attrs.visibility. The kind discriminator
// ("workspace_context") is set in lockstep so a single rule can
// gate on the (kind, visibility) tuple — e.g.
//
//	when: 'context.request_attrs.kind == "workspace_context"
//	       && context.request_attrs.visibility == "private"'
//
// Other request_attrs already on ctx (e.g. note from
// clinote.WithContext) are preserved.
func WithContextVariableAttrs(ctx context.Context, visibility string) context.Context {
	ctx = withKind(ctx, "workspace_context")
	return withRequestAttr(ctx, "visibility", visibility)
}

// withKind merges {kind: <kind>} into ctx's request_attrs map so CEL
// rules can read context.request_attrs.kind. Preserves any pre-existing
// note (set by clinote.WithContext) and other attrs.
func withKind(ctx context.Context, kind string) context.Context {
	return withRequestAttr(ctx, "kind", kind)
}

// withRequestAttr merges {<key>: <value>} into ctx's
// request_attrs map under policy.ContextAttrsKey. Preserves all
// pre-existing keys (note, kind, visibility, …).
func withRequestAttr(ctx context.Context, key string, value any) context.Context {
	merged := copyMergedAttrs(ctx, 1)
	merged["request_attrs"].(map[string]any)[key] = value
	return context.WithValue(ctx, policy.ContextAttrsKey, merged)
}

// copyMergedAttrs returns a fresh policy.ContextAttrsKey map with
// existing top-level keys (note, …) preserved and request_attrs
// initialized as a fresh map containing the previously-set attrs.
// hint sizes the inner request_attrs map for the caller's expected
// additions; callers mutate the returned map's "request_attrs" entry.
func copyMergedAttrs(ctx context.Context, hint int) map[string]any {
	existing, _ := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	merged := make(map[string]any, len(existing)+1)
	var existingAttrs map[string]any
	for k, v := range existing {
		if k == "request_attrs" {
			if m, ok := v.(map[string]any); ok {
				existingAttrs = m
			}
			continue
		}
		merged[k] = v
	}
	attrs := make(map[string]any, len(existingAttrs)+hint)
	for k, v := range existingAttrs {
		attrs[k] = v
	}
	merged["request_attrs"] = attrs
	return merged
}
