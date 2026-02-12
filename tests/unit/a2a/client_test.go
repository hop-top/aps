package a2a_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"oss-aps-cli/internal/a2a"
	"oss-aps-cli/internal/core"
)

func TestNewClient_InvalidProfileID(t *testing.T) {
	client, err := a2a.NewClient("", nil)

	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_NilProfile(t *testing.T) {
	client, err := a2a.NewClient("test-profile", nil)

	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_A2ADisabled(t *testing.T) {
	profile := &core.Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		A2A:         &core.A2AConfig{},
	}

	client, err := a2a.NewClient(profile.ID, profile)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Equal(t, a2a.ErrA2ANotEnabled, err)
}

func TestNewClient_ValidProfile(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Profile",
		Capabilities: []string{"a2a"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			ListenAddr:      "127.0.0.1:8081",
			IsolationTier:   "process",
		},
	}

	client, err := a2a.NewClient(profile.ID, profile)

	assert.NoError(t, err)
	assert.NotNil(t, client)
}
