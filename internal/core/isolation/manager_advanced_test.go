package isolation

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oss-aps-cli/internal/core"
)

// ============================================================================
// Manager Tests - Get() Method (5 tests)
// ============================================================================

// TestManagerGetContainerLevel retrieves correct adapter for container isolation level
func TestManagerGetContainerLevel(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, containerAdapter)
	adapter, err := manager.Get(core.IsolationContainer)

	require.NoError(t, err)
	assert.Equal(t, containerAdapter, adapter)
}

// TestManagerGetPlatformLevel retrieves correct adapter for platform isolation level
func TestManagerGetPlatformLevel(t *testing.T) {
	manager := NewManager()
	platformAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, platformAdapter)
	adapter, err := manager.Get(core.IsolationPlatform)

	require.NoError(t, err)
	assert.Equal(t, platformAdapter, adapter)
}

// TestManagerGetProcessLevel retrieves correct adapter for process isolation level
func TestManagerGetProcessLevel(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)
	adapter, err := manager.Get(core.IsolationProcess)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestManagerGetInvalidLevel handles invalid isolation levels
func TestManagerGetInvalidLevel(t *testing.T) {
	manager := NewManager()
	invalidLevel := core.IsolationLevel("invalid-level")

	adapter, err := manager.Get(invalidLevel)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Equal(t, ErrIsolationNotSupported, err)
}

// TestManagerGetUnregisteredLevel returns error for unregistered level
func TestManagerGetUnregisteredLevel(t *testing.T) {
	manager := NewManager()
	manager.Register(core.IsolationProcess, &MockIsolationManager{available: true})

	adapter, err := manager.Get(core.IsolationContainer)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Equal(t, ErrIsolationNotSupported, err)
}

// ============================================================================
// GetIsolationManager Tests - Basic Functionality (8 tests)
// ============================================================================

// TestGetIsolationManagerRequestedLevelAvailable returns requested level when available
func TestGetIsolationManagerRequestedLevelAvailable(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: true}
	manager.Register(core.IsolationContainer, containerAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, containerAdapter, adapter)
}

// TestGetIsolationManagerUsesProfileLevelWhenSet uses profile level over global default
func TestGetIsolationManagerUsesProfileLevelWhenSet(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: true}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: false,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, containerAdapter, adapter)
}

// TestGetIsolationManagerUsesGlobalDefaultWhenProfileEmpty uses global default when profile level is empty
func TestGetIsolationManagerUsesGlobalDefaultWhenProfileEmpty(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    "",
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestGetIsolationManagerStrictModePreventsFallback strict mode prevents fallback
func TestGetIsolationManagerStrictModePreventsFallback(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   true,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "strict mode violation")
}

// TestGetIsolationManagerFallbackDisabledOnProfile returns error when profile fallback disabled
func TestGetIsolationManagerFallbackDisabledOnProfile(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: false,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "fallback disabled")
}

// TestGetIsolationManagerFallbackDisabledGlobally returns error when global fallback disabled
func TestGetIsolationManagerFallbackDisabledGlobally(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: false,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "global fallback disabled")
}

// TestGetIsolationManagerNoAvailableAdapters returns error when no adapters available
func TestGetIsolationManagerNoAvailableAdapters(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: false}
	platformAdapter := &MockIsolationManager{available: false}
	processAdapter := &MockIsolationManager{available: false}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "no available isolation adapter")
}

// ============================================================================
// Fallback Chain Tests (12 tests)
// ============================================================================

// TestFallbackContainerToPlatform falls back from container to platform
func TestFallbackContainerToPlatform(t *testing.T) {
	manager := NewManager()
	platformAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, platformAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, platformAdapter, adapter)
}

