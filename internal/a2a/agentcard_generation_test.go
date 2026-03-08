package a2a

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"hop.top/aps/internal/core"
)

func TestGenerateAgentCardFromProfile_Enabled(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Profile",
		Capabilities: []string{"a2a", "deploy", "query"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			SecurityScheme:  "apikey",
			IsolationTier:   "process",
		},
	}

	card, err := GenerateAgentCardFromProfile(profile)

	assert.NoError(t, err)
	assert.NotNil(t, card)
	assert.Equal(t, "Test Profile", card.Name)
	assert.Equal(t, "1.0.0", card.Version)
	assert.Equal(t, "0.3.4", card.ProtocolVersion)
	assert.Contains(t, card.URL, ":8081")
	assert.Equal(t, "JSONRPC", string(card.PreferredTransport))
}

func TestGenerateAgentCardFromProfile_Disabled(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Profile",
		Capabilities: []string{"deploy", "query"},
		A2A:          &core.A2AConfig{},
	}

	card, err := GenerateAgentCardFromProfile(profile)

	assert.Error(t, err)
	assert.Nil(t, card)
}

func TestGenerateAgentCardFromProfile_NoA2AConfig(t *testing.T) {
	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Profile",
		Capabilities: []string{"deploy", "query"},
	}

	card, err := GenerateAgentCardFromProfile(profile)

	assert.Error(t, err)
	assert.Nil(t, card)
}

func TestGenerateAgentCardForProfile(t *testing.T) {
	card, err := GenerateAgentCardForProfile("test-profile")

	assert.Error(t, err, "profile doesn't exist")
	assert.Nil(t, card)
}

func TestGenerateAgentCardForProfile_InvalidID(t *testing.T) {
	card, err := GenerateAgentCardForProfile("nonexistent-profile")

	assert.Error(t, err, "profile doesn't exist")
	assert.Nil(t, card)
}
