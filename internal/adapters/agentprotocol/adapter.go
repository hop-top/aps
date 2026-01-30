package agentprotocol

import (
	"encoding/json"
	"net/http"
	"strings"

	"oss-aps-cli/internal/core/protocol"
)

type AgentProtocolAdapter struct {
	core protocol.APSCore
}

func NewAgentProtocolAdapter() *AgentProtocolAdapter {
	return &AgentProtocolAdapter{}
}

func (a *AgentProtocolAdapter) Name() string {
	return "agent-protocol"
}

func (a *AgentProtocolAdapter) RegisterRoutes(mux *http.ServeMux, core protocol.APSCore) error {
	a.core = core

	mux.HandleFunc("POST /v1/runs", a.handleCreateRun)
	mux.HandleFunc("POST /v1/runs/wait", a.handleRunWait)
	mux.HandleFunc("POST /v1/runs/stream", a.handleRunStream)
	mux.HandleFunc("POST /v1/runs/background", a.handleRunsCreateBackground)
	mux.HandleFunc("POST /v1/threads", a.handleCreateThread)
	mux.HandleFunc("POST /v1/threads/search", a.handleThreadsSearch)
	mux.HandleFunc("POST /v1/agents/search", a.handleAgentSearch)
	mux.HandleFunc("GET /v1/agents/", a.handleAgentsGet)
	mux.HandleFunc("PUT /v1/store", a.handleStorePut)
	mux.HandleFunc("POST /v1/store", a.handleStorePut)
	mux.HandleFunc("GET /v1/store/", a.handleStoreGet)
	mux.HandleFunc("DELETE /v1/store/", a.handleStoreDelete)
	mux.HandleFunc("POST /v1/store/namespaces", a.handleStoreNamespaces)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/v1/runs/") && !strings.Contains(path, "/wait") && !strings.Contains(path, "/stream") && r.Method == http.MethodGet {
			a.handleGetRun(w, r)
		} else if strings.HasPrefix(path, "/v1/runs/") && !strings.Contains(path, "/cancel") && r.Method == http.MethodPost {
			a.handleRunAction(w, r)
		} else if strings.HasPrefix(path, "/v1/runs/") && strings.HasSuffix(path, "/cancel") && r.Method == http.MethodPost {
			a.handleRunCancelFromPath(w, r)
		} else if strings.HasPrefix(path, "/v1/runs/") && strings.Contains(path, "/wait") && r.Method == http.MethodGet {
			a.handleRunsWaitExisting(w, r)
		} else if strings.HasPrefix(path, "/v1/runs/") && strings.Contains(path, "/stream") && r.Method == http.MethodGet {
			a.handleRunsStreamExisting(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && strings.Contains(path, "/history") {
			a.handleThreadHistory(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && strings.Contains(path, "/runs") && r.Method == http.MethodPost {
			a.handleThreadRunCreate(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && strings.Contains(path, "/runs") && r.Method == http.MethodGet {
			a.handleThreadRunsList(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && r.Method == http.MethodGet && !strings.Contains(path, "/history") && !strings.Contains(path, "/runs") {
			a.handleThreadsGet(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && r.Method == http.MethodDelete {
			a.handleThreadsDelete(w, r)
		} else if strings.HasPrefix(path, "/v1/threads/") && r.Method == http.MethodPatch {
			a.handleThreadsUpdate(w, r)
		} else if strings.HasPrefix(path, "/v1/runs/") && r.Method == http.MethodDelete {
			a.handleRunsDelete(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	return nil
}

func (a *AgentProtocolAdapter) handleCreateThread(w http.ResponseWriter, r *http.Request) {
	var req CreateThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	metadata := make(map[string]string)
	for k, v := range req.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	session, err := a.core.CreateSession(req.AgentID, metadata)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.sendJSON(w, http.StatusCreated, ThreadResponse{
		ThreadID: session.SessionID,
		AgentID:  session.ProfileID,
		Metadata: session.Metadata,
	})
}

func (a *AgentProtocolAdapter) handleThreadsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "thread id required")
		return
	}

	session, err := a.core.GetSession(path)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "thread not found")
		return
	}

	a.sendJSON(w, http.StatusOK, ThreadResponse{
		ThreadID: session.SessionID,
		AgentID:  session.ProfileID,
		Metadata: session.Metadata,
	})
}

func (a *AgentProtocolAdapter) handleThreadsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	if path == "" || strings.Contains(path, "/history") || strings.Contains(path, "/runs") {
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

func (a *AgentProtocolAdapter) handleThreadsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/threads/")
	if path == "" || strings.Contains(path, "/history") || strings.Contains(path, "/runs") {
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

	a.sendJSON(w, http.StatusOK, ThreadResponse{
		ThreadID: session.SessionID,
		AgentID:  session.ProfileID,
		Metadata: session.Metadata,
	})
}

func (a *AgentProtocolAdapter) handleAgentSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AgentSearchRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			a.sendError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	agents, err := a.core.ListAgents()
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var result []AgentSummary
	for _, agent := range agents {
		if req.Query != "" && !strings.Contains(agent.Name, req.Query) && !strings.Contains(agent.ID, req.Query) {
			continue
		}
		result = append(result, AgentSummary{
			ID:           agent.ID,
			Name:         agent.Name,
			Description:  agent.Description,
			Capabilities: agent.Capabilities,
		})
	}

	if req.Limit > 0 && len(result) > req.Limit {
		result = result[:req.Limit]
	}

	a.sendJSON(w, http.StatusOK, AgentSearchResponse{
		Agents: result,
	})
}

func (a *AgentProtocolAdapter) handleAgentsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/agents/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "agent id required")
		return
	}

	if strings.HasPrefix(path, "schemas") {
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			a.sendError(w, http.StatusBadRequest, "invalid path")
			return
		}
		agentID := parts[0]
		a.handleGetAgentSchemas(w, r, agentID)
		return
	}

	agent, err := a.core.GetAgent(path)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "agent not found")
		return
	}

	schemas, _ := a.core.GetAgentSchemas(path)
	var schemaList []ActionSchema
	for _, s := range schemas {
		schemaList = append(schemaList, ActionSchema{
			Name:        s.Name,
			Description: s.Description,
			Input:       s.Input.(map[string]interface{}),
		})
	}

	a.sendJSON(w, http.StatusOK, AgentDetailResponse{
		ID:           agent.ID,
		Name:         agent.Name,
		Description:  agent.Description,
		Capabilities: agent.Capabilities,
		Schemas:      schemaList,
	})
}

