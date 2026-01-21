package core_test

import (
	"testing"

	"oss-aps-cli/internal/core/session"

	"github.com/stretchr/testify/assert"
)

func TestGetRegistry(t *testing.T) {
	registry := session.GetRegistry()
	assert.NotNil(t, registry)
}

func TestSSHKeyManager(t *testing.T) {
	manager := session.NewSSHKeyManager()
	assert.NotNil(t, manager)
}
