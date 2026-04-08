package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"hop.top/aps/internal/core/protocol"
)

// sessionScopedMethods lists JSON-RPC methods whose params carry a
// sessionId that identifies a live registry session. Requests for
// these methods trigger a heartbeat refresh before dispatch so
// SessionInfo.LastSeenAt reflects actual client activity.
var sessionScopedMethods = map[string]bool{
	"session/prompt":     true,
	"session/cancel":     true,
	"session/set_mode":   true,
	"session/load":       true,
	"fs/read_text_file":  true,
	"fs/write_text_file": true,
	"terminal/create":    true,
}

// Server implements the ACP (Agent Client Protocol) server
// Handles JSON-RPC 2.0 communication over stdio and other transports
type Server struct {
	profileID         string
	core              protocol.APSCore
	status            string
	mu                sync.RWMutex
	sessionManager    *SessionManager
	permissionManager *PermissionManager
	terminalManager   *TerminalManager
	transport         Transport
	ctx               context.Context
	cancel            context.CancelFunc
	initialized       bool
	protocolVer       uint16
	capabilityBuilder *CapabilityBuilder
}

// Transport defines how the ACP server communicates with clients
type Transport interface {
	Read() (*JSONRPCRequest, error)
	Write(response interface{}) error
	Close() error
}

// Verify ACP Server implements the common protocol interface
var _ protocol.ProtocolServer = (*Server)(nil)

// ACP Server manages its own server lifecycle (stdio/WebSocket)
var _ protocol.StandaloneProtocolServer = (*Server)(nil)

// NewServer creates a new ACP server for a given profile
func NewServer(profileID string, core protocol.APSCore) (*Server, error) {
	if profileID == "" {
		return nil, fmt.Errorf("profile ID cannot be empty")
	}
	if core == nil {
		return nil, fmt.Errorf("core cannot be nil")
	}

	return &Server{
		profileID:         profileID,
		core:              core,
		status:            "stopped",
		sessionManager:    NewSessionManager(),
		permissionManager: NewPermissionManager(),
		terminalManager:   NewTerminalManager(),
		protocolVer:       1,
		// Note: capabilityBuilder will be set after we load the profile
	}, nil
}

// Name returns the protocol name
func (s *Server) Name() string {
	return "acp"
}

// Start initializes and starts the ACP server
func (s *Server) Start(ctx context.Context, config interface{}) error {
	s.mu.Lock()
	if s.status == "running" {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.mu.Unlock()

	// Create context for server lifecycle
	serverCtx, cancel := context.WithCancel(ctx)
	s.ctx = serverCtx
	s.cancel = cancel

	// Create transport based on config
	transport, err := s.createTransport(config)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create transport: %w", err)
	}

	s.mu.Lock()
	s.transport = transport
	s.status = "running"
	s.mu.Unlock()

	// Start message loop in background
	go s.messageLoop()

	return nil
}

// Stop gracefully stops the ACP server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == "stopped" {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	if s.transport != nil {
		s.transport.Close()
	}

	// Release all terminals
	s.terminalManager.ReleaseAll()

	// Close all sessions (get list without holding lock)
	s.mu.Unlock()
	allSessions := s.sessionManager.ListSessions("")
	s.mu.Lock()
	for _, session := range allSessions {
		s.mu.Unlock()
		s.closeSession(session.SessionID)
		s.mu.Lock()
	}

	s.status = "stopped"
	return nil
}

// Status returns the current status of the server
func (s *Server) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// GetAddress returns the server address (empty for stdio)
func (s *Server) GetAddress() string {
	return "" // Stdio has no network address
}

// createTransport creates the appropriate transport based on configuration
func (s *Server) createTransport(config interface{}) (Transport, error) {
	// For now, default to stdio transport
	// In future, can support HTTP, WebSocket based on config
	return NewStdioTransport(os.Stdin, os.Stdout), nil
}

// messageLoop handles incoming JSON-RPC messages
func (s *Server) messageLoop() {
	defer func() {
		s.mu.Lock()
		s.status = "stopped"
		s.mu.Unlock()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Read next request from transport
		req, err := s.transport.Read()
		if err != nil {
			if err != io.EOF {
				s.sendError(nil, ErrInternalError)
			}
			return
		}

		// Handle the request
		response := s.handleRequest(req)

		// Send response if this is a request (has ID), not a notification
		if req.ID != nil {
			if err := s.transport.Write(response); err != nil {
				return
			}
		}
	}
}

