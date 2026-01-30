package a2a

import (
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/stretchr/testify/assert"
)

func TestValidateAgentCard_ValidCard(t *testing.T) {
	card := &a2a.AgentCard{
		Name:    "Test Agent",
		URL:     "http://127.0.0.1:8081",
		Version: "1.0.0",
		Skills: []a2a.AgentSkill{
			{
				ID:   "execute",
				Name: "execute",
			},
		},
	}

	err := validateAgentCard(card)
	assert.NoError(t, err)
}

func TestValidateAgentCard_MissingName(t *testing.T) {
	card := &a2a.AgentCard{
		URL: "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{
			{
				ID:   "execute",
				Name: "execute",
			},
		},
	}

	err := validateAgentCard(card)
	assert.Error(t, err)
	assert.Equal(t, "a2a: invalid agent card: name is required", err.Error())
}

func TestValidateAgentCard_MissingURL(t *testing.T) {
	card := &a2a.AgentCard{
		Name: "Test Agent",
		Skills: []a2a.AgentSkill{
			{
				ID:   "execute",
				Name: "execute",
			},
		},
	}

	err := validateAgentCard(card)
	assert.Error(t, err)
	assert.Equal(t, "a2a: invalid agent card: url is required", err.Error())
}

func TestValidateAgentCard_EmptySkills(t *testing.T) {
	card := &a2a.AgentCard{
		Name:   "Test Agent",
		URL:    "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{},
	}

	err := validateAgentCard(card)
	assert.Error(t, err)
	assert.Equal(t, "a2a: invalid agent card: at least one skill is required", err.Error())
}
