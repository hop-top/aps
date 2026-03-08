package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"hop.top/aps/internal/core"
)

// IPCConfig holds IPC transport configuration
type IPCConfig struct {
	ProfileID    string
	QueuePath    string
	Polling      bool
	PollInterval time.Duration
}

// DefaultIPCConfig returns default IPC configuration
func DefaultIPCConfig(profileID string) *IPCConfig {
	dataDir, err := core.GetDataDir()
	if err != nil {
		dataDir = ""
	}

	queuePath := filepath.Join(dataDir, "ipc", "queues", profileID, "incoming")

	return &IPCConfig{
		ProfileID:    profileID,
		QueuePath:    queuePath,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}
}

// IPCTransport implements A2A transport via filesystem IPC queues
type IPCTransport struct {
	config  *IPCConfig
	handler MessageHandler
	msgChan chan *a2a.Message
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      chan struct{}
}

var _ Transport = (*IPCTransport)(nil)

// NewIPCTransport creates a new IPC transport instance
func NewIPCTransport(config *IPCConfig, handler MessageHandler) (*IPCTransport, error) {
	if config == nil {
		return nil, fmt.Errorf("ipc config cannot be nil")
	}

	if config.ProfileID == "" {
		return nil, fmt.Errorf("profile id cannot be empty")
	}

	if err := os.MkdirAll(config.QueuePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create ipc queue directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IPCTransport{
		config:  config,
		handler: handler,
		msgChan: make(chan *a2a.Message, 100),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
		mu:      make(chan struct{}, 1),
	}, nil
}

// Type returns the transport type
func (t *IPCTransport) Type() TransportType {
	return TransportIPC
}

// Send sends a message via IPC transport
func (t *IPCTransport) Send(ctx context.Context, message *a2a.Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%d_%s.json", timestamp, uuid.New().String())
	messagePath := filepath.Join(t.config.QueuePath, filename)

	if err := os.WriteFile(messagePath, messageData, 0600); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Receive receives a message from IPC transport
func (t *IPCTransport) Receive(ctx context.Context) (*a2a.Message, error) {
	select {
	case msg := <-t.msgChan:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes IPC transport
func (t *IPCTransport) Close() error {
	t.cancel()
	return nil
}

// IsHealthy checks if IPC transport is operational
func (t *IPCTransport) IsHealthy() bool {
	_, err := os.Stat(t.config.QueuePath)
	return err == nil
}

// Start begins IPC message polling
func (t *IPCTransport) Start() error {
	select {
	case t.mu <- struct{}{}:
	default:
		return nil
	}

	t.running = true
	go t.pollMessages()

	return nil
}

// Stop halts IPC message polling
func (t *IPCTransport) Stop() error {
	t.running = false
	<-t.mu
	return nil
}

// pollMessages continuously checks queue directory for new messages
func (t *IPCTransport) pollMessages() {
	for t.running {
		select {
		case <-t.ctx.Done():
			return
		default:
			t.processQueuedMessages()
			time.Sleep(t.config.PollInterval)
		}
	}
}

// processQueuedMessages reads and processes all messages in queue
func (t *IPCTransport) processQueuedMessages() {
	entries, err := os.ReadDir(t.config.QueuePath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		messagePath := filepath.Join(t.config.QueuePath, entry.Name())
		data, err := os.ReadFile(messagePath)
		if err != nil {
			os.Remove(messagePath)
			continue
		}

		var message a2a.Message
		if err := json.Unmarshal(data, &message); err != nil {
			os.Remove(messagePath)
			continue
		}

		select {
		case t.msgChan <- &message:
			os.Remove(messagePath)
		case <-t.ctx.Done():
			return
		default:
			os.Remove(messagePath)
		}
	}
}

// GetQueuePath returns the IPC queue path
func (t *IPCTransport) GetQueuePath() string {
	return t.config.QueuePath
}

// GetProfileID returns the profile ID for this transport
func (t *IPCTransport) GetProfileID() string {
	return t.config.ProfileID
}
