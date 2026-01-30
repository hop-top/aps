package a2a

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/core"
)

func TestNewServer_A2ADisabled(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-disabled",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: false,
		},
	}

	server, err := NewServer(profile, config)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Equal(t, ErrA2ANotEnabled, err)
}

func TestNewServer_NilProfile(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-nil",
	}

	server, err := NewServer(nil, config)

	assert.Error(t, err)
	assert.Nil(t, server)
}

func TestNewServer_NilConfig(t *testing.T) {
	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled: true,
		},
	}

	server, err := NewServer(profile, nil)

	assert.Error(t, err)
	assert.Nil(t, server)
}

func TestServer_Start(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-start",
	}

	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		A2A: &core.A2AConfig{
			Enabled:         true,
			ListenAddr:      "127.0.0.1:8081",
			ProtocolBinding: "jsonrpc",
			SecurityScheme:  "apikey",
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	assert.True(t, server.IsRunning())

	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServer_Stop(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-stop",
	}

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{
			Enabled:         true,
			ListenAddr:      "127.0.0.1:8081",
			ProtocolBinding: "jsonrpc",
		},
	}

	server, err := NewServer(profile, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	assert.True(t, server.IsRunning())

	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServer_GetAddress(t *testing.T) {
	config := &StorageConfig{
		BasePath: "/tmp/test-a2a-server-addr",
	}

	t.Run("default address", func(t *testing.T) {
		profile := &core.Profile{
			ID: "test-profile",
			A2A: &core.A2AConfig{
				Enabled: true,
			},
		}

		server, err := NewServer(profile, config)
		require.NoError(t, err)
		assert.Contains(t, server.getAddress(), ":8081")
	})

	t.Run("custom address", func(t *testing.T) {
		profile := &core.Profile{
			ID: "test-profile",
			A2A: &core.A2AConfig{
				Enabled:    true,
				ListenAddr: "127.0.0.1:9999",
			},
		}

		server, err := NewServer(profile, config)
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1:9999", server.getAddress())
	})
}
