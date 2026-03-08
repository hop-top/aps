package isolation_test

import (
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/isolation"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := isolation.NewManager()
	assert.NotNil(t, manager)
}

func TestRegisterAndGet(t *testing.T) {
	manager := isolation.NewManager()
	adapter := isolation.NewProcessIsolation()

	manager.Register(core.IsolationProcess, adapter)

	retrieved, err := manager.Get(core.IsolationProcess)
	assert.NoError(t, err)
	assert.Equal(t, adapter, retrieved)
}

func TestGetNotRegistered(t *testing.T) {
	manager := isolation.NewManager()

	_, err := manager.Get(core.IsolationContainer)
	assert.Error(t, err)
	assert.ErrorIs(t, err, isolation.ErrIsolationNotSupported)
}

func TestIsolationLevels(t *testing.T) {
	assert.Equal(t, "process", string(core.IsolationProcess))
	assert.Equal(t, "platform", string(core.IsolationPlatform))
	assert.Equal(t, "container", string(core.IsolationContainer))
}
