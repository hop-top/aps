package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

// HTTPConfig holds HTTP transport configuration
type HTTPConfig struct {
	Endpoint     string
	ContentType  string
	Timeout      time.Duration
	APIKey       string
	SecurityType string
}

// DefaultHTTPConfig returns default HTTP configuration
func DefaultHTTPConfig(endpoint string) *HTTPConfig {
	return &HTTPConfig{
		Endpoint:     endpoint,
		ContentType:  "application/json",
		Timeout:      30 * time.Second,
		SecurityType: "none",
	}
}

// HTTPTransport implements A2A transport via HTTP
type HTTPTransport struct {
	config  *HTTPConfig
	handler MessageHandler
	client  *http.Client
	msgChan chan *a2a.Message
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

var _ Transport = (*HTTPTransport)(nil)

// NewHTTPTransport creates a new HTTP transport instance
func NewHTTPTransport(config *HTTPConfig, handler MessageHandler) (*HTTPTransport, error) {
	if config == nil {
		return nil, fmt.Errorf("http config cannot be nil")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HTTPTransport{
		config:  config,
		handler: handler,
		client:  &http.Client{Timeout: config.Timeout},
		msgChan: make(chan *a2a.Message, 100),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}, nil
}

// Type returns the transport type
func (t *HTTPTransport) Type() TransportType {
	return TransportHTTP
}

// Send sends a message via HTTP transport
func (t *HTTPTransport) Send(ctx context.Context, message *a2a.Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "SendMessage",
		"params":  message,
		"id":      generateRequestID(),
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", t.config.ContentType)

	if t.config.APIKey != "" {
		req.Header.Set("X-API-Key", t.config.APIKey)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Receive receives a message from HTTP transport (via SSE or webhook)
func (t *HTTPTransport) Receive(ctx context.Context) (*a2a.Message, error) {
	select {
	case msg := <-t.msgChan:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes HTTP transport
func (t *HTTPTransport) Close() error {
	t.cancel()
	t.running = false
	return nil
}

// IsHealthy checks if HTTP transport is operational
func (t *HTTPTransport) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", t.config.Endpoint+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// SubmitMessage delivers message to HTTP endpoint
func (t *HTTPTransport) SubmitMessage(ctx context.Context, message *a2a.Message) error {
	return t.Send(ctx, message)
}

// HandleServerResponse processes incoming messages from HTTP server
func (t *HTTPTransport) HandleServerResponse(message *a2a.Message) error {
	select {
	case t.msgChan <- message:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("transport is closed")
	}
}

var requestID int

func generateRequestID() int {
	requestID++
	return requestID
}
