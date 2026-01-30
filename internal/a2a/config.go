package a2a

import (
	"fmt"
	"os"
	"path/filepath"
)

// StorageConfig holds A2A storage configuration
type StorageConfig struct {
	BasePath       string
	TasksPath      string
	AgentCardsPath string
	IPCPath        string
}

// DefaultStorageConfig returns a default storage configuration
func DefaultStorageConfig() *StorageConfig {
	homeDir, err := os.UserConfigDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	basePath := filepath.Join(homeDir, ".agents", "a2a")

	return &StorageConfig{
		BasePath:       basePath,
		TasksPath:      filepath.Join(basePath, "tasks"),
		AgentCardsPath: filepath.Join(basePath, "agent-cards"),
		IPCPath:        filepath.Join(basePath, "..", "ipc", "queues"),
	}
}

// ValidateProfileConfig validates A2A configuration
func ValidateProfileConfig(enabled bool, protocolBinding, securityScheme, isolationTier string) error {
	if !enabled {
		return nil
	}

	if protocolBinding == "" {
		return fmt.Errorf("protocol_binding is required when A2A is enabled")
	}

	validBindings := map[string]bool{
		"jsonrpc": true,
		"grpc":    true,
		"http":    true,
	}
	if !validBindings[protocolBinding] {
		return fmt.Errorf("invalid protocol_binding: %s", protocolBinding)
	}

	validSchemes := map[string]bool{
		"apikey": true,
		"mtls":   true,
		"openid": true,
	}
	if !validSchemes[securityScheme] {
		return fmt.Errorf("invalid security_scheme: %s", securityScheme)
	}

	validTiers := map[string]bool{
		"process":   true,
		"platform":  true,
		"container": true,
	}
	if !validTiers[isolationTier] {
		return fmt.Errorf("invalid isolation_tier: %s", isolationTier)
	}

	return nil
}

type A2AConfig struct {
	Enabled         bool
	ProtocolBinding string
	ListenAddr      string
	SecurityScheme  string
	IsolationTier   string
}
