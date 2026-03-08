//go:build linux

package isolation

import (
	"hop.top/aps/internal/core"
)

func RegisterLinuxSandbox(manager *Manager) error {
	linux := NewLinuxSandbox()
	manager.Register(core.IsolationPlatform, linux)
	return nil
}
