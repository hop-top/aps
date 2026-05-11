package acp

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketTransport implements Transport for WebSocket communication
type WebSocketTransport struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	closed  bool
	readCh  chan *JSONRPCRequest
	closeCh chan struct{}
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(conn *websocket.Conn) *WebSocketTransport {
	wst := &WebSocketTransport{
		conn:    conn,
		readCh:  make(chan *JSONRPCRequest, 10),
		closeCh: make(chan struct{}),
	}

	// Start reading messages
	go wst.readLoop()

	return wst
}

// Read reads a JSON-RPC request from the WebSocket
func (wst *WebSocketTransport) Read() (*JSONRPCRequest, error) {
	select {
	case req := <-wst.readCh:
		return req, nil
	case <-wst.closeCh:
		return nil, io.EOF
	}
}

// Write writes a JSON-RPC response to the WebSocket
func (wst *WebSocketTransport) Write(response interface{}) error {
	wst.mu.Lock()
	defer wst.mu.Unlock()

	if wst.closed {
		return fmt.Errorf("transport is closed")
	}

	return wst.conn.WriteJSON(response)
}

// Close closes the WebSocket transport
func (wst *WebSocketTransport) Close() error {
	wst.mu.Lock()
	defer wst.mu.Unlock()

	if wst.closed {
		return nil
	}

	wst.closed = true
	close(wst.closeCh)

	return wst.conn.Close()
}

// readLoop continuously reads messages from the WebSocket
func (wst *WebSocketTransport) readLoop() {
	defer close(wst.readCh)

	for {
		var req JSONRPCRequest
		err := wst.conn.ReadJSON(&req)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error but continue
			}
			return
		}

		select {
		case wst.readCh <- &req:
		case <-wst.closeCh:
			return
		}
	}
}

// WebSocketServer provides WebSocket upgrade capability
type WebSocketServer struct {
	upgrader websocket.Upgrader
	handler  func(*WebSocketTransport) error
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(handler func(*WebSocketTransport) error) *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
		handler: handler,
	}
}

// ServeHTTP implements http.Handler for WebSocket upgrades
func (ws *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("WebSocket upgrade error: %v", err), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	transport := NewWebSocketTransport(conn)
	if err := ws.handler(transport); err != nil {
		// Handler error - connection will be closed
	}
}

// StartWebSocketServer starts a WebSocket server on the given address
func StartWebSocketServer(addr string, handler func(*WebSocketTransport) error) error {
	server := NewWebSocketServer(handler)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/acp" {
				server.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		}),
	}

	return httpServer.Serve(listener)
}

