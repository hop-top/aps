package agentprotocol

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"oss-aps-cli/internal/core/protocol"
)

type streamingWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan bool
}

func (sw *streamingWriter) Write(event string, data []byte) error {
	select {
	case <-sw.done:
		return io.EOF
	default:
		fmt.Fprintf(sw.w, "event: %s\n", event)
		fmt.Fprintf(sw.w, "data: %s\n\n", data)
		sw.flusher.Flush()
		return nil
	}
}

func (sw *streamingWriter) Close() error {
	close(sw.done)
	return nil
}

func (a *AgentProtocolAdapter) handleCreateRun(w http.ResponseWriter, r *http.Request) {
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

	state, err := a.core.ExecuteRun(r.Context(), input, nil)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.sendJSON(w, http.StatusCreated, RunResponse{
		RunID:    state.RunID,
		Status:   string(state.Status),
		Output:   "",
		ExitCode: state.ExitCode,
		Metadata: map[string]string{},
	})
}

func (a *AgentProtocolAdapter) handleRunWait(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RunWaitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := protocol.RunInput{
		ProfileID: req.AgentID,
		ActionID:  req.ActionID,
		ThreadID:  req.ThreadID,
	}

	if req.Input != nil {
		if payload, ok := req.Input["input"].(string); ok {
			input.Payload = []byte(payload)
		}
	}

	state, err := a.core.ExecuteRun(r.Context(), input, nil)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.sendJSON(w, http.StatusOK, RunResponse{
		RunID:    state.RunID,
		Status:   string(state.Status),
		Output:   "",
		ExitCode: state.ExitCode,
		Metadata: map[string]string{},
	})
}

func (a *AgentProtocolAdapter) handleGetRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := r.URL.Path[len("/v1/runs/"):]
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "run id required")
		return
	}

	state, err := a.core.GetRun(path)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "run not found")
		return
	}

	a.sendJSON(w, http.StatusOK, RunResponse{
		RunID:    state.RunID,
		Status:   string(state.Status),
		Output:   "",
		ExitCode: state.ExitCode,
		Metadata: map[string]string{},
	})
}

func (a *AgentProtocolAdapter) handleRunAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := r.URL.Path[len("/v1/runs/"):]
	parts := []string{}
	if path != "" {
		parts = parsePathParts(path)
	}

	if len(parts) < 2 {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	action := parts[1]

	switch action {
	case "cancel":
		a.handleRunCancelFromPath(w, r)
	default:
		a.sendError(w, http.StatusNotFound, "unknown action")
	}
}

func parsePathParts(path string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(path[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
