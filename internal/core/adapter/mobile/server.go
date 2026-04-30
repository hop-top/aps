package mobile

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"hop.top/aps/internal/logging"
)

const (
	DefaultPort          = 8443
	HeartbeatInterval    = 30 * time.Second
	WriteWait            = 10 * time.Second
	PongWait             = 60 * time.Second
	MaxMessageSize       = 512 * 1024 // 512KB
)

// AdapterServer manages the HTTP + WebSocket server for mobile adapter connections.
// It implements the StandaloneProtocolServer pattern used by A2A and ACP.
type AdapterServer struct {
	mu       sync.RWMutex
	registry *Registry
	tokenMgr *TokenManager
	server   *http.Server
	listener net.Listener
	status   string
	addr     string
	conns    map[string]*AdapterConnection

	// Pairing state
	pairingMu    sync.RWMutex
	pairingCodes map[string]*PairingSession

	// TLS
	tlsCert *tls.Certificate

	// Config
	profileID       string
	approvalRequired bool
	maxAdapters       int
}

// PairingSession tracks an active pairing code
type PairingSession struct {
	Code         string
	ProfileID    string
	Capabilities []string
	ExpiresAt    time.Time
	Used         bool
}

// AdapterConnection tracks an active WebSocket connection
type AdapterConnection struct {
	conn     *websocket.Conn
	device   *MobileAdapter
	mu       sync.Mutex
	closeCh  chan struct{}
	closed   bool
}

// NewAdapterServer creates a new device server
func NewAdapterServer(profileID string, registry *Registry, tokenMgr *TokenManager, opts ...ServerOption) *AdapterServer {
	s := &AdapterServer{
		registry:     registry,
		tokenMgr:     tokenMgr,
		profileID:    profileID,
		status:       "stopped",
		conns:        make(map[string]*AdapterConnection),
		pairingCodes: make(map[string]*PairingSession),
		maxAdapters:   10,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ServerOption configures the device server
type ServerOption func(*AdapterServer)

func WithApprovalRequired(required bool) ServerOption {
	return func(s *AdapterServer) { s.approvalRequired = required }
}

func WithMaxAdapters(max int) ServerOption {
	return func(s *AdapterServer) { s.maxAdapters = max }
}

func WithTLSCert(cert *tls.Certificate) ServerOption {
	return func(s *AdapterServer) { s.tlsCert = cert }
}

// --- StandaloneProtocolServer interface ---

func (s *AdapterServer) Name() string { return "mobile-adapter" }

func (s *AdapterServer) Start(ctx context.Context, config any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == "running" {
		return fmt.Errorf("server already running")
	}

	addr := fmt.Sprintf(":%d", DefaultPort)
	if config != nil {
		if a, ok := config.(string); ok && a != "" {
			addr = a
		}
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // disabled for WebSocket
		IdleTimeout:  60 * time.Second,
	}

	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return ErrPortInUse(DefaultPort, err)
	}

	s.addr = s.listener.Addr().String()
	s.status = "running"

	go func() {
		var serveErr error
		if s.tlsCert != nil {
			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{*s.tlsCert},
				MinVersion:   tls.VersionTLS12,
			}
			tlsListener := tls.NewListener(s.listener, tlsConfig)
			serveErr = s.server.Serve(tlsListener)
		} else {
			serveErr = s.server.Serve(s.listener)
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			logging.GetLogger().Error("adapter server error", serveErr)
		}
		s.mu.Lock()
		s.status = "stopped"
		s.mu.Unlock()
	}()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

func (s *AdapterServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != "running" {
		return nil
	}

	// Close all WebSocket connections
	for id, conn := range s.conns {
		conn.Close()
		delete(s.conns, id)
	}

	// Invalidate all pairing codes
	s.pairingMu.Lock()
	s.pairingCodes = make(map[string]*PairingSession)
	s.pairingMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.status = "stopping"
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	s.status = "stopped"
	return nil
}

func (s *AdapterServer) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *AdapterServer) GetAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

// --- Pairing management ---

// RegisterPairingCode creates a new pairing session
func (s *AdapterServer) RegisterPairingCode(code string, capabilities []string, expiry time.Duration) {
	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()

	s.pairingCodes[code] = &PairingSession{
		Code:         code,
		ProfileID:    s.profileID,
		Capabilities: capabilities,
		ExpiresAt:    time.Now().Add(expiry),
	}
}

// ActiveConnections returns the number of active WebSocket connections
func (s *AdapterServer) ActiveConnections() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.conns)
}