// handleRequest processes a JSON-RPC request and returns a response
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	if req == nil {
		return s.errorResponse(nil, ErrInvalidParams)
	}

	// Validate JSON-RPC format
	if req.JSONRPC != "2.0" {
		return s.errorResponse(req.ID, NewErrorResponse(ErrCodeInvalidRequest, "invalid jsonrpc version", nil))
	}

	// Refresh session heartbeat for session-scoped methods. Heartbeat
	// failures do not block dispatch: the downstream handler will
	// return its own not-found error if the session is truly gone.
	if sessionScopedMethods[req.Method] {
		if sid := extractSessionID(req); sid != "" {
			if err := s.core.HeartbeatSession(sid); err != nil {
				log.Printf("acp: heartbeat for session %s (method=%s) failed: %v", sid, req.Method, err)
			}
		}
	}

	// Route to appropriate handler based on method
	switch req.Method {
	// Protocol initialization
	case "initialize":
		return s.handleInitialize(req)
	case "authenticate":
		return s.handleAuthenticate(req)

	// Session management
	case "session/new":
		return s.handleSessionNew(req)
	case "session/load":
		return s.handleSessionLoad(req)
	case "session/prompt":
		return s.handleSessionPrompt(req)
	case "session/cancel":
		return s.handleSessionCancel(req)
	case "session/set_mode":
		return s.handleSessionSetMode(req)

	// File system operations
	case "fs/read_text_file":
		return s.handleFSReadTextFile(req)
	case "fs/write_text_file":
		return s.handleFSWriteTextFile(req)

	// Terminal operations
	case "terminal/create":
		return s.handleTerminalCreate(req)
	case "terminal/output":
		return s.handleTerminalOutput(req)
	case "terminal/wait_for_exit":
		return s.handleTerminalWaitForExit(req)
	case "terminal/kill":
		return s.handleTerminalKill(req)
	case "terminal/release":
		return s.handleTerminalRelease(req)

	// Skill operations
	case "skill/list":
		return s.handleSkillList(req)
	case "skill/get":
		return s.handleSkillGet(req)
	case "skill/invoke":
		return s.handleSkillInvoke(req)

	default:
		return s.errorResponse(req.ID, ErrMethodNotFound)
	}
}

// handleInitialize processes the initialize method
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	var params InitializeParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return s.errorResponse(req.ID, NewErrorResponse(ErrCodeInvalidRequest, "already initialized", nil))
	}

	// For now, accept any protocol version. In future, enforce compatibility.
	s.initialized = true

	result := InitializeResult{
		ProtocolVersion: s.protocolVer,
		ServerInfo: map[string]string{
			"name":    "APS-ACP",
			"version": "0.1.0",
		},
		AgentCapabilities: s.buildAgentCapabilities(),
	}

	return s.successResponse(req.ID, result)
}

// handleAuthenticate processes the authenticate method
func (s *Server) handleAuthenticate(req *JSONRPCRequest) *JSONRPCResponse {
	// For now, always succeed
	// In future, implement actual authentication
	return s.successResponse(req.ID, map[string]bool{"authenticated": true})
}

// handleSessionNew creates a new session
func (s *Server) handleSessionNew(req *JSONRPCRequest) *JSONRPCResponse {
	var params SessionNewParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Use provided profile ID or default to server's profile
	profileID := params.ProfileID
	if profileID == "" {
		profileID = s.profileID
	}

	// Create underlying APS session
	metadata := make(map[string]string)
	if params.Mode == "" {
		params.Mode = SessionModeDefault
	}
	metadata["acp_mode"] = string(params.Mode)

	coreSession, err := s.core.CreateSession(profileID, metadata)
	if err != nil {
		return s.errorResponse(req.ID, ErrInternalError)
	}

	// Build agent capabilities
	agentCaps := s.buildAgentCapabilities()
	if params.Mode != "" {
		agentCaps = FilterCapabilities(agentCaps, params.Mode)
	}

	// Create ACP session with manager
	acpSession := s.sessionManager.CreateSession(
		coreSession.SessionID,
		profileID,
		params.Mode,
		params.ClientCapabilities,
		coreSession,
	)
	acpSession.AgentCapabilities = agentCaps

	result := SessionNewResult{
		SessionID:         acpSession.SessionID,
		Mode:              acpSession.Mode,
		AgentCapabilities: acpSession.AgentCapabilities,
	}

	return s.successResponse(req.ID, result)
}

// handleSessionLoad loads an existing session
func (s *Server) handleSessionLoad(req *JSONRPCRequest) *JSONRPCResponse {
	return s.errorResponse(req.ID, ErrNotImplemented)
}

// handleSessionPrompt processes a user prompt
func (s *Server) handleSessionPrompt(req *JSONRPCRequest) *JSONRPCResponse {
	return s.errorResponse(req.ID, ErrNotImplemented)
}

