package agentprotocol

import (
	"encoding/json"
	"net/http"
	"strings"

	"oss-aps-cli/internal/core/protocol"
)

func (a *AgentProtocolAdapter) handleRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RunWaitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sseWriter, err := NewSSEWriter(w)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	defer sseWriter.Close()

	input := protocol.RunInput{
		ProfileID: req.AgentID,
		ActionID:  req.ActionID,
	}

	if req.Input != nil {
		if payload, ok := req.Input["input"].(string); ok {
			input.Payload = []byte(payload)
		}
	}

	state, err := a.core.ExecuteRun(r.Context(), input, sseWriter)
	if err != nil {
		sseWriter.WriteEvent("error", map[string]string{"message": err.Error()})
		return
	}

	sseWriter.WriteEvent("done", map[string]interface{}{
		"run_id":    state.RunID,
		"status":    string(state.Status),
		"exit_code": state.ExitCode,
	})
}

func (a *AgentProtocolAdapter) handleRunStreamExisting(w http.ResponseWriter, r *http.Request) {
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

	sseWriter, err := NewSSEWriter(w)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	defer sseWriter.Close()

	state, err := a.core.GetRun(runID)
	if err != nil {
		sseWriter.WriteEvent("error", map[string]string{"message": "run not found"})
		return
	}

	if state.Status == protocol.RunStatusCompleted || state.Status == protocol.RunStatusFailed || state.Status == protocol.RunStatusCancelled {
		sseWriter.WriteEvent("done", map[string]interface{}{
			"run_id":    state.RunID,
			"status":    string(state.Status),
			"exit_code": state.ExitCode,
		})
		return
	}

	sseWriter.WriteEvent("running", map[string]string{
		"run_id": runID,
	})
}
