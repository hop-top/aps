package acp

import (
	"oss-aps-cli/internal/core"
)

// CapabilityBuilder builds agent capabilities based on profile and session configuration
type CapabilityBuilder struct {
	profile *core.Profile
}

// NewCapabilityBuilder creates a new capability builder
func NewCapabilityBuilder(profile *core.Profile) *CapabilityBuilder {
	return &CapabilityBuilder{
		profile: profile,
	}
}

// BuildAgentCapabilities builds the complete agent capabilities map
func (cb *CapabilityBuilder) BuildAgentCapabilities() map[string]interface{} {
	caps := map[string]interface{}{
		// File system capabilities
		"filesystem": map[string]interface{}{
			"readTextFile": map[string]interface{}{
				"supported": true,
				"maxSize":   10 * 1024 * 1024, // 10MB
			},
			"writeTextFile": map[string]interface{}{
				"supported": true,
				"maxSize":   10 * 1024 * 1024,
			},
		},

		// Terminal capabilities
		"terminal": map[string]interface{}{
			"create": map[string]interface{}{
				"supported": true,
				"shells":    []string{"bash", "sh", "zsh"},
			},
			"interactive": map[string]interface{}{
				"supported": true,
				"timeout":   300, // 5 minutes
			},
			"output": map[string]interface{}{
				"supported": true,
				"streaming": true,
			},
		},

		// Session capabilities
		"session": map[string]interface{}{
			"modes": []string{
				"default",      // Request permissions for sensitive ops
				"auto_approve", // Auto-approve all
				"read_only",    // Deny writes
			},
			"loadSession": cb.profile != nil, // Support session loading
		},

		// Content capabilities
		"content": map[string]interface{}{
			"types": []string{"text", "image", "audio", "resource"},
			"image": map[string]interface{}{
				"supported": true,
				"formats":   []string{"png", "jpg", "gif", "webp"},
			},
		},

		// Server info
		"server": map[string]interface{}{
			"name":           "APS-ACP",
			"version":        "0.1.0",
			"protocolVersion": 1,
		},
	}

	// Add profile-specific capabilities if available
	if cb.profile != nil {
		caps["profile"] = map[string]interface{}{
			"id":           cb.profile.ID,
			"displayName":  cb.profile.DisplayName,
			"capabilities": cb.profile.Capabilities,
		}

		// Add isolation-specific capabilities
		if cb.profile.Isolation.Level == core.IsolationContainer {
			caps["isolation"] = map[string]interface{}{
				"level": "container",
			}
		}
	}

	return caps
}

// BuildClientCapabilities builds default client capabilities for responses
func (cb *CapabilityBuilder) BuildClientCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"filesystem": map[string]bool{
			"readTextFile":  true,
			"writeTextFile": true,
		},
		"terminal": true,
	}
}

// FilterCapabilities filters capabilities based on session mode
func FilterCapabilities(caps map[string]interface{}, mode SessionMode) map[string]interface{} {
	filtered := make(map[string]interface{})

	// Copy map
	for k, v := range caps {
		filtered[k] = v
	}

	// Apply mode-specific filtering
	if mode == SessionModeReadOnly {
		// Remove write capabilities
		if fsMap, ok := filtered["filesystem"].(map[string]interface{}); ok {
			delete(fsMap, "writeTextFile")
			filtered["filesystem"] = fsMap
		}

		// Disable terminal creation
		if termMap, ok := filtered["terminal"].(map[string]interface{}); ok {
			delete(termMap, "create")
			filtered["terminal"] = termMap
		}
	}

	return filtered
}

// CapabilityRequest represents a client's capability request
type CapabilityRequest struct {
	Filesystem map[string]bool `json:"filesystem,omitempty"`
	Terminal   bool            `json:"terminal,omitempty"`
	MCP        bool            `json:"mcp,omitempty"`
}

// NegotiateCapabilities negotiates capabilities based on client request
func NegotiateCapabilities(agentCaps map[string]interface{}, clientReq CapabilityRequest) map[string]interface{} {
	negotiated := make(map[string]interface{})

	// Negotiate filesystem capabilities
	if clientReq.Filesystem != nil {
		if fsAgent, ok := agentCaps["filesystem"].(map[string]interface{}); ok {
			fsNeg := make(map[string]interface{})
			for capName, supported := range clientReq.Filesystem {
				if supported {
					if agentSupports, ok := fsAgent[capName]; ok {
						fsNeg[capName] = agentSupports
					}
				}
			}
			if len(fsNeg) > 0 {
				negotiated["filesystem"] = fsNeg
			}
		}
	}

	// Negotiate terminal capabilities
	if clientReq.Terminal {
		if termAgent, ok := agentCaps["terminal"].(map[string]interface{}); ok {
			negotiated["terminal"] = termAgent
		}
	}

	// Add server info always
	if serverInfo, ok := agentCaps["server"].(map[string]interface{}); ok {
		negotiated["server"] = serverInfo
	}

	return negotiated
}
