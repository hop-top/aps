package voice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/voice"
)

func TestPromptGenerator_UsesTemplateWhenSet(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{
		DisplayName: "Alice",
		Voice:       &core.VoiceConfig{PromptTemplate: "You are a pirate."},
	}
	assert.Equal(t, "You are a pirate.", g.Generate(p))
}

func TestPromptGenerator_AutoGeneratesFromPersona(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{
		DisplayName: "Support Bot",
		Persona:     core.Persona{Tone: "friendly", Style: "concise", Risk: "low"},
		Voice:       &core.VoiceConfig{},
	}
	result := g.Generate(p)
	assert.Contains(t, result, "Support Bot")
	assert.Contains(t, result, "warm and approachable")
	assert.Contains(t, result, "brief and to the point")
	assert.Contains(t, result, "Never speculate")
}

func TestPromptGenerator_NilVoiceConfig(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{DisplayName: "Bot", Persona: core.Persona{Tone: "casual"}}
	result := g.Generate(p)
	assert.Contains(t, result, "Bot")
	assert.Contains(t, result, "relaxed and conversational")
}

func TestPromptGenerator_UnknownToneFallsBack(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{DisplayName: "Bot", Persona: core.Persona{Tone: "weird"}}
	result := g.Generate(p)
	assert.Contains(t, result, "Bot")
	// should not panic, unknown tone produces empty string gracefully
}
