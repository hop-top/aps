package isolation

import (
	"errors"
	"fmt"

	"oss-aps-cli/internal/core"
)

var (
	ErrIsolationNotSupported = errors.New("isolation level not supported")
	ErrInvalidProfile        = errors.New("invalid profile")
	ErrExecutionFailed       = errors.New("execution failed")
	ErrStrictModeViolation   = errors.New("strict mode violation: requested isolation level not available")
	ErrNoAvailableAdapter    = errors.New("no available isolation adapter")
)

type ExecutionContext struct {
	ProfileID   string
	ProfileDir  string
	ProfileYaml string
	SecretsPath string
	DocsDir     string
	Environment map[string]string
	WorkingDir  string
}

type IsolationManager interface {
	PrepareContext(profileID string) (*ExecutionContext, error)
	SetupEnvironment(cmd interface{}) error
	Execute(command string, args []string) error
	ExecuteAction(actionID string, payload []byte) error
	Cleanup() error
	Validate() error
	IsAvailable() bool
}

type Manager struct {
	adapters map[core.IsolationLevel]IsolationManager
}

func NewManager() *Manager {
	return &Manager{
		adapters: make(map[core.IsolationLevel]IsolationManager),
	}
}

func (m *Manager) Register(level core.IsolationLevel, adapter IsolationManager) {
	m.adapters[level] = adapter
}

func (m *Manager) Get(level core.IsolationLevel) (IsolationManager, error) {
	adapter, ok := m.adapters[level]
	if !ok {
		return nil, ErrIsolationNotSupported
	}
	return adapter, nil
}

// GetIsolationManager returns the appropriate isolation manager based on profile and global config
// It implements fallback logic and strict mode enforcement
func (m *Manager) GetIsolationManager(profile *core.Profile, globalConfig *core.Config) (IsolationManager, error) {
	requestedLevel := profile.Isolation.Level
	if requestedLevel == "" {
		requestedLevel = globalConfig.Isolation.DefaultLevel
	}

	// Try to get requested level
	adapter, err := m.Get(requestedLevel)
	if err == nil {
		// Check if adapter is available
		if adapter.IsAvailable() {
			return adapter, nil
		}
	}

	// If not available and strict mode is enabled, fail
	if profile.Isolation.Strict {
		return nil, fmt.Errorf("%w: requested level %s not available", ErrStrictModeViolation, requestedLevel)
	}

	// If fallback is disabled, return error
	if !profile.Isolation.Fallback {
		return nil, fmt.Errorf("%w: fallback disabled and %s not available", ErrIsolationNotSupported, requestedLevel)
	}

	// Check global fallback setting
	if !globalConfig.Isolation.FallbackEnabled {
		return nil, fmt.Errorf("%w: global fallback disabled and %s not available", ErrIsolationNotSupported, requestedLevel)
	}

	// Implement graceful degradation - try next best options
	fallbackLevels := getFallbackLevels(requestedLevel)

	for _, level := range fallbackLevels {
		adapter, err := m.Get(level)
		if err == nil {
			if adapter.IsAvailable() {
				return adapter, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: no available isolation adapter after fallback", ErrNoAvailableAdapter)
}

// getFallbackLevels returns ordered list of fallback isolation levels
// Ordered from most secure to least secure
func getFallbackLevels(level core.IsolationLevel) []core.IsolationLevel {
	switch level {
	case core.IsolationContainer:
		return []core.IsolationLevel{core.IsolationPlatform, core.IsolationProcess}
	case core.IsolationPlatform:
		return []core.IsolationLevel{core.IsolationProcess}
	case core.IsolationProcess:
		return []core.IsolationLevel{}
	default:
		return []core.IsolationLevel{core.IsolationProcess}
	}
}