// handleSessionCancel cancels a session operation
func (s *Server) handleSessionCancel(req *JSONRPCRequest) *JSONRPCResponse {
	return s.errorResponse(req.ID, ErrNotImplemented)
}

// handleSessionSetMode sets the session mode
func (s *Server) handleSessionSetMode(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string      `json:"sessionId"`
		Mode      SessionMode `json:"mode"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SessionID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Update mode
	if err := s.sessionManager.SetSessionMode(params.SessionID, params.Mode); err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	// Get updated session
	session, err := s.sessionManager.GetSession(params.SessionID)
	if err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	result := map[string]interface{}{
		"sessionId": params.SessionID,
		"mode":      session.Mode,
	}

	return s.successResponse(req.ID, result)
}

// handleFSReadTextFile reads a text file
func (s *Server) handleFSReadTextFile(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"sessionId"`
		Path      string `json:"path"`
		StartLine int    `json:"startLine,omitempty"`
		EndLine   int    `json:"endLine,omitempty"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SessionID == "" || params.Path == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Get session
	session, err := s.sessionManager.GetSession(params.SessionID)
	if err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	// Check permission
	if !session.HasPermission("fs/read_text_file", params.Path) {
		return s.errorResponse(req.ID, ErrPermissionDenied)
	}

	// Read file
	fsh := NewFileSystemHandler(os.TempDir(), MaxFileSize)
	content, err := fsh.ReadTextFile(params.Path, params.StartLine, params.EndLine)
	if err != nil {
		return s.errorResponse(req.ID, NewErrorResponse(ErrCodeResourceNotFound, err.Error(), nil))
	}

	result := map[string]interface{}{
		"path":    params.Path,
		"content": content,
		"size":    len(content),
	}

	return s.successResponse(req.ID, result)
}

// handleFSWriteTextFile writes a text file
func (s *Server) handleFSWriteTextFile(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"sessionId"`
		Path      string `json:"path"`
		Content   string `json:"content"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SessionID == "" || params.Path == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Get session
	session, err := s.sessionManager.GetSession(params.SessionID)
	if err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	// Check permission
	if !session.HasPermission("fs/write_text_file", params.Path) {
		return s.errorResponse(req.ID, ErrPermissionDenied)
	}

	// Write file
	fsh := NewFileSystemHandler(os.TempDir(), MaxFileSize)
	if err := fsh.WriteTextFile(params.Path, params.Content); err != nil {
		return s.errorResponse(req.ID, NewErrorResponse(ErrCodeInternalError, err.Error(), nil))
	}

	result := map[string]interface{}{
		"path":    params.Path,
		"size":    len(params.Content),
		"success": true,
	}

	return s.successResponse(req.ID, result)
}

// handleTerminalCreate creates a terminal
func (s *Server) handleTerminalCreate(req *JSONRPCRequest) *JSONRPCResponse {
	var params TerminalCreateParams
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.SessionID == "" || params.Command == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Get session
	session, err := s.sessionManager.GetSession(params.SessionID)
	if err != nil {
		return s.errorResponse(req.ID, ErrSessionEnded)
	}

	// Check permission
	if !session.HasPermission("terminal/create", params.Command) {
		return s.errorResponse(req.ID, ErrPermissionDenied)
	}

	// Create terminal
	terminal, err := s.terminalManager.CreateTerminal(
		params.Command,
		params.Arguments,
		params.WorkingDirectory,
		params.Environment,
	)
	if err != nil {
		return s.errorResponse(req.ID, NewErrorResponse(ErrCodeInternalError, err.Error(), nil))
	}

	result := TerminalCreateResult{
		TerminalID: terminal.ID,
		Status:     terminal.Status,
	}

	return s.successResponse(req.ID, result)
}

// handleTerminalOutput gets terminal output
func (s *Server) handleTerminalOutput(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		TerminalID string `json:"terminalId"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.TerminalID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Get output
	output, err := s.terminalManager.GetOutput(params.TerminalID)
	if err != nil {
		return s.errorResponse(req.ID, ErrResourceNotFound)
	}

	result := map[string]interface{}{
		"terminalId": params.TerminalID,
		"output":     output,
	}

	return s.successResponse(req.ID, result)
}