// MCPTool describes a tool exposed by a registered MCP server.
type MCPTool struct {
	Server      string      `json:"server,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// MCPToolProvider lists tool metadata for a registered MCP server.
type MCPToolProvider interface {
	ListMCPTools() ([]MCPTool, error)
}

// MCPToolCaller dispatches a tool call to a registered MCP server.
type MCPToolCaller interface {
	CallMCPTool(toolName string, arguments json.RawMessage) (interface{}, error)
}

// MCPServer combines the metadata and call surfaces required by MCPBridge.
type MCPServer interface {
	MCPToolProvider
	MCPToolCaller
}

// MCPToolCallerFunc adapts a function into an MCPToolCaller.
type MCPToolCallerFunc func(toolName string, arguments json.RawMessage) (interface{}, error)

// CallMCPTool calls f(toolName, arguments).
func (f MCPToolCallerFunc) CallMCPTool(toolName string, arguments json.RawMessage) (interface{}, error) {
	return f(toolName, arguments)
}

// MCPServerConfig is a simple in-process MCP server registration.
type MCPServerConfig struct {
	Type    string        `json:"type,omitempty"`
	Command string        `json:"command,omitempty"`
	Tools   []MCPTool     `json:"tools,omitempty"`
	Caller  MCPToolCaller `json:"-"`
}

// ListMCPTools returns the static tools configured for this server.
func (cfg MCPServerConfig) ListMCPTools() ([]MCPTool, error) {
	tools := make([]MCPTool, len(cfg.Tools))
	copy(tools, cfg.Tools)
	return tools, nil
}

// CallMCPTool dispatches to the configured caller.
func (cfg MCPServerConfig) CallMCPTool(toolName string, arguments json.RawMessage) (interface{}, error) {
	if cfg.Caller == nil {
		return nil, fmt.Errorf("MCP server has no tool caller")
	}
	return cfg.Caller.CallMCPTool(toolName, arguments)
}

type mcpServerRegistration struct {
	config   interface{}
	provider MCPToolProvider
	caller   MCPToolCaller
}

// MCPBridge handles integration with MCP servers.
type MCPBridge struct {
	servers map[string]mcpServerRegistration
	mu      sync.RWMutex
}

// NewMCPBridge creates a new MCP bridge
func NewMCPBridge() *MCPBridge {
	return &MCPBridge{
		servers: make(map[string]mcpServerRegistration),
	}
}

// RegisterMCPServer registers an MCP server configuration
func (mb *MCPBridge) RegisterMCPServer(name string, config interface{}) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	reg := mcpServerRegistration{config: config}
	if provider, ok := config.(MCPToolProvider); ok {
		reg.provider = provider
	}
	if caller, ok := config.(MCPToolCaller); ok {
		reg.caller = caller
	}

	mb.servers[name] = reg
	return nil
}

// ListMCPServers returns a list of registered MCP servers
func (mb *MCPBridge) ListMCPServers() []string {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	servers := make([]string, 0, len(mb.servers))
	for name := range mb.servers {
		servers = append(servers, name)
	}
	sort.Strings(servers)

	return servers
}

// GetMCPServerConfig returns the configuration for an MCP server
func (mb *MCPBridge) GetMCPServerConfig(name string) (interface{}, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	reg, exists := mb.servers[name]
	if !exists {
		return nil, fmt.Errorf("MCP server not found: %s", name)
	}

	return reg.config, nil
}

type namedMCPRegistration struct {
	name string
	reg  mcpServerRegistration
}

func (mb *MCPBridge) registrations() []namedMCPRegistration {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	regs := make([]namedMCPRegistration, 0, len(mb.servers))
	for name, reg := range mb.servers {
		regs = append(regs, namedMCPRegistration{name: name, reg: reg})
	}
	sort.Slice(regs, func(i, j int) bool {
		return regs[i].name < regs[j].name
	})
	return regs
}

// ListMCPTools returns tools available from registered MCP servers.
func (mb *MCPBridge) ListMCPTools() ([]MCPTool, error) {
	regs := mb.registrations()

	tools := make([]MCPTool, 0)
	for _, entry := range regs {
		if entry.reg.provider == nil {
			continue
		}

		serverTools, err := entry.reg.provider.ListMCPTools()
		if err != nil {
			return nil, fmt.Errorf("list MCP tools for %s: %w", entry.name, err)
		}
		for _, tool := range serverTools {
			if tool.Name == "" {
				return nil, fmt.Errorf("MCP server %s returned a tool without a name", entry.name)
			}
			tool.Server = entry.name
			tools = append(tools, tool)
		}
	}

	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Server == tools[j].Server {
			return tools[i].Name < tools[j].Name
		}
		return tools[i].Server < tools[j].Server
	})
	return tools, nil
}

// GetAvailableTools returns tools available from registered MCP servers
func (mb *MCPBridge) GetAvailableTools() []map[string]interface{} {
	registeredTools, err := mb.ListMCPTools()
	if err != nil {
		return nil
	}

	tools := make([]map[string]interface{}, 0, len(registeredTools))
	for _, tool := range registeredTools {
		item := map[string]interface{}{
			"server": tool.Server,
			"name":   tool.Name,
		}
		if tool.Description != "" {
			item["description"] = tool.Description
		}
		if tool.InputSchema != nil {
			item["inputSchema"] = tool.InputSchema
		}
		tools = append(tools, item)
	}

	return tools
}

// CallMCPTool calls a tool on an MCP server
func (mb *MCPBridge) CallMCPTool(serverName string, toolName string, arguments json.RawMessage) (interface{}, error) {
	mb.mu.RLock()
	reg, exists := mb.servers[serverName]
	mb.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("MCP server not found: %s", serverName)
	}
	if reg.provider == nil {
		return nil, fmt.Errorf("MCP server has no registered tools: %s", serverName)
	}

	tools, err := reg.provider.ListMCPTools()
	if err != nil {
		return nil, fmt.Errorf("list MCP tools for %s: %w", serverName, err)
	}
	found := false
	for _, tool := range tools {
		if tool.Name == toolName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("MCP tool not found: %s/%s", serverName, toolName)
	}
	if reg.caller == nil {
		return nil, fmt.Errorf("MCP server has no tool caller: %s", serverName)
	}

	return reg.caller.CallMCPTool(toolName, arguments)
}