// TestFallbackContainerToProcess falls back from container to process
func TestFallbackContainerToProcessAdvanced(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestFallbackPlatformToProcess falls back from platform to process
func TestFallbackPlatformToProcessAdvanced(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationPlatform,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestFallbackChainContainerThroughPlatformToProcess uses full fallback chain
func TestFallbackChainContainerThroughPlatformToProcess(t *testing.T) {
	manager := NewManager()
	platformAdapter := &MockIsolationManager{available: false}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestFallbackPrefersPlatformWhenAvailable prefers platform over process when available
func TestFallbackPrefersPlatformWhenAvailable(t *testing.T) {
	manager := NewManager()
	platformAdapter := &MockIsolationManager{available: true}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, platformAdapter, adapter)
}

// TestFallbackSelectsFirstAvailableAdapter selects first available in chain
func TestFallbackSelectsFirstAvailableAdapter(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: false}
	platformAdapter := &MockIsolationManager{available: false}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestFallbackOrderConsistency verifies fallback chain order is consistent
func TestFallbackOrderConsistency(t *testing.T) {
	// Verify fallback levels order for Container
	levels := getFallbackLevels(core.IsolationContainer)
	require.Equal(t, 2, len(levels))
	assert.Equal(t, core.IsolationPlatform, levels[0])
	assert.Equal(t, core.IsolationProcess, levels[1])

	// Verify fallback levels order for Platform
	levels = getFallbackLevels(core.IsolationPlatform)
	require.Equal(t, 1, len(levels))
	assert.Equal(t, core.IsolationProcess, levels[0])

	// Verify fallback levels order for Process
	levels = getFallbackLevels(core.IsolationProcess)
	require.Equal(t, 0, len(levels))
}

// TestFallbackWithUnavailableRequestedLevel falls back when requested level unavailable
func TestFallbackWithUnavailableRequestedLevel(t *testing.T) {
	manager := NewManager()
	unavailableContainer := &MockIsolationManager{available: false}
	availablePlatform := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, unavailableContainer)
	manager.Register(core.IsolationPlatform, availablePlatform)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, availablePlatform, adapter)
}

// TestFallbackWithPartialRegistration works with partially registered adapters
func TestFallbackWithPartialRegistration(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestFallbackWithAllUnavailableLevels returns error when all levels unavailable
func TestFallbackWithAllUnavailableLevels(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: false}
	platformAdapter := &MockIsolationManager{available: false}
	processAdapter := &MockIsolationManager{available: false}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
}

// ============================================================================
// Availability Checks Tests (6 tests)
// ============================================================================

// TestIsAvailableContainerLevel checks availability for container level
func TestIsAvailableContainerLevel(t *testing.T) {
	manager := NewManager()
	availableAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, availableAdapter)
	adapter, err := manager.Get(core.IsolationContainer)

	require.NoError(t, err)
	assert.True(t, adapter.IsAvailable())
}

// TestIsAvailablePlatformLevel checks availability for platform level
func TestIsAvailablePlatformLevel(t *testing.T) {
	manager := NewManager()
	availableAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, availableAdapter)
	adapter, err := manager.Get(core.IsolationPlatform)

	require.NoError(t, err)
	assert.True(t, adapter.IsAvailable())
}

// TestIsAvailableProcessLevel checks availability for process level
func TestIsAvailableProcessLevel(t *testing.T) {
	manager := NewManager()
	processAdapter := NewProcessIsolation()

	manager.Register(core.IsolationProcess, processAdapter)
	adapter, err := manager.Get(core.IsolationProcess)

	require.NoError(t, err)
	assert.True(t, adapter.IsAvailable())
}

// TestIsAvailableUnavailableAdapter returns false for unavailable adapters
func TestIsAvailableUnavailableAdapter(t *testing.T) {
	manager := NewManager()
	unavailableAdapter := &MockIsolationManager{available: false}

	manager.Register(core.IsolationContainer, unavailableAdapter)
	adapter, err := manager.Get(core.IsolationContainer)

	require.NoError(t, err)
	assert.False(t, adapter.IsAvailable())
}

// TestIsAvailableMultipleAdapters checks multiple adapters availability
func TestIsAvailableMultipleAdapters(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: true}
	platformAdapter := &MockIsolationManager{available: false}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	container, _ := manager.Get(core.IsolationContainer)
	assert.True(t, container.IsAvailable())

	platform, _ := manager.Get(core.IsolationPlatform)
	assert.False(t, platform.IsAvailable())

	process, _ := manager.Get(core.IsolationProcess)
	assert.True(t, process.IsAvailable())
}

// TestIsAvailableAfterRegistration checks availability after registration
func TestIsAvailableAfterRegistration(t *testing.T) {
	manager := NewManager()
	adapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, adapter)
	retrieved, err := manager.Get(core.IsolationProcess)

	require.NoError(t, err)
	assert.True(t, retrieved.IsAvailable())
}

// ============================================================================
// Concurrent Access Tests (5 tests)
// ============================================================================

