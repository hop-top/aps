package core_test

import (
	"testing"

	"oss-aps-cli/internal/core/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHKeyGeneration(t *testing.T) {
	manager := session.NewSSHKeyManager()

	t.Run("Generate Ed25519 key pair", func(t *testing.T) {
		key, err := manager.GenerateKeyPair(session.SSHKeyEd25519)
		require.NoError(t, err)
		assert.Equal(t, session.SSHKeyEd25519, key.KeyType)
		assert.NotEmpty(t, key.PublicKey)
		assert.NotEmpty(t, key.PrivateKey)
	})

	t.Run("Generate RSA key pair", func(t *testing.T) {
		key, err := manager.GenerateKeyPair(session.SSHKeyRSA)
		require.NoError(t, err)
		assert.Equal(t, session.SSHKeyRSA, key.KeyType)
		assert.NotEmpty(t, key.PublicKey)
		assert.NotEmpty(t, key.PrivateKey)
	})

	t.Run("Reject invalid key type", func(t *testing.T) {
		_, err := manager.GenerateKeyPair(session.SSHKeyType("invalid"))
		assert.Error(t, err)
	})
}
