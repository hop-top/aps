package collaboration

import (
	"context"
	"fmt"

	"hop.top/aps/internal/events"
	"hop.top/kit/go/runtime/bus"
)

// GlobalAuditWorkspace is the synthetic workspace ID used to file audit
// events that have no workspace context (e.g. profile + adapter
// lifecycle events). Querying the audit log with this workspace ID
// returns the global aps timeline.
const GlobalAuditWorkspace = "aps:global"

// SubscribeAudit wires the WorkspaceAuditLog as a bus subscriber. It
// subscribes to "aps.#" (all aps.* descendants) and converts each
// delivered event into an AuditEvent recorded against the appropriate
// workspace:
//
//   - Session events route to the session's WorkspaceID when set,
//     falling back to GlobalAuditWorkspace.
//   - Profile + adapter events route to GlobalAuditWorkspace.
//
// The returned Unsubscribe detaches the handler. Recording errors are
// returned from the handler so the bus can log/route them; they do not
// panic the handler goroutine.
func SubscribeAudit(b bus.Bus, log *WorkspaceAuditLog) bus.Unsubscribe {
	return b.Subscribe("aps.#", func(ctx context.Context, e bus.Event) error {
		ae, ok := auditEventFromBusEvent(e)
		if !ok {
			return nil
		}
		return log.Record(ctx, ae)
	})
}

// auditEventFromBusEvent maps a bus.Event to an AuditEvent suitable for
// the workspace audit log. Returns ok=false for topics we deliberately
// don't audit (or unknown payload types so we don't drop garbage into
// the log).
func auditEventFromBusEvent(e bus.Event) (AuditEvent, bool) {
	ae := AuditEvent{
		Event:       string(e.Topic),
		Actor:       e.Source,
		Timestamp:   e.Timestamp,
		WorkspaceID: GlobalAuditWorkspace,
	}
	if ae.Actor == "" {
		ae.Actor = "aps"
	}

	switch p := e.Payload.(type) {
	case events.ProfileCreatedPayload:
		ae.Resource = "profile/" + p.ProfileID
		ae.Details = fmt.Sprintf("display=%q email=%q", p.DisplayName, p.Email)
	case events.ProfileUpdatedPayload:
		ae.Resource = "profile/" + p.ProfileID
		ae.Details = fmt.Sprintf("fields=%v", p.Fields)
	case events.ProfileDeletedPayload:
		ae.Resource = "profile/" + p.ProfileID
	case events.AdapterLinkedPayload:
		ae.Resource = "profile/" + p.ProfileID + "/adapter/" + p.AdapterType
		ae.Details = "adapter_id=" + p.AdapterID
	case events.AdapterUnlinkedPayload:
		ae.Resource = "profile/" + p.ProfileID + "/adapter/" + p.AdapterType
		ae.Details = "adapter_id=" + p.AdapterID
	case events.SessionStartedPayload:
		ae.Resource = "session/" + p.SessionID
		ae.Details = fmt.Sprintf("profile=%s command=%q pid=%d tier=%s", p.ProfileID, p.Command, p.PID, p.Tier)
	case events.SessionStoppedPayload:
		ae.Resource = "session/" + p.SessionID
		ae.Details = fmt.Sprintf("profile=%s reason=%s", p.ProfileID, p.Reason)
	default:
		return AuditEvent{}, false
	}
	return ae, true
}