// TestConcurrentAccessSameLevel multiple goroutines access same level
func TestConcurrentAccessSameLevel(t *testing.T) {
	manager := NewManager()
	adapter := &MockIsolationManager{available: true}
	manager.Register(core.IsolationProcess, adapter)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.Get(core.IsolationProcess)
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentAccessDifferentLevels multiple goroutines access different levels
func TestConcurrentAccessDifferentLevels(t *testing.T) {
	manager := NewManager()
	containerAdapter := &MockIsolationManager{available: true}
	platformAdapter := &MockIsolationManager{available: true}
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationContainer, containerAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationProcess, processAdapter)

	var wg sync.WaitGroup
	errors := make(chan error, 9)

	// 3 goroutines per level
	for i := 0; i < 3; i++ {
		wg.Add(3)
		go func(idx int) {
			defer wg.Done()
			_, err := manager.Get(core.IsolationContainer)
			errors <- err
		}(i)
		go func(idx int) {
			defer wg.Done()
			_, err := manager.Get(core.IsolationPlatform)
			errors <- err
		}(i)
		go func(idx int) {
			defer wg.Done()
			_, err := manager.Get(core.IsolationProcess)
			errors <- err
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentGetIsolationManager multiple concurrent GetIsolationManager calls
func TestConcurrentGetIsolationManager(t *testing.T) {
	manager := NewManager()
	adapter := &MockIsolationManager{available: true}
	manager.Register(core.IsolationProcess, adapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationProcess,
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetIsolationManager(profile, globalConfig)
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestSequentialRegistrationAndRetrieval sequential registration and retrieval
func TestSequentialRegistrationAndRetrieval(t *testing.T) {
	manager := NewManager()

	// Register adapters sequentially
	adapter1 := &MockIsolationManager{available: true}
	adapter2 := &MockIsolationManager{available: true}
	adapter3 := &MockIsolationManager{available: true}

	level1 := core.IsolationProcess
	level2 := core.IsolationPlatform
	level3 := core.IsolationContainer

	manager.Register(level1, adapter1)
	manager.Register(level2, adapter2)
	manager.Register(level3, adapter3)

	// Retrieve adapters
	retrieved1, err1 := manager.Get(level1)
	retrieved2, err2 := manager.Get(level2)
	retrieved3, err3 := manager.Get(level3)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Equal(t, adapter1, retrieved1)
	assert.Equal(t, adapter2, retrieved2)
	assert.Equal(t, adapter3, retrieved3)
}

// TestAdapterReplacementSequential sequential adapter replacement
func TestAdapterReplacementSequential(t *testing.T) {
	manager := NewManager()
	adapter1 := &MockIsolationManager{available: true}
	manager.Register(core.IsolationProcess, adapter1)

	// Replace adapter sequentially
	retrieved1, _ := manager.Get(core.IsolationProcess)
	assert.Equal(t, adapter1, retrieved1)
	assert.True(t, retrieved1.IsAvailable())

	// Replace with new adapter
	adapter2 := &MockIsolationManager{available: false}
	manager.Register(core.IsolationProcess, adapter2)

	retrieved2, _ := manager.Get(core.IsolationProcess)
	assert.Equal(t, adapter2, retrieved2)
	assert.False(t, retrieved2.IsAvailable())

	// Replace again
	adapter3 := &MockIsolationManager{available: true}
	manager.Register(core.IsolationProcess, adapter3)

	retrieved3, _ := manager.Get(core.IsolationProcess)
	assert.Equal(t, adapter3, retrieved3)
	assert.True(t, retrieved3.IsAvailable())
}

// ============================================================================
// Manager Integration Tests (6 tests)
// ============================================================================

// TestManagerCreateRegisterRetrieve create manager, register adapters, retrieve in sequence
func TestManagerCreateRegisterRetrieve(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)

	processAdapter := &MockIsolationManager{available: true}
	platformAdapter := &MockIsolationManager{available: true}
	containerAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)
	manager.Register(core.IsolationPlatform, platformAdapter)
	manager.Register(core.IsolationContainer, containerAdapter)

	proc, err1 := manager.Get(core.IsolationProcess)
	plat, err2 := manager.Get(core.IsolationPlatform)
	cont, err3 := manager.Get(core.IsolationContainer)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Equal(t, processAdapter, proc)
	assert.Equal(t, platformAdapter, plat)
	assert.Equal(t, containerAdapter, cont)
}

// TestManagerMultipleInstances independent managers don't interfere
func TestManagerMultipleInstances(t *testing.T) {
	manager1 := NewManager()
	manager2 := NewManager()

	adapter1 := &MockIsolationManager{available: true}
	adapter2 := &MockIsolationManager{available: false}

	manager1.Register(core.IsolationProcess, adapter1)
	manager2.Register(core.IsolationProcess, adapter2)

	retrieved1, _ := manager1.Get(core.IsolationProcess)
	retrieved2, _ := manager2.Get(core.IsolationProcess)

	assert.Equal(t, adapter1, retrieved1)
	assert.Equal(t, adapter2, retrieved2)
	assert.True(t, retrieved1.IsAvailable())
	assert.False(t, retrieved2.IsAvailable())
}

// TestManagerStateAfterError manager state valid after error
func TestManagerStateAfterError(t *testing.T) {
	manager := NewManager()

	// Attempt to get non-existent adapter
	_, err := manager.Get(core.IsolationLevel("nonexistent"))
	assert.Error(t, err)

	// Register new adapter
	adapter := &MockIsolationManager{available: true}
	manager.Register(core.IsolationProcess, adapter)

	// Should work now
	retrieved, err := manager.Get(core.IsolationProcess)
	require.NoError(t, err)
	assert.Equal(t, adapter, retrieved)
}

// TestManagerCleanupRegistration register, use, then verify cleanup
func TestManagerCleanupRegistration(t *testing.T) {
	manager := NewManager()
	adapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, adapter)
	retrieved, _ := manager.Get(core.IsolationProcess)

	// Simulate cleanup by getting adapter and calling its cleanup
	err := retrieved.Cleanup()
	require.NoError(t, err)

	// Adapter should still be retrievable
	retrieved2, err := manager.Get(core.IsolationProcess)
	require.NoError(t, err)
	assert.NotNil(t, retrieved2)
}

// TestManagerIsolationLevelConstants verifies isolation level constants
func TestManagerIsolationLevelConstants(t *testing.T) {
	levels := []core.IsolationLevel{
		core.IsolationProcess,
		core.IsolationPlatform,
		core.IsolationContainer,
	}

	assert.Equal(t, core.IsolationLevel("process"), levels[0])
	assert.Equal(t, core.IsolationLevel("platform"), levels[1])
	assert.Equal(t, core.IsolationLevel("container"), levels[2])

	for _, level := range levels {
		assert.NotEmpty(t, level)
	}
}

// TestManagerEmptyInitialization manager starts with empty adapters
func TestManagerEmptyInitialization(t *testing.T) {
	manager := NewManager()

	assert.NotNil(t, manager.adapters)
	assert.Equal(t, 0, len(manager.adapters))
}

// ============================================================================
// Error Handling Tests (8 tests)
// ============================================================================

// TestGetWithoutRegistration get adapter before registration
func TestGetWithoutRegistration(t *testing.T) {
	manager := NewManager()

	adapter, err := manager.Get(core.IsolationContainer)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Equal(t, ErrIsolationNotSupported, err)
}

// TestGetIsolationManagerWithoutFallback error when no fallback and unavailable
func TestGetIsolationManagerWithoutFallback(t *testing.T) {
	manager := NewManager()

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: false,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: false,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
}

// TestStrictModeWithUnavailableLevel strict mode error has correct message
func TestStrictModeWithUnavailableLevel(t *testing.T) {
	manager := NewManager()
	platformAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationPlatform, platformAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: true,
			Strict:   true,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "container")
}

// TestProfileOverridesGlobalDefault profile isolation overrides global config
func TestProfileOverridesGlobalDefault(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}
	containerAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)
	manager.Register(core.IsolationContainer, containerAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Fallback: false,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, containerAdapter, adapter)
}

