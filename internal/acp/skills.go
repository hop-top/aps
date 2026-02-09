package acp

import (
	"oss-aps-cli/internal/skills"
)

// SkillListParams represents parameters for skill/list method
type SkillListParams struct {
	ProfileID string `json:"profileId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

// SkillGetParams represents parameters for skill/get method
type SkillGetParams struct {
	SkillID   string `json:"skillId"`
	ProfileID string `json:"profileId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

// SkillInvokeParams represents parameters for skill/invoke method
type SkillInvokeParams struct {
	SkillID   string                 `json:"skillId"`
	Script    string                 `json:"script"`
	Args      map[string]interface{} `json:"args,omitempty"`
	SessionID string                 `json:"sessionId"`
}

// SkillSummary represents a brief skill summary for ACP
type SkillSummary struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Scripts     []string          `json:"scripts,omitempty"`
	Location    string            `json:"location,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SkillDetail represents detailed skill information for ACP
type SkillDetail struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Scripts       []string          `json:"scripts,omitempty"`
	References    []string          `json:"references,omitempty"`
	Location      string            `json:"location"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	BodyContent   string            `json:"bodyContent,omitempty"`
}

// handleSkillList handles the skill/list method
func (s *Server) handleSkillList(req *JSONRPCRequest) *JSONRPCResponse {
	var params SkillListParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Determine profile ID
	profileID := params.ProfileID
	if profileID == "" {
		if params.SessionID != "" {
			// Get profile from session
			session, err := s.sessionManager.GetSession(params.SessionID)
			if err != nil {
				return s.errorResponse(req.ID, ErrSessionEnded)
			}
			profileID = session.ProfileID
		} else {
			// Use server's default profile
			profileID = s.profileID
		}
	}

	// Create registry and discover skills
	registry := skills.NewRegistry(profileID, []string{}, false)
	if err := registry.Discover(); err != nil {
		return s.errorResponse(req.ID, ErrInternalError)
	}

	// Convert to response format (filesystem-based: include location)
	skillList := registry.List()
	summaries := make([]SkillSummary, 0, len(skillList))
	for _, skill := range skillList {
		scriptList, _ := skill.ListScripts()
		summary := SkillSummary{
			ID:          skill.Name,
			Name:        skill.Name,
			Description: skill.Description,
			Scripts:     scriptList,
			Location:    skill.BasePath,
			Metadata:    skill.Metadata,
		}
		summaries = append(summaries, summary)
	}

	result := map[string]interface{}{
		"skills": summaries,
		"count":  len(summaries),
	}

	return s.successResponse(req.ID, result)
}

// handleSkillGet handles the skill/get method
func (s *Server) handleSkillGet(req *JSONRPCRequest) *JSONRPCResponse {
	var params SkillGetParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SkillID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Determine profile ID
	profileID := params.ProfileID
	if profileID == "" {
		if params.SessionID != "" {
			// Get profile from session
			session, err := s.sessionManager.GetSession(params.SessionID)
			if err != nil {
				return s.errorResponse(req.ID, ErrSessionEnded)
			}
			profileID = session.ProfileID
		} else {
			// Use server's default profile
			profileID = s.profileID
		}
	}

	// Create registry and discover skills
	registry := skills.NewRegistry(profileID, []string{}, false)
	if err := registry.Discover(); err != nil {
		return s.errorResponse(req.ID, ErrInternalError)
	}

	// Get the specific skill
	skill, found := registry.Get(params.SkillID)
	if !found {
		return s.errorResponse(req.ID, ErrResourceNotFound)
	}

	// Get lists from skill
	scriptList, _ := skill.ListScripts()
	refList, _ := skill.ListReferences()

	// Convert to response format (filesystem-based: include location)
	detail := SkillDetail{
		ID:            skill.Name,
		Name:          skill.Name,
		Description:   skill.Description,
		License:       skill.License,
		Compatibility: skill.Compatibility,
		Scripts:       scriptList,
		References:    refList,
		Location:      skill.BasePath,
		Metadata:      skill.Metadata,
		BodyContent:   skill.BodyContent,
	}

	return s.successResponse(req.ID, detail)
}

// handleSkillInvoke handles the skill/invoke method
func (s *Server) handleSkillInvoke(req *JSONRPCRequest) *JSONRPCResponse {
	var params SkillInvokeParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SkillID == "" || params.Script == "" || params.SessionID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Get session
	session, err := s.sessionManager.GetSession(params.SessionID)
	if err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	// Check permission for skill invocation
	if !session.HasPermission("skill/invoke", params.SkillID) {
		return s.errorResponse(req.ID, ErrPermissionDenied)
	}

	// For now, return a placeholder response indicating the skill would be executed
	// Full execution will be implemented in Phase 5
	result := map[string]interface{}{
		"runId":    "run_" + params.SkillID + "_" + params.Script + "_placeholder",
		"status":   "queued",
		"skillId":  params.SkillID,
		"script":   params.Script,
		"sessionId": params.SessionID,
	}

	return s.successResponse(req.ID, result)
}