// --- HTTP routes ---

func (s *AdapterServer) registerRoutes(mux *http.ServeMux) {
	prefix := fmt.Sprintf("/aps/adapter/%s", s.profileID)

	mux.HandleFunc(prefix+"/health", s.handleHealth)
	mux.HandleFunc(prefix+"/pair", s.handlePair)
	mux.HandleFunc(prefix+"/ws", s.handleWebSocket)
}

func (s *AdapterServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      s.Status(),
		"profile":     s.profileID,
		"connections": s.ActiveConnections(),
	})
}

func (s *AdapterServer) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PairingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate pairing code
	s.pairingMu.Lock()
	session, exists := s.pairingCodes[req.PairingCode]
	if !exists {
		s.pairingMu.Unlock()
		writeJSONError(w, "invalid pairing code", http.StatusUnauthorized)
		return
	}
	if session.Used {
		s.pairingMu.Unlock()
		writeJSONError(w, "pairing code already used", http.StatusConflict)
		return
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.pairingCodes, req.PairingCode)
		s.pairingMu.Unlock()
		writeJSONError(w, "pairing code expired", http.StatusGone)
		return
	}
	session.Used = true
	s.pairingMu.Unlock()

	// Check max devices
	activeCount, err := s.registry.CountActive(s.profileID)
	if err != nil {
		writeJSONError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if activeCount >= s.maxAdapters {
		writeJSONError(w, fmt.Sprintf("maximum adapters reached (%d/%d)", s.maxAdapters, s.maxAdapters), http.StatusConflict)
		return
	}

	// Generate device ID
	deviceID := generateAdapterID(req.AdapterName, req.AdapterOS)

	// Determine initial status
	initialStatus := PairingStateActive
	if s.approvalRequired {
		initialStatus = PairingStatePending
	}

	// Create mobile adapter record
	now := time.Now()
	device := &MobileAdapter{
		AdapterID:         deviceID,
		ProfileID:        s.profileID,
		AdapterName:       req.AdapterName,
		AdapterOS:         req.AdapterOS,
		AdapterVersion:    req.AdapterVersion,
		AdapterModel:      req.AdapterModel,
		RegisteredAt:     now,
		LastSeenAt:       now,
		ExpiresAt:        now.Add(DefaultTokenExpiry),
		Status:           initialStatus,
		Capabilities:     session.Capabilities,
		ApprovalRequired: s.approvalRequired,
	}

	// Generate token
	tokenString, err := s.tokenMgr.CreateToken(device, DefaultTokenExpiry)
	if err != nil {
		writeJSONError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	device.TokenHash = HashToken(tokenString)

	// Register device
	if err := s.registry.RegisterAdapter(device); err != nil {
		writeJSONError(w, "failed to register adapter", http.StatusInternalServerError)
		return
	}

	// Build response
	scheme := "ws"
	if s.tlsCert != nil {
		scheme = "wss"
	}
	wsEndpoint := fmt.Sprintf("%s://%s/aps/adapter/%s/ws", scheme, r.Host, s.profileID)

	resp := PairingResponse{
		AdapterID:   deviceID,
		Token:      tokenString,
		WSEndpoint: wsEndpoint,
		ExpiresAt:  device.ExpiresAt.Format(time.RFC3339),
		ProfileID:  s.profileID,
		Status:     string(initialStatus),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *AdapterServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := s.tokenMgr.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Verify device is active
	device, err := s.registry.GetAdapter(claims.AdapterID)
	if err != nil {
		http.Error(w, "adapter not found", http.StatusNotFound)
		return
	}
	if !device.IsActive() {
		http.Error(w, fmt.Sprintf("adapter not active (status: %s)", device.Status), http.StatusForbidden)
		return
	}

	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	dc := &AdapterConnection{
		conn:    conn,
		device:  device,
		closeCh: make(chan struct{}),
	}

	s.mu.Lock()
	s.conns[claims.AdapterID] = dc
	s.mu.Unlock()

	defer func() {
		dc.Close()
		s.mu.Lock()
		delete(s.conns, claims.AdapterID)
		s.mu.Unlock()
	}()

	// Update last seen
	s.registry.UpdateLastSeen(claims.AdapterID)

	// Send connection ACK
	dc.WriteJSON(&WSMessage{
		Type: "status",
		Payload: map[string]string{
			"status":    "connected",
			"device_id": claims.AdapterID,
			"profile":   s.profileID,
		},
	})

	// Message loop
	conn.SetReadLimit(MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	// Start heartbeat
	go s.heartbeat(dc)

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logging.GetLogger().Error("websocket error", err, "adapter", claims.AdapterID)
			}
			return
		}

		s.registry.UpdateLastSeen(claims.AdapterID)
		s.handleWSMessage(dc, &msg)
	}
}