func (a *AgentProtocolAdapter) handleGetAgentSchemas(w http.ResponseWriter, r *http.Request, agentID string) {
	schemas, err := a.core.GetAgentSchemas(agentID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "agent schemas not found")
		return
	}

	var result []ActionSchema
	for _, s := range schemas {
		result = append(result, ActionSchema{
			Name:        s.Name,
			Description: s.Description,
			Input:       s.Input.(map[string]interface{}),
		})
	}

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"agent_id": agentID,
		"schemas":  result,
	})
}

func (a *AgentProtocolAdapter) handleStorePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req StorePutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Namespace == "" {
		a.sendError(w, http.StatusBadRequest, "namespace is required")
		return
	}
	if req.Key == "" {
		a.sendError(w, http.StatusBadRequest, "key is required")
		return
	}

	err := a.core.StorePut(req.Namespace, req.Key, []byte(req.Value))
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "item stored successfully",
	})
}

func (a *AgentProtocolAdapter) handleStoreGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/store/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		a.sendError(w, http.StatusBadRequest, "namespace and key required")
		return
	}

	namespace := parts[0]
	key := parts[1]

	value, err := a.core.StoreGet(namespace, key)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "item not found")
		return
	}

	a.sendJSON(w, http.StatusOK, StoreItem{
		Namespace: namespace,
		Key:       key,
		Value:     string(value),
	})
}

func (a *AgentProtocolAdapter) handleStoreDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/store/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		a.sendError(w, http.StatusBadRequest, "namespace and key required")
		return
	}

	namespace := parts[0]
	key := parts[1]

	err := a.core.StoreDelete(namespace, key)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *AgentProtocolAdapter) handleStoreSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req StoreSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := a.core.StoreSearch(req.Namespace, req.Prefix)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]StoreItem, 0, len(result))
	for k, v := range result {
		items = append(items, StoreItem{
			Namespace: req.Namespace,
			Key:       k,
			Value:     string(v),
		})
	}

	a.sendJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	})
}

func (a *AgentProtocolAdapter) handleStoreNamespaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	namespaces, err := a.core.StoreListNamespaces()
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.sendJSON(w, http.StatusOK, StoreNamespacesResponse{
		Namespaces: namespaces,
		Count:      len(namespaces),
	})
}

func (a *AgentProtocolAdapter) handleRunCancelFromPath(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	if path == "" {
		a.sendError(w, http.StatusBadRequest, "run id required")
		return
	}

	runID := strings.TrimSuffix(path, "/cancel")
	if runID == path {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}

	err := a.core.CancelRun(r.Context(), runID)
	if err != nil {
		a.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *AgentProtocolAdapter) sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: message,
	})
}

func (a *AgentProtocolAdapter) sendJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
