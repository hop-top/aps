package a2a

import (
	"fmt"

	"hop.top/aps/internal/a2a/transport"
	"hop.top/aps/internal/core"
)

// MapIsolationToTransport maps APS isolation tier to A2A transport type
func MapIsolationToTransport(tier core.IsolationLevel) (transport.TransportType, error) {
	return transport.SelectTransport(tier)
}

// CreateTransportForProfile creates appropriate transport for a profile
func CreateTransportForProfile(profile *core.Profile, handler transport.MessageHandler) (transport.Transport, error) {
	if !core.ProfileHasCapability(profile, "a2a") {
		return nil, ErrA2ANotEnabled
	}

	if profile.Isolation.Level == "" {
		profile.Isolation.Level = core.IsolationProcess
	}

	tier := profile.Isolation.Level

	transportType, err := transport.SelectTransport(tier)
	if err != nil {
		return nil, fmt.Errorf("failed to select transport: %w", err)
	}

	switch transportType {
	case transport.TransportIPC:
		config := transport.DefaultIPCConfig(profile.ID)
		return transport.NewIPCTransport(config, handler)
	case transport.TransportHTTP:
		endpoint := getHTTPEndpoint(profile)
		config := transport.DefaultHTTPConfig(endpoint)
		return transport.NewHTTPTransport(config, handler)
	case transport.TransportGRPC:
		endpoint := getGRPCEndpoint(profile)
		config := transport.DefaultGRPCConfig(endpoint)
		return transport.NewGRPCTransport(config, handler)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

// getHTTPEndpoint returns HTTP endpoint from profile config
func getHTTPEndpoint(profile *core.Profile) string {
	if profile.A2A.ListenAddr != "" {
		return fmt.Sprintf("http://%s", profile.A2A.ListenAddr)
	}
	return fmt.Sprintf("http://127.0.0.1:8081")
}

// getGRPCEndpoint returns gRPC endpoint from profile config
func getGRPCEndpoint(profile *core.Profile) string {
	if profile.A2A.ListenAddr != "" {
		return profile.A2A.ListenAddr
	}
	return "127.0.0.1:8081"
}
