package transport

import (
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/core"
)

// IsolationTierMapping maps APS isolation tiers to preferred transports
var IsolationTierMapping = map[core.IsolationLevel]TransportType{
	core.IsolationProcess:   TransportIPC,
	core.IsolationPlatform:  TransportHTTP,
	core.IsolationContainer: TransportGRPC,
}

// TransportPriority defines fallback order for transports
var TransportPriority = []TransportType{
	TransportIPC,
	TransportHTTP,
	TransportGRPC,
	TransportSLIM,
}

// SelectTransport chooses the best transport based on isolation tier
func SelectTransport(tier core.IsolationLevel) (TransportType, error) {
	transport, ok := IsolationTierMapping[tier]
	if !ok {
		return "", fmt.Errorf("unsupported isolation tier: %s", tier)
	}

	return transport, nil
}

// SelectTransportFromCard selects transport from Agent Card
func SelectTransportFromCard(card *a2a.AgentCard) (TransportType, error) {
	if card == nil {
		return "", fmt.Errorf("agent card cannot be nil")
	}

	for _, priority := range TransportPriority {
		for _, iface := range card.AdditionalInterfaces {
			if iface.Transport == a2a.TransportProtocol(priority) {
				return priority, nil
			}
		}

		if card.PreferredTransport == a2a.TransportProtocol(priority) {
			return priority, nil
		}
	}

	return "", fmt.Errorf("no compatible transport found in agent card")
}

// GetFallbackTransport returns the next transport in priority order
func GetFallbackTransport(current TransportType) (TransportType, bool) {
	for i, transport := range TransportPriority {
		if transport == current && i < len(TransportPriority)-1 {
			return TransportPriority[i+1], true
		}
	}

	return "", false
}
