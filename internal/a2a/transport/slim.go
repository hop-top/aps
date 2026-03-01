package transport

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

// TransportSLIM is the SLIM transport type constant.
const TransportSLIM TransportType = "slim"

// SLIMConfig holds configuration for the SLIM transport.
type SLIMConfig struct {
	Endpoint string
	GroupID  string
}

// SLIMTransport implements the Transport interface for AGNTCY SLIM messaging.
// This is a stub — the Go binding (github.com/agntcy/slim/bindings/go) requires
// CGO and is not yet stable. Track for maturity before enabling.
type SLIMTransport struct {
	config  SLIMConfig
	handler MessageHandler
}

// NewSLIMTransport creates a new SLIM transport (stub).
func NewSLIMTransport(cfg SLIMConfig, handler MessageHandler) (*SLIMTransport, error) {
	return &SLIMTransport{
		config:  cfg,
		handler: handler,
	}, nil
}

func (t *SLIMTransport) Type() TransportType {
	return TransportSLIM
}

func (t *SLIMTransport) Send(ctx context.Context, message *a2a.Message) error {
	return fmt.Errorf("SLIM transport not yet implemented: Go binding pending release")
}

func (t *SLIMTransport) Receive(ctx context.Context) (*a2a.Message, error) {
	return nil, fmt.Errorf("SLIM transport not yet implemented: Go binding pending release")
}

func (t *SLIMTransport) Close() error {
	return nil
}

func (t *SLIMTransport) IsHealthy() bool {
	return false
}
