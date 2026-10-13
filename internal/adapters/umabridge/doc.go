// Package umabridge hosts a hop.top/uma engine in-process behind an aps
// message service of adapter "uma".
//
// It is the only package that imports hop.top/uma: dropping the uma
// dependency from go.mod means deleting this package and the `replace` block
// that points at it.
//
// One Bridge serves one service. It owns an engine (memory store, an
// authorizer that admits only the bridge's own two actors, credentials
// resolved from the service's secret bindings, an HTTP client), the channel
// and inbox of the chosen uma adapter, and a bot companion subscribed to
// message.received. Each received message is normalized into an aps
// NormalizedMessage and run through messenger.Runtime, so routing, action
// execution, reply policy and history stay aps's. The action's reply goes
// back as a uma Messages.Reply, which the engine's delivery worker sends
// through the provider adapter.
//
// Not here: mounting the provider webhook at a URL (Bridge.Webhook is the
// seam; the messenger adapter mounts it at the service's webhook route), and
// mapping uma conversations onto aps session history.
package umabridge
