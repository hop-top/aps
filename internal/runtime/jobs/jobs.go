// Package jobs is the aps-internal shim over kit/runtime/job.
//
// It exposes the Service interface as a typed alias so call sites can
// depend on the package-local name (and the package-level setter
// pattern used elsewhere in aps for runtime singletons like the bus
// publisher) without dragging the kit import into every consumer.
//
// Wiring lives in internal/cli/jobs.go: the CLI process owns the
// durabletask Engine and the Poller(s). Library packages (core/session,
// adapters/agentprotocol) read the singleton via Get() and degrade to
// nil-checked fall-backs when the service is absent (tests, library
// embeddings, non-CLI binaries).
package jobs

import (
	"context"
	"sync"

	"hop.top/kit/go/runtime/job"
)

// Service is the kit job.Service interface re-exported under the
// aps-local name. Re-exporting keeps consumer files free of the kit
// import (mirrors how internal/events wraps runtime/bus) and gives the
// project a single point to swap backends in the future.
type Service = job.Service

// EnqueueOpts mirrors job.EnqueueOpts so callers don't import kit.
type EnqueueOpts = job.EnqueueOpts

// Job mirrors job.Job for handler signatures.
type Job = job.Job

// Queue / type identifiers reserved by aps. Centralised here so the
// CLI poller wiring and the consumer enqueue sites can't drift apart.
const (
	QueueActions = "aps.actions"
	QueueSweeps  = "aps.sweeps"

	TypeActionRun    = "action.run"
	TypeSessionSweep = "session.sweep"
)

// HandlerFunc is the signature kit/job uses for type dispatch,
// re-exported under the local name for the same reason as Service.
type HandlerFunc = func(ctx context.Context, j Job) error

var (
	mu       sync.RWMutex
	current  Service
	handlers = map[string]HandlerFunc{}
)

// Set installs the process-wide job service. Pass nil to unset (tests).
// Multiple sets are tolerated — last write wins. The setter is safe for
// concurrent reads via Get.
func Set(svc Service) {
	mu.Lock()
	current = svc
	mu.Unlock()
}

// Get returns the installed service, or nil if none. Consumers MUST
// nil-check; the in-process default for library / test paths is to
// have no service wired.
func Get() Service {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// RegisterHandler stores a handler for a job type. The CLI poller
// reads the registered set at startup via Handlers(); subsequent calls
// override (last write wins, mirrors the Set + Get pattern). Intended
// to be called from a consumer package init() so handler wiring stays
// next to the enqueue call.
func RegisterHandler(jobType string, h HandlerFunc) {
	mu.Lock()
	handlers[jobType] = h
	mu.Unlock()
}

// Handlers returns a snapshot of the registered handler map. The
// returned map is a fresh allocation so the caller can hand it to a
// kit job.Poller without worrying about subsequent RegisterHandler
// races mutating the live map.
func Handlers() map[string]HandlerFunc {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]HandlerFunc, len(handlers))
	for k, v := range handlers {
		out[k] = v
	}
	return out
}

// Enqueue is a convenience that fans through Get(). Returns
// (id, true) on success, ("", false) when no service is installed (the
// caller is responsible for a synchronous fall-back). Errors from the
// service surface as (id, false) with no log — consumers log with the
// right semantic level.
func Enqueue(ctx context.Context, opts EnqueueOpts) (string, bool) {
	svc := Get()
	if svc == nil {
		return "", false
	}
	id, err := svc.Enqueue(ctx, opts)
	if err != nil {
		return "", false
	}
	return id, true
}
