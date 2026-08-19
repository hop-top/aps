package msgroute

import (
	"path"
	"strings"
)

// Decision is the outcome of evaluating a sender against a Table.
type Decision struct {
	// SenderKey is the normalized key the table was evaluated with.
	SenderKey string
	// Contact is the resolved contact snapshot entry, nil when the sender is
	// not in the contacts source (or no source is configured).
	Contact *Contact
	// Route is the winning route with its profile already defaulted.
	Route Route
	// Index is the winning route's position in evaluation order (-1 when the
	// table is nil).
	Index int
	// Terminal reports that the fail-safe `match: unknown` route won.
	Terminal bool
}

// Mapping returns the canonical "profile=action" target string.
func (d Decision) Mapping() string {
	if d.Route.Profile == "" || d.Route.Action == "" {
		return ""
	}
	return d.Route.Profile + "=" + d.Route.Action
}

// Metadata renders the decision as a JSON/YAML-friendly map for stamping on
// the routed message (platform_metadata.routing) and logs.
func (d Decision) Metadata() map[string]any {
	out := map[string]any{
		"sender_key": d.SenderKey,
		"match":      d.Route.Match,
		"route":      d.Index,
		"terminal":   d.Terminal,
		"profile":    d.Route.Profile,
		"action":     d.Route.Action,
	}
	if d.Contact != nil {
		out["contact"] = map[string]any{
			"id":   d.Contact.ID,
			"name": d.Contact.Name,
			"org":  d.Contact.Org,
		}
	}
	return out
}

// Resolve evaluates senderID against the table: normalize the key, look up
// the contact (when a source is configured), then walk routes in declared
// order and return the first match. Because Load guarantees a terminal
// route, a non-nil table always yields a decision.
func (t *Table) Resolve(senderID string) Decision {
	if t == nil {
		return Decision{Index: -1}
	}
	decision := Decision{
		SenderKey: NormalizeSenderKey(t.platform, senderID),
		Index:     -1,
	}
	if t.contacts != nil && decision.SenderKey != "" {
		if contact, ok := t.contacts.LookupContact(decision.SenderKey); ok {
			decision.Contact = &contact
		}
	}
	for i, cr := range t.routes {
		if !cr.matches(decision.SenderKey, decision.Contact) {
			continue
		}
		decision.Route = cr.route
		decision.Index = i
		decision.Terminal = cr.kind == kindTerminal
		return decision
	}
	return decision
}

func (cr compiledRoute) matches(senderKey string, contact *Contact) bool {
	switch cr.kind {
	case kindTerminal:
		return true
	case kindOrg:
		if contact == nil || contact.Org == "" {
			return false
		}
		return matchPattern(cr.pattern, cr.glob, strings.ToLower(contact.Org))
	case kindContact:
		if contact == nil {
			return false
		}
		return matchPattern(cr.pattern, cr.glob, strings.ToLower(contact.ID))
	case kindSender:
		if senderKey == "" {
			return false
		}
		return matchPattern(cr.pattern, cr.glob, senderKey)
	default:
		return false
	}
}

func matchPattern(pattern string, glob bool, value string) bool {
	if !glob {
		return pattern == value
	}
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}
