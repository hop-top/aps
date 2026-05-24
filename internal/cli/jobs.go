package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/logging"
	apsjobs "hop.top/aps/internal/runtime/jobs"
	"hop.top/kit/go/core/xdg"
	"hop.top/kit/go/runtime/job"
	"hop.top/kit/go/runtime/job/durabletask"
)

// pollerWorkerID identifies this aps process inside the durabletask
// worker registry. Hostname + pid is plenty: aps is a single-node
// adopter; cross-process coordination is the durabletask DB itself.
func pollerWorkerID() string {
	hn, err := os.Hostname()
	if err != nil || hn == "" {
		hn = appName
	}
	return fmt.Sprintf("%s:%d", hn, os.Getpid())
}

// appName mirrors the literal aps consumers pass to xdg.DataDir. Kept
// as a file-local const so this file's worker-id default and data-dir
// lookup can never drift.
const appName = "aps"

var (
	jobEngine   *durabletask.Engine
	jobCancel   context.CancelFunc
	jobWG       sync.WaitGroup
	jobInitOnce sync.Once
)

// jobsEnabled reports whether the durabletask runtime should be wired
// for this invocation. CLI commands that are short-lived and do not
// need to drain background work can set APS_JOBS_DISABLE=1 to skip the
// open(); this keeps `aps --help` and unit tests free of DB allocation.
func jobsEnabled() bool {
	return os.Getenv("APS_JOBS_DISABLE") != "1"
}

// initJobs opens the durabletask SQLite backend, registers handlers,
// and starts the Poller goroutines. Safe to call multiple times; only
// the first invocation does anything. Errors are logged but do not
// fail the CLI invocation — the service is best-effort, callers fall
// back to synchronous paths when apsjobs.Get() returns nil.
//
// The lifecycle goroutine here is the ONLY production goroutine spawned
// by this file; it owns the kit-job Poller loops, which is the entire
// point of the refactor. All other ad-hoc goroutines elsewhere in aps
// either remain idiomatic (cmd.Wait, http.Server.Serve) or are
// migrated to enqueue + handler in subsequent commits.
func initJobs() {
	jobInitOnce.Do(func() {
		if !jobsEnabled() {
			return
		}
		// Disable the legacy in-process ticker reaper before any caller
		// in this process touches session.GetRegistry. The reaper races
		// the kit-poller sweep otherwise; one of them has to win, and
		// the kit version is what this refactor exists to deliver.
		session.DisableInlineReaper()

		// Register internal handlers that live here so they're present
		// before the poller starts polling. Consumer packages register
		// their own handlers via apsjobs.RegisterHandler from init().
		apsjobs.RegisterHandler(apsjobs.TypeSessionSweep, sessionSweepHandler)

		dataDir, err := xdg.DataDir(appName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: jobs runtime: xdg data dir: %v\n", err)
			return
		}
		if err := os.MkdirAll(dataDir, 0o750); err != nil {
			fmt.Fprintf(os.Stderr, "warn: jobs runtime: mkdir %s: %v\n", dataDir, err)
			return
		}
		dbPath := filepath.Join(dataDir, "jobs.db")

		eng, err := durabletask.New(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: jobs runtime: open %s: %v\n", dbPath, err)
			return
		}
		jobEngine = eng
		apsjobs.Set(eng)

		ctx, cancel := context.WithCancel(context.Background())
		jobCancel = cancel

		handlerSnapshot := apsjobs.Handlers()
		actionsHandlers := pickHandlers(handlerSnapshot, apsjobs.TypeActionRun)
		sweepsHandlers := pickHandlers(handlerSnapshot, apsjobs.TypeSessionSweep)

		startPoller(ctx, eng, apsjobs.QueueActions, actionsHandlers)
		startPoller(ctx, eng, apsjobs.QueueSweeps, sweepsHandlers)

		seedSessionSweep(ctx, eng)
	})
}