// handleTerminalWaitForExit waits for terminal exit
func (s *Server) handleTerminalWaitForExit(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		TerminalID string `json:"terminalId"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.TerminalID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Wait for exit
	exitCode, err := s.terminalManager.WaitForExit(params.TerminalID)
	if err != nil {
		return s.errorResponse(req.ID, ErrResourceNotFound)
	}

	result := map[string]interface{}{
		"terminalId": params.TerminalID,
		"exitCode":   exitCode,
	}

	return s.successResponse(req.ID, result)
}

// handleTerminalKill kills a terminal
func (s *Server) handleTerminalKill(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		TerminalID string `json:"terminalId"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.TerminalID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Kill terminal
	if err := s.terminalManager.Kill(params.TerminalID); err != nil {
		return s.errorResponse(req.ID, ErrResourceNotFound)
	}

	result := map[string]interface{}{
		"terminalId": params.TerminalID,
		"status":     "killed",
	}

	return s.successResponse(req.ID, result)
}

// handleTerminalRelease releases terminal resources
func (s *Server) handleTerminalRelease(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		TerminalID string `json:"terminalId"`
	}
	if err := parseParams(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	if params.TerminalID == "" {
		return s.errorResponse(req.ID, ErrInvalidParams)
	}

	// Release terminal
	if err := s.terminalManager.Release(params.TerminalID); err != nil {
		return s.errorResponse(req.ID, ErrResourceNotFound)
	}

	result := map[string]interface{}{
		"terminalId": params.TerminalID,
		"status":     "released",
	}

	return s.successResponse(req.ID, result)
}

// closeSession closes a session
func (s *Server) closeSession(sessionID string) error {
	// Clear pending permission requests
	s.permissionManager.ClearPendingRequests(sessionID)

	// Delete from session manager
	if err := s.sessionManager.DeleteSession(sessionID); err != nil {
		return err
	}

	// Delete core session
	return s.core.DeleteSession(sessionID)
}

// buildAgentCapabilities builds the agent capabilities for this profile
func (s *Server) buildAgentCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"filesystem": map[string]bool{
			"readTextFile":  true,
			"writeTextFile": true,
		},
		"terminal": map[string]bool{
			"create":      true,
			"output":      true,
			"waitForExit": true,
			"kill":        true,
			"release":     true,
		},
		"sessionModes": []string{"default", "auto_approve", "read_only"},
		"contentTypes": []string{"text", "image", "audio", "resource"},
	}
}

// successResponse creates a JSON-RPC success response
func (s *Server) successResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// errorResponse creates a JSON-RPC error response
func (s *Server) errorResponse(id interface{}, err *ErrorResponse) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
}

// sendError sends an error notification (no ID)
func (s *Server) sendError(id interface{}, err *ErrorResponse) error {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
	return s.transport.Write(resp)
}

// GetSessionManager returns the session manager
func (s *Server) GetSessionManager() *SessionManager {
	return s.sessionManager
}

// SessionManager returns the session manager (convenience alias)
func (s *Server) SessionManager() *SessionManager {
	return s.sessionManager
}

// GetPermissionManager returns the permission manager
func (s *Server) GetPermissionManager() *PermissionManager {
	return s.permissionManager
}

// PermissionManager returns the permission manager (convenience alias)
func (s *Server) PermissionManager() *PermissionManager {
	return s.permissionManager
}

// GetTerminalManager returns the terminal manager
func (s *Server) GetTerminalManager() *TerminalManager {
	return s.terminalManager
}

// TerminalManager returns the terminal manager (convenience alias)
func (s *Server) TerminalManager() *TerminalManager {
	return s.terminalManager
}

// IsInitialized returns whether the server is initialized
func (s *Server) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// extractSessionID pulls a "sessionId" string field out of a request's
// params payload. Returns empty string if params is nil, not an
// object, or lacks a sessionId. Does not error — callers use the
// empty string to mean "no session scope".
func extractSessionID(req *JSONRPCRequest) string {
	if req == nil || req.Params == nil {
		return ""
	}
	var probe struct {
		SessionID string `json:"sessionId"`
	}
	_ = parseParams(req.Params, &probe)
	return probe.SessionID
}

// parseParams unmarshals params into a target struct
func parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	// Convert params to JSON bytes then unmarshal
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, target)
}

// StdioTransport implements Transport for stdin/stdout communication
type StdioTransport struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(stdin io.Reader, stdout io.Writer) *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(stdin),
		writer: bufio.NewWriter(stdout),
	}
}

// Read reads a JSON-RPC request from stdin
func (t *StdioTransport) Read() (*JSONRPCRequest, error) {
	var req JSONRPCRequest

	// Read one line at a time until we get valid JSON
	line, err := t.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse JSON
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &req, nil
}

// Write writes a JSON-RPC response to stdout
func (t *StdioTransport) Write(response interface{}) error {
	encoder := json.NewEncoder(t.writer)
	if err := encoder.Encode(response); err != nil {
		return err
	}
	return t.writer.Flush()
}

// Close closes the stdio transport
func (t *StdioTransport) Close() error {
	return t.writer.Flush()
}
