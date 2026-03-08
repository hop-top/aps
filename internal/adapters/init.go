package adapters

import (
	"hop.top/aps/internal/adapters/agentprotocol"
)

func init() {
	// Register all protocol adapters
	GetProtocolRegistry().RegisterHTTPAdapter("agent-protocol", agentprotocol.NewAgentProtocolAdapter())
}
