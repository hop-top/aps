package jobs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apsjobs "hop.top/aps/internal/runtime/jobs"
	"hop.top/kit/go/runtime/job"
	"hop.top/kit/go/runtime/job/mock"
)

// TestSetGet_RoundTrip verifies the package-level singleton accessor
// returns nil when nothing is wired, the most recent Set value
// otherwise, and that Set(nil) clears it. This is the contract every
// fallback path (runs_advanced.go, future sweep handlers) relies on.
func TestSetGet_RoundTrip(t *testing.T) {
	if got := apsjobs.Get(); got != nil {
		t.Fatalf("Get() with no Set should be nil, got %v", got)
	}

	svc := mock.New()
	apsjobs.Set(svc)
	t.Cleanup(func() { apsjobs.Set(nil) })

	if got := apsjobs.Get(); got == nil {
		t.Fatal("Get() after Set should not be nil")
	}

	apsjobs.Set(nil)
	if got := apsjobs.Get(); got != nil {
		t.Fatalf("Get() after Set(nil) should be nil, got %v", got)
	}
}

// TestEnqueue_FallbackWhenUnwired confirms the (id, ok) tuple returned
// by Enqueue: false when there's no service, true with a non-empty id
// when there is one. The callers in agentprotocol fan into a
// synchronous path on false, so this contract is load-bearing.
func TestEnqueue_FallbackWhenUnwired(t *testing.T) {
	apsjobs.Set(nil)

	id, ok := apsjobs.Enqueue(context.Background(), apsjobs.EnqueueOpts{
		Queue: apsjobs.QueueActions,
		Type:  apsjobs.TypeActionRun,
	})
	if ok {
		t.Fatalf("Enqueue should return ok=false with no service, got id=%q", id)
	}
	if id != "" {
		t.Fatalf("Enqueue with no service should return empty id, got %q", id)
	}
}

func TestEnqueue_ReachesService(t *testing.T) {
	svc := mock.New()
	apsjobs.Set(svc)
	t.Cleanup(func() { apsjobs.Set(nil) })

	id, ok := apsjobs.Enqueue(context.Background(), apsjobs.EnqueueOpts{
		Queue:   apsjobs.QueueActions,
		Type:    apsjobs.TypeActionRun,
		Payload: map[string]string{"profile_id": "p1", "action_id": "a1"},
	})
	if !ok {
		t.Fatal("Enqueue returned ok=false with mock service installed")
	}
	if id == "" {
		t.Fatal("Enqueue returned empty id")
	}
	if _, err := svc.Get(context.Background(), id); err != nil {
		t.Fatalf("Get after Enqueue: %v", err)
	}
}

// TestRegisterHandler_Snapshot verifies handler registration is
// process-global, snapshotted by Handlers, and that subsequent
// registrations don't mutate a snapshot already returned.
func TestRegisterHandler_Snapshot(t *testing.T) {
	apsjobs.RegisterHandler("test.one", func(context.Context, apsjobs.Job) error { return nil })
	snap := apsjobs.Handlers()
	if _, ok := snap["test.one"]; !ok {
		t.Fatalf("snapshot missing 'test.one', got keys=%v", keysOf(snap))
	}

	apsjobs.RegisterHandler("test.two", func(context.Context, apsjobs.Job) error { return nil })
	if _, ok := snap["test.two"]; ok {
		t.Fatal("snapshot mutated by later RegisterHandler — not a copy")
	}
}

// TestPollerRetry_OnHandlerFailure proves the retry contract that the
// inventory promised: a failing handler triggers re-enqueue, the next
// attempt sees Attempts incremented, and the job eventually goes to
// terminal failed when MaxAttempts is exhausted. This is the test that
// previously had no observable state in the raw goroutine path.
func TestPollerRetry_OnHandlerFailure(t *testing.T) {
	svc := mock.New()
	defer apsjobs.Set(nil)
	apsjobs.Set(svc)

	type payload struct {
		Tag string `json:"tag"`
	}

	var attempts atomic.Int32
	handler := func(_ context.Context, j apsjobs.Job) error {
		var p payload
		_ = json.Unmarshal(j.Payload, &p)
		attempts.Add(1)
		return fmt.Errorf("synthetic failure tag=%s attempt=%d", p.Tag, attempts.Load())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := svc.Enqueue(ctx, job.EnqueueOpts{
		Queue:       apsjobs.QueueActions,
		Type:        apsjobs.TypeActionRun,
		Payload:     payload{Tag: "retry-test"},
		MaxAttempts: 2,
		// Tight backoff so the test finishes in real time.
		Backoff: job.BackoffStrategy{
			Initial: 10 * time.Millisecond,
			Max:     10 * time.Millisecond,
			Factor:  1.0,
			Jitter:  0.0,
		},
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	p := &job.Poller{
		Service:  svc,
		Queue:    apsjobs.QueueActions,
		WorkerID: "test-worker",
		Interval: 5 * time.Millisecond,
		Handlers: job.HandlerMap{apsjobs.TypeActionRun: handler},
	}
	pollerCtx, pollerCancel := context.WithCancel(ctx)
	defer pollerCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Run(pollerCtx)
	}()

	deadline := time.After(3 * time.Second)
	for {
		got, err := svc.Get(ctx, id)
		if err == nil && got.Status == job.StatusFailed {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("job did not reach terminal failed within deadline (status=%v err=%v)",
				statusOf(got), err)
		case <-time.After(10 * time.Millisecond):
		}
	}

	pollerCancel()
	wg.Wait()

	finalAttempts := attempts.Load()
	if finalAttempts != 2 {
		t.Fatalf("expected exactly 2 handler invocations (MaxAttempts=2), got %d", finalAttempts)
	}

	final, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("final Get: %v", err)
	}
	if final.Status != job.StatusFailed {
		t.Fatalf("final status: want failed, got %s", final.Status)
	}
	if final.Error == "" {
		t.Fatal("expected non-empty Error on terminal job")
	}
}

func keysOf(m map[string]apsjobs.HandlerFunc) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func statusOf(j *job.Job) string {
	if j == nil {
		return "<nil>"
	}
	return string(j.Status)
}
