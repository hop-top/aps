package isolation_test

import (
	"errors"
	"testing"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/isolation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAdapter is a test implementation of IsolationManager
type mockAdapter struct {
	validateError error
}

func (m *mockAdapter) PrepareContext(profileID string) (*isolation.ExecutionContext, error) {
	return &isolation.ExecutionContext{ProfileID: profileID}, nil
}

func (m *mockAdapter) SetupEnvironment(cmd interface{}) error {
	return nil
}

func (m *mockAdapter) Execute(command string, args []string) error {
	return nil
}

func (m *mockAdapter) ExecuteAction(actionID string, payload []byte) error {
	return nil
}

func (m *mockAdapter) Cleanup() error {
	return nil
}

func (m *mockAdapter) Validate() error {
	if m.validateError != nil {
		return m.validateError
	}
	return nil
}

func (m *mockAdapter) IsAvailable() bool {
	return m.validateError == nil
}

func TestGetIsolationManager_ExactMatch(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationProcess,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	require.NoError(t, err)
	assert.Equal(t, adapter, result)
}

func TestGetIsolationManager_StrictModeFailure(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   true,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "strict mode violation")
}

func TestGetIsolationManager_FallbackDisabled(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   false,
			Fallback: false,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "fallback disabled")
}

func TestGetIsolationManager_GlobalFallbackDisabled(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: false,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "global fallback disabled")
}

func TestGetIsolationManager_GracefulDegradation(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	require.NoError(t, err)
	assert.Equal(t, adapter, result)
}

func TestGetIsolationManager_MultipleFallbackLevels(t *testing.T) {
	manager := isolation.NewManager()
	processAdapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, processAdapter)
	manager.Register(core.IsolationPlatform, &mockAdapter{
		validateError: errors.New("platform not available"),
	})

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	require.NoError(t, err)
	assert.Equal(t, processAdapter, result)
}

func TestGetIsolationManager_InvalidAdapter(t *testing.T) {
	manager := isolation.NewManager()
	adapter := &mockAdapter{
		validateError: errors.New("adapter failed validation"),
	}
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationProcess,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no available isolation adapter")
}

func TestGetIsolationManager_UseDefaultLevel(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    "",
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	require.NoError(t, err)
	assert.Equal(t, adapter, result)
}

func TestGetIsolationManager_NoAdaptersAvailable(t *testing.T) {
	manager := isolation.NewManager()

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   false,
			Fallback: true,
		},
	}
	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	result, err := manager.GetIsolationManager(profile, globalConfig)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no available isolation adapter")
}
