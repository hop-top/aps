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
	ctx = withKind(ctx, kind)
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

// withKind merges {kind: <kind>} into ctx's request_attrs map so CEL
// rules can read context.request_attrs.kind. Preserves any pre-existing
// note (set by clinote.WithContext) and other attrs.
func withKind(ctx context.Context, kind string) context.Context {
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
	attrs := make(map[string]any, len(existingAttrs)+1)
	for k, v := range existingAttrs {
		attrs[k] = v
	}
	attrs["kind"] = kind
	merged["request_attrs"] = attrs
	return context.WithValue(ctx, policy.ContextAttrsKey, merged)
}
