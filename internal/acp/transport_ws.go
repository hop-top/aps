package acp

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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

// MCPBridge handles integration with MCP servers
type MCPBridge struct {
	servers map[string]interface{} // MCP server configurations
	mu      sync.RWMutex
}

// NewMCPBridge creates a new MCP bridge
func NewMCPBridge() *MCPBridge {
	return &MCPBridge{
		servers: make(map[string]interface{}),
	}
}

// RegisterMCPServer registers an MCP server configuration
func (mb *MCPBridge) RegisterMCPServer(name string, config interface{}) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	mb.servers[name] = config
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

	return servers
}

// GetMCPServerConfig returns the configuration for an MCP server
func (mb *MCPBridge) GetMCPServerConfig(name string) (interface{}, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	config, exists := mb.servers[name]
	if !exists {
		return nil, fmt.Errorf("MCP server not found: %s", name)
	}

	return config, nil
}

// GetAvailableTools returns tools available from registered MCP servers
func (mb *MCPBridge) GetAvailableTools() []map[string]interface{} {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	tools := make([]map[string]interface{}, 0)

	// For now, return a placeholder structure
	// In a real implementation, this would query the MCP servers
	for serverName := range mb.servers {
		tools = append(tools, map[string]interface{}{
			"server": serverName,
			"name":   "placeholder_tool",
			"description": "Tool from " + serverName,
		})
	}

	return tools
}

// CallMCPTool calls a tool on an MCP server
func (mb *MCPBridge) CallMCPTool(serverName string, toolName string, arguments json.RawMessage) (interface{}, error) {
	_, err := mb.GetMCPServerConfig(serverName)
	if err != nil {
		return nil, err
	}

	// In a real implementation, this would forward to the actual MCP server
	// For now, return a placeholder response
	return map[string]interface{}{
		"result": "Tool call placeholder for " + toolName,
		"server": serverName,
	}, nil
}
