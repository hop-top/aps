package adapters

import (
	"oss-aps-cli/internal/adapters/agentprotocol"
)

func init() {
	// Register all protocol adapters
	GetProtocolRegistry().RegisterHTTPAdapter("agent-protocol", agentprotocol.NewAgentProtocolAdapter())
}