// pickHandlers narrows the global handler map down to the subset that
// matches a given queue's job types. Keeps each Poller's HandlerMap
// scoped — the poller fails a job whose type isn't routable, so we
// must NOT register an unrelated handler under a queue that won't
// receive it.
func pickHandlers(all map[string]apsjobs.HandlerFunc, types ...string) job.HandlerMap {
	out := job.HandlerMap{}
	for _, t := range types {
		if h, ok := all[t]; ok {
			out[t] = h
		}
	}
	return out
}

// startPoller launches a kit Poller for the given queue.
//
// The poller blocks until ctx cancellation. We track each Poller on
// jobWG so drainJobs can wait for in-flight handlers to finish before
// closing the engine (which would invalidate any pending Complete/Fail
// call from a still-running handler).
func startPoller(ctx context.Context, svc job.Service, queue string, handlers job.HandlerMap) {
	if len(handlers) == 0 {
		// Skip the poller entirely if nothing is registered for this
		// queue — otherwise the loop runs a Claim every interval for
		// no productive reason.
		return
	}
	p := &job.Poller{
		Service:  svc,
		Interval: 500 * time.Millisecond,
		Queue:    queue,
		WorkerID: pollerWorkerID(),
		Handlers: handlers,
	}
	jobWG.Add(1)
	go func() {
		defer jobWG.Done()
		_ = p.Run(ctx)
	}()
}

// drainJobs stops the poller goroutines and closes the engine. Mirrors
// drainBus: safe to call when nothing was wired (jobsEnabled was false
// or initJobs failed early). Called from Execute's defer chain so the
// engine flushes even on error paths.
func drainJobs() {
	if jobCancel != nil {
		jobCancel()
	}
	jobWG.Wait()
	if jobEngine != nil {
		_ = jobEngine.Close()
		jobEngine = nil
	}
	apsjobs.Set(nil)
	jobInitOnce = sync.Once{}
}

// sessionSweepHandler runs one tick of the session reaper. After each
// successful sweep it re-enqueues itself with ScheduledAt set to
// now + ReaperTickInterval, giving aps the same periodic behaviour as
// the original time.Ticker without a long-lived goroutine in
// core/session.
//
// A handler failure does NOT re-enqueue; the kit backoff scheduler
// will retry the failed job up to MaxAttempts. This is intentional —
// we'd rather have one stuck sweep dead-letter loudly than silently
// double up future sweeps.
func sessionSweepHandler(ctx context.Context, _ apsjobs.Job) error {
	r := session.GetRegistry()
	expired, err := r.CleanupInactive(session.DefaultTimeout)
	if err != nil {
		return fmt.Errorf("session.sweep: %w", err)
	}
	if len(expired) > 0 {
		logging.GetLogger().Info("session reaper: removed inactive sessions",
			"count", len(expired),
			"ids", expired,
		)
	}
	return reenqueueSweep(ctx)
}

// seedSessionSweep schedules the first sweep relative to process start
// so a freshly-launched aps process doesn't tight-loop on an empty
// registry the moment the poller wakes.
func seedSessionSweep(ctx context.Context, svc job.Service) {
	at := time.Now().Add(session.ReaperTickInterval)
	_, _ = svc.Enqueue(ctx, job.EnqueueOpts{
		Queue:       apsjobs.QueueSweeps,
		Type:        apsjobs.TypeSessionSweep,
		ScheduledAt: &at,
		MaxAttempts: 3,
	})
}

// reenqueueSweep is invoked at the end of every successful sweep
// handler. The new job is scheduled for ReaperTickInterval in the
// future; ScheduledAt is what makes durabletask honour the delay.
func reenqueueSweep(ctx context.Context) error {
	svc := apsjobs.Get()
	if svc == nil {
		return nil
	}
	at := time.Now().Add(session.ReaperTickInterval)
	if _, err := svc.Enqueue(ctx, job.EnqueueOpts{
		Queue:       apsjobs.QueueSweeps,
		Type:        apsjobs.TypeSessionSweep,
		ScheduledAt: &at,
		MaxAttempts: 3,
	}); err != nil {
		return fmt.Errorf("re-enqueue session sweep: %w", err)
	}
	return nil
}
