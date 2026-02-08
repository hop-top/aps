package acp

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"oss-aps-cli/internal/core/protocol"
)

// SessionMode defines how the session handles permissions
type SessionMode string

const (
	SessionModeDefault     SessionMode = "default"      // Request permissions for sensitive operations
	SessionModeAutoApprove SessionMode = "auto_approve" // Auto-approve all operations
	SessionModeReadOnly    SessionMode = "read_only"    // Deny all write operations
)

// ACPSession represents an ACP session wrapping a core session
type ACPSession struct {
	SessionID          string
	ProfileID          string
	Mode               SessionMode
	CoreSession        *protocol.SessionState
	ClientCapabilities map[string]interface{}
	AgentCapabilities  map[string]interface{}
	PermissionRules    []PermissionRule
	CreatedAt          time.Time
	LastActivity       time.Time
	mu                 sync.RWMutex
}

// PermissionRule defines a permission for a specific operation
type PermissionRule struct {
	Operation   string // e.g., "fs/write", "terminal/create"
	Allowed     bool
	PathPattern string // For fs operations
}

// ExecutionPlan tracks the execution plan for a task
type ExecutionPlan struct {
	Steps     []PlanStep `json:"entries"`
	Status    string     `json:"status"`
}

// PlanStep represents a single step in the execution plan
type PlanStep struct {
	Content  string `json:"content"`
	Priority string `json:"priority"` // "high", "medium", "low"
	Status   string `json:"status"`   // "pending", "in_progress", "completed"
}

// ToolCall represents a tool call in the plan
type ToolCall struct {
	ToolName  string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// SessionUpdate represents a notification update sent to client
type SessionUpdate struct {
	ID     string      `json:"id,omitempty"`
	Params UpdateData  `json:"params"`
	Method string      `json:"method"`
}

// UpdateData contains the actual update data
type UpdateData struct {
	SessionID string      `json:"sessionId,omitempty"`
	Status    string      `json:"status,omitempty"`
	Content   interface{} `json:"content,omitempty"`
}

// ContentBlock represents a content block in the protocol
type ContentBlock struct {
	Type     string                 `json:"type"` // "text", "image", "audio", "resource"
	Text     string                 `json:"text,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Data     string                 `json:"data,omitempty"`     // Base64 for binary
	URI      string                 `json:"uri,omitempty"`
	Extra    map[string]interface{} `json:"_meta,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  interface{}     `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID)
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// InitializeParams contains parameters for the initialize method
type InitializeParams struct {
	ProtocolVersion uint16                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
	ClientInfo      map[string]string      `json:"clientInfo,omitempty"`
}

// InitializeResult contains the result of initialize
type InitializeResult struct {
	ProtocolVersion    uint16                 `json:"protocolVersion"`
	ServerInfo         map[string]string      `json:"serverInfo,omitempty"`
	AgentCapabilities  map[string]interface{} `json:"agentCapabilities,omitempty"`
	ServerCapabilities map[string]interface{} `json:"serverCapabilities,omitempty"`
}

// SessionNewParams contains parameters for session/new
type SessionNewParams struct {
	ProfileID       string                 `json:"profileId"`
	Mode            SessionMode            `json:"mode,omitempty"`
	ClientCapabilities map[string]interface{} `json:"clientCapabilities,omitempty"`
}

// SessionNewResult contains the result of session/new
type SessionNewResult struct {
	SessionID         string                 `json:"sessionId"`
	Mode              SessionMode            `json:"mode"`
	AgentCapabilities map[string]interface{} `json:"agentCapabilities"`
}

// SessionPromptParams contains parameters for session/prompt
type SessionPromptParams struct {
	SessionID string         `json:"sessionId"`
	Messages  []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string         `json:"role"`    // "user", "assistant"
	Content []ContentBlock `json:"content"`
}

// SessionPromptResult is streamed back as notifications
type SessionPromptResult struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

// PermissionRequestParams contains parameters for permission request
type PermissionRequestParams struct {
	SessionID string `json:"sessionId"`
	Operation string `json:"operation"`
	Resource  string `json:"resource,omitempty"`
}

// TerminalCreateParams contains parameters for terminal/create
type TerminalCreateParams struct {
	SessionID      string   `json:"sessionId"`
	Command        string   `json:"command"`
	Arguments      []string `json:"arguments,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
}

// TerminalCreateResult contains result of terminal/create
type TerminalCreateResult struct {
	TerminalID string `json:"terminalId"`
	Status     string `json:"status"`
}

// generateID generates a random ID
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
