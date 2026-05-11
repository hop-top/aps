package agentprotocol

import (
	"encoding/json"
	"net/http"
	"strings"

	"hop.top/aps/internal/core/protocol"
)

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

	go func() {
		_, _ = a.core.ExecuteRun(r.Context(), input, nil)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
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
