package agentprotocol

import (
	"encoding/json"
	"net/http"
	"strings"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/skills"
)

// SkillListResponse represents the response for listing skills
type SkillListResponse struct {
	Skills []SkillSummary `json:"skills"`
	Count  int            `json:"count"`
}

// SkillSummary represents a brief skill summary
type SkillSummary struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Scripts     []string          `json:"scripts,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SkillDetailResponse represents detailed skill information
type SkillDetailResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	License      string            `json:"license,omitempty"`
	Compatibility string           `json:"compatibility,omitempty"`
	Scripts      []string          `json:"scripts,omitempty"`
	References   []string          `json:"references,omitempty"`
	Assets       []string          `json:"assets,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	BodyContent  string            `json:"body_content,omitempty"`
}

// SkillInvokeRequest represents a request to invoke a skill
type SkillInvokeRequest struct {
	Script   string                 `json:"script"`
	Args     map[string]interface{} `json:"args,omitempty"`
	ThreadID string                 `json:"thread_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SkillInvokeResponse represents the response from skill invocation
type SkillInvokeResponse struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	Output   string `json:"output,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

// handleSkillsList handles GET /v1/skills
func (a *AgentProtocolAdapter) handleSkillsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get agent ID from query parameter
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		a.sendError(w, http.StatusBadRequest, "agent_id query parameter is required")
		return
	}

	// Load profile to verify it exists
	_, err := core.LoadProfile(agentID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Create registry with default settings
	// User paths and auto-detect will be loaded from config
	registry := skills.NewRegistry(agentID, []string{}, false)
	if err := registry.Discover(); err != nil {
		a.sendError(w, http.StatusInternalServerError, "failed to discover skills")
		return
	}

	// Convert skills to response format
	skillList := registry.List()
	summaries := make([]SkillSummary, 0, len(skillList))
	for _, skill := range skillList {
		scriptList, _ := skill.ListScripts()
		summary := SkillSummary{
			ID:          skill.Name,
			Name:        skill.Name,
			Description: skill.Description,
			Scripts:     scriptList,
			Metadata:    skill.Metadata,
		}
		summaries = append(summaries, summary)
	}

	a.sendJSON(w, http.StatusOK, SkillListResponse{
		Skills: summaries,
		Count:  len(summaries),
	})
}

// handleSkillsGet handles GET /v1/skills/{skillId}
func (a *AgentProtocolAdapter) handleSkillsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract skill ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/skills/")
	if path == "" || strings.Contains(path, "/") {
		a.sendError(w, http.StatusBadRequest, "skill id required")
		return
	}
	skillID := path

	// Get agent ID from query parameter
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		a.sendError(w, http.StatusBadRequest, "agent_id query parameter is required")
		return
	}

	// Load profile to verify it exists
	_, err := core.LoadProfile(agentID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Create registry and discover skills
	registry := skills.NewRegistry(agentID, []string{}, false)
	if err := registry.Discover(); err != nil {
		a.sendError(w, http.StatusInternalServerError, "failed to discover skills")
		return
	}

	// Get the specific skill
	skill, found := registry.Get(skillID)
	if !found {
		a.sendError(w, http.StatusNotFound, "skill not found")
		return
	}

	// Get lists from skill
	scriptList, _ := skill.ListScripts()
	refList, _ := skill.ListReferences()
	assetList, _ := skill.ListAssets()

	// Convert to response format
	detail := SkillDetailResponse{
		ID:            skill.Name,
		Name:          skill.Name,
		Description:   skill.Description,
		License:       skill.License,
		Compatibility: skill.Compatibility,
		Scripts:       scriptList,
		References:    refList,
		Assets:        assetList,
		Metadata:      skill.Metadata,
		BodyContent:   skill.BodyContent,
	}

	a.sendJSON(w, http.StatusOK, detail)
}

// handleSkillsInvoke handles POST /v1/skills/{skillId}/invoke
func (a *AgentProtocolAdapter) handleSkillsInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract skill ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/skills/")
	if !strings.Contains(path, "/invoke") {
		a.sendError(w, http.StatusBadRequest, "invalid path")
		return
	}
	skillID := strings.TrimSuffix(path, "/invoke")

	// Parse request body
	var req SkillInvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get agent ID from query parameter or request body
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		if threadID, ok := req.Metadata["agent_id"].(string); ok {
			agentID = threadID
		}
	}

	if agentID == "" {
		a.sendError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	// Load profile to verify it exists
	_, err := core.LoadProfile(agentID)
	if err != nil {
		a.sendError(w, http.StatusNotFound, "agent not found")
		return
	}

	// For now, return a placeholder response indicating the skill would be executed
	// Full execution will be implemented in Phase 5
	response := SkillInvokeResponse{
		RunID:  "run_" + skillID + "_placeholder",
		Status: "queued",
		Error:  "",
	}

	a.sendJSON(w, http.StatusAccepted, response)
}

// handleSkillsGetOrInvoke routes between GET and POST requests for /v1/skills/{skillId}
func (a *AgentProtocolAdapter) handleSkillsGetOrInvoke(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/invoke") && r.Method == http.MethodPost {
		a.handleSkillsInvoke(w, r)
	} else if r.Method == http.MethodGet {
		a.handleSkillsGet(w, r)
	} else {
		a.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
