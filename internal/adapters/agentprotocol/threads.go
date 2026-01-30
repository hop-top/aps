package agentprotocol

import (
	"encoding/json"
	"net/http"
	"strings"

	"oss-aps-cli/internal/core/protocol"
)

func (a *AgentProtocolAdapter) handleThreadsSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AgentID string `json:"agent_id,omitempty"`
		Limit   int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sessions, err := a.core.ListSessions(req.AgentID)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Limit > 0 && len(sessions) > req.Limit {
		sessions = sessions[:req.Limit]
	}

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"threads": sessions,
		"count":   len(sessions),
	})
}

func (a *AgentProtocolAdapter) handleThreadHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "history" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	threadID := parts[0]

	session, err := a.core.GetSession(threadID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "thread not found")
		return
	}

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"thread_id":    session.SessionID,
		"profile_id":   session.ProfileID,
		"created_at":   session.CreatedAt,
		"last_seen_at": session.LastSeenAt,
		"metadata":     session.Metadata,
	})
}

func (a *AgentProtocolAdapter) handleThreadRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.handleThreadRunCreate(w, r)
	} else if r.Method == http.MethodGet {
		a.handleThreadRunsList(w, r)
	} else {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *AgentProtocolAdapter) handleThreadRunCreate(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "runs" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	threadID := parts[0]

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := protocol.RunInput{
		ProfileID: req.AgentID,
		ActionID:  req.ActionID,
		ThreadID:  threadID,
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

func (a *AgentProtocolAdapter) handleThreadRunsList(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "runs" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	threadID := parts[0]

	_, err := a.core.GetSession(threadID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "thread not found")
		return
	}

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"thread_id": threadID,
		"runs":      []string{},
		"message":   "run history not yet implemented",
	})
}

func (a *AgentProtocolAdapter) handleThreadDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "thread id required")
		return
	}

	err := a.core.DeleteSession(path)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *AgentProtocolAdapter) handleThreadUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "thread id required")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	metadata := make(map[string]string)
	for k, v := range req {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	err := a.core.UpdateSession(path, metadata)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	session, _ := a.core.GetSession(path)

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"thread_id": session.SessionID,
		"metadata":  session.Metadata,
	})
}