func (s *AdapterServer) heartbeat(dc *AdapterConnection) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dc.mu.Lock()
			if dc.closed {
				dc.mu.Unlock()
				return
			}
			dc.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := dc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				dc.mu.Unlock()
				return
			}
			dc.mu.Unlock()
		case <-dc.closeCh:
			return
		}
	}
}

func (s *AdapterServer) handleWSMessage(dc *AdapterConnection, msg *WSMessage) {
	switch msg.Type {
	case "command":
		s.handleCommand(dc, msg)
	default:
		dc.WriteJSON(&WSMessage{
			ID:   msg.ID,
			Type: "error",
			Payload: map[string]string{
				"error": fmt.Sprintf("unknown message type: %s", msg.Type),
			},
		})
	}
}

func (s *AdapterServer) handleCommand(dc *AdapterConnection, msg *WSMessage) {
	// Parse command payload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		dc.WriteJSON(&WSMessage{
			ID:   msg.ID,
			Type: "error",
			Payload: map[string]string{"error": "invalid payload"},
		})
		return
	}

	var cmdPayload WSCommandPayload
	if err := json.Unmarshal(payloadBytes, &cmdPayload); err != nil {
		dc.WriteJSON(&WSMessage{
			ID:   msg.ID,
			Type: "error",
			Payload: map[string]string{"error": "invalid command payload"},
		})
		return
	}

	// Send running status
	dc.WriteJSON(&WSMessage{
		ID:   msg.ID,
		Type: "status",
		Payload: WSStatusPayload{
			Status:    "running",
			StartedAt: time.Now().Format(time.RFC3339),
		},
	})

	// TODO: Execute command via APSCore.ExecuteRun() when integrated with serve command
	// For now, acknowledge receipt
	dc.WriteJSON(&WSMessage{
		ID:   msg.ID,
		Type: "status",
		Payload: WSStatusPayload{
			Status: "received",
		},
	})
}

// --- AdapterConnection methods ---

func (dc *AdapterConnection) WriteJSON(v any) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.closed {
		return fmt.Errorf("connection closed")
	}
	dc.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	return dc.conn.WriteJSON(v)
}

func (dc *AdapterConnection) Close() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.closed {
		return
	}
	dc.closed = true
	close(dc.closeCh)
	dc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	dc.conn.Close()
}

// --- Helpers ---

func generateAdapterID(name, os string) string {
	// Create a human-friendly ID from device name and OS
	clean := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	clean = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, clean)
	if len(clean) > 20 {
		clean = clean[:20]
	}
	// Add short hash for uniqueness
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", name, os, time.Now().UnixNano())))
	suffix := hex.EncodeToString(h[:4])
	return fmt.Sprintf("%s-%s", clean, suffix)
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// DetectLANAddress returns the most likely LAN IP address for the machine
func DetectLANAddress() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			candidates = append(candidates, ip.String())
		}
	}

	if len(candidates) == 0 {
		return "127.0.0.1", nil
	}
	return candidates[0], nil
}

// ListNetworkInterfaces returns all available network interfaces with their IPs
type NetworkInterface struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Type string `json:"type"` // "wifi", "ethernet", "vpn", "other"
}

func ListNetworkInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []NetworkInterface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			ifaceType := classifyInterface(iface.Name)
			result = append(result, NetworkInterface{
				Name: iface.Name,
				IP:   ip.String(),
				Type: ifaceType,
			})
		}
	}

	return result, nil
}

func classifyInterface(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "en0"), strings.HasPrefix(n, "wl"):
		return "wifi"
	case strings.HasPrefix(n, "en"), strings.HasPrefix(n, "eth"):
		return "ethernet"
	case strings.HasPrefix(n, "utun"), strings.HasPrefix(n, "tun"), strings.HasPrefix(n, "wg"):
		return "vpn"
	default:
		return "other"
	}
}

// TLSCertFingerprint returns the SHA256 fingerprint of a TLS certificate
func TLSCertFingerprint(cert *tls.Certificate) (string, error) {
	if len(cert.Certificate) == 0 {
		return "", fmt.Errorf("no certificate data")
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(parsed.Raw)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
