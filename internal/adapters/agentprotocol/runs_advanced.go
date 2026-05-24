package agentprotocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/protocol"
	apsjobs "hop.top/aps/internal/runtime/jobs"
)

// agentProtocolRunPayload is the wire form of an action.run job
// enqueued by handleRunsCreateBackground. Kept private to the adapter
// because only the matching handler in this file decodes it.
type agentProtocolRunPayload struct {
	ProfileID string `json:"profile_id"`
	ActionID  string `json:"action_id"`
	ThreadID  string `json:"thread_id,omitempty"`
	Input     []byte `json:"input,omitempty"`
}

func init() {
	// Register the handler at package load so the cli poller picks it
	// up alongside the session-sweep handler. The cli/jobs.go init
	// pulls the registered set via apsjobs.Handlers() before starting
	// the queue pollers.
	apsjobs.RegisterHandler(apsjobs.TypeActionRun, runActionJobHandler)
}

// runActionJobHandler is invoked by the kit poller for every
// action.run job. It hits the same core.RunAction code path the
// foreground `aps run` command uses, which is the existing source of
// truth for action execution. Kit re-runs the handler on transient
// failure up to MaxAttempts (with backoff); a returned error from this
// function is the retry trigger.
func runActionJobHandler(_ context.Context, j apsjobs.Job) error {
	var p agentProtocolRunPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return fmt.Errorf("action.run: decode payload: %w", err)
	}
	if p.ProfileID == "" || p.ActionID == "" {
		return fmt.Errorf("action.run: profile_id and action_id required")
	}
	if err := core.RunAction(p.ProfileID, p.ActionID, p.Input); err != nil {
		return fmt.Errorf("action.run: %w", err)
	}
	return nil
}

func (a *AgentProtocolAdapter) handleRunsCreateBackground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := protocol.RunInput{
		ProfileID: req.AgentID,
		ActionID:  req.ActionID,
		ThreadID:  req.SessionID,
	}

	if req.Input != nil {
		if payload, ok := req.Input["input"].(string); ok {
			input.Payload = []byte(payload)
		}
	}

	// Both branches below are fire-and-forget: this endpoint has never
	// returned a run_id, so clients cannot poll /v1/runs/{id}. The
	// preferred path enqueues a durable job (kit re-runs on transient
	// failure up to MaxAttempts); the fallback spawns an in-process
	// goroutine for library/test embeddings or APS_JOBS_DISABLE=1.
	// Keeping both branches on core.RunAction guarantees identical
	// execution semantics — the durable path is purely about restart
	// survival and retry, not about adding a tracked RunState.
	if _, ok := apsjobs.Enqueue(r.Context(), apsjobs.EnqueueOpts{
		Queue: apsjobs.QueueActions,
		Type:  apsjobs.TypeActionRun,
		Payload: agentProtocolRunPayload{
			ProfileID: input.ProfileID,
			ActionID:  input.ActionID,
			ThreadID:  input.ThreadID,
			Input:     input.Payload,
		},
		MaxAttempts: 3,
	}); !ok {
		// Fallback uses context.Background deliberately: r.Context is
		// cancelled when this handler returns (well before the spawned
		// goroutine finishes), so the request-scoped context would
		// surface as spurious cancels in the action handler.
		go func() {
			_ = core.RunAction(input.ProfileID, input.ActionID, input.Payload)
		}()
	}

	a.sendJSON(w, http.StatusAccepted, map[string]string{
		"message": "run started in background",
	})
}

func (a *AgentProtocolAdapter) handleRunsWaitExisting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "wait" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	runID := parts[0]

	state, err := a.core.GetRun(runID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "run not found")
		return
	}

	if state.Status == protocol.RunStatusPending || state.Status == protocol.RunStatusRunning {
		a.sendError(w, http.StatusAccepted, "run is still in progress")
		return
	}

	a.sendJSON(w, http.StatusOK, runResponseFromState(state))
}

func (a *AgentProtocolAdapter) handleRunsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "run id required")
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "run deletion not yet implemented",
	})
}

func (a *AgentProtocolAdapter) handleRunsStreamExisting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "stream" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	runID := parts[0]

	state, err := a.core.GetRun(runID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "run not found")
		return
	}

	if state.Status == protocol.RunStatusCompleted || state.Status == protocol.RunStatusFailed || state.Status == protocol.RunStatusCancelled {
		a.sendError(w, http.StatusBadRequest, "run has completed, cannot stream")
		return
	}

	sseWriter, err := NewSSEWriter(w)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	defer sseWriter.Close()

	sseWriter.WriteEvent("running", map[string]string{"run_id": runID})
}