// TestMissingProfileLevelUsesGlobal empty profile level uses global default
func TestMissingProfileLevelUsesGlobal(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    "",
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.Equal(t, processAdapter, adapter)
}

// TestNilProfileConfig handles nil profile config gracefully
func TestNilProfileConfigHandling(t *testing.T) {
	manager := NewManager()
	processAdapter := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, processAdapter)

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Level:    "",
			Fallback: true,
			Strict:   false,
		},
	}

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	adapter, err := manager.GetIsolationManager(profile, globalConfig)

	require.NoError(t, err)
	assert.NotNil(t, adapter)
}

// TestAdapterReplacementWithGet adapter replacement is reflected in Get
func TestAdapterReplacementWithGet(t *testing.T) {
	manager := NewManager()
	adapter1 := &MockIsolationManager{available: true}

	manager.Register(core.IsolationProcess, adapter1)
	retrieved1, _ := manager.Get(core.IsolationProcess)
	assert.Equal(t, adapter1, retrieved1)

	// Replace adapter
	adapter2 := &MockIsolationManager{available: false}
	manager.Register(core.IsolationProcess, adapter2)
	retrieved2, _ := manager.Get(core.IsolationProcess)

	assert.Equal(t, adapter2, retrieved2)
	assert.NotEqual(t, retrieved1, retrieved2)
}
