package agentprotocol

type CreateRunRequest struct {
	AgentID   string                 `json:"agent_id"`
	ActionID  string                 `json:"action_id"`
	Input     map[string]interface{} `json:"input,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type RunWaitRequest struct {
	AgentID  string                 `json:"agent_id,omitempty"`
	ActionID string                 `json:"action_id,omitempty"`
	Input    map[string]interface{} `json:"input,omitempty"`
	Timeout  int                    `json:"timeout,omitempty"`
	ThreadID string                 `json:"thread_id,omitempty"`
}

type RunResponse struct {
	RunID    string            `json:"run_id"`
	Status   string            `json:"status"`
	Output   string            `json:"output,omitempty"`
	ExitCode *int              `json:"exit_code,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CreateThreadRequest struct {
	AgentID  string                 `json:"agent_id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ThreadRequest struct {
	ThreadID string `json:"thread_id"`
}

type ThreadResponse struct {
	ThreadID string            `json:"thread_id"`
	AgentID  string            `json:"agent_id"`
	Metadata map[string]string `json:"metadata"`
}

type AgentSearchRequest struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type AgentSearchResponse struct {
	Agents []AgentSummary `json:"agents"`
}

type AgentSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

type AgentDetailResponse struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Capabilities []string       `json:"capabilities"`
	Schemas      []ActionSchema `json:"schemas"`
}

type ActionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
}

type StorePutRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type StoreGetRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

type StoreDeleteRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

type StoreSearchRequest struct {
	Namespace string `json:"namespace"`
	Prefix    string `json:"prefix,omitempty"`
}

type StoreItem struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

type ThreadHistoryResponse struct {
	ThreadID   string        `json:"thread_id"`
	ProfileID  string        `json:"profile_id"`
	CreatedAt  string        `json:"created_at"`
	LastSeenAt string        `json:"last_seen_at"`
	History    []RunResponse `json:"history"`
}

type ThreadRunsResponse struct {
	ThreadID string        `json:"thread_id"`
	Runs     []RunResponse `json:"runs"`
}

type StoreNamespacesResponse struct {
	Namespaces []string `json:"namespaces"`
	Count      int      `json:"count"`
}
