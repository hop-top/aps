//go:build linux

package isolation

import (
	"oss-aps-cli/internal/core"
)

func RegisterLinuxSandbox(manager *Manager) error {
	linux := NewLinuxSandbox()
	manager.Register(core.IsolationPlatform, linux)
	return nil
}
