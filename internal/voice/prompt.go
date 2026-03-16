package voice

import (
	"fmt"

	"hop.top/aps/internal/core"
)

var toneMap = map[string]string{
	"friendly":     "warm and approachable",
	"professional": "formal and precise",
	"casual":       "relaxed and conversational",
}

var styleMap = map[string]string{
	"concise":  "brief and to the point",
	"detailed": "thorough and comprehensive",
	"casual":   "conversational and informal",
}

var riskMap = map[string]string{
	"low":    "Never speculate. If unsure, say so.",
	"medium": "Use best judgement, flag uncertainty.",
	"high":   "Act decisively with available information.",
}

// PromptGenerator builds a PersonaPlex/Moshi text prompt from an APS Profile.
type PromptGenerator struct{}

func NewPromptGenerator() *PromptGenerator { return &PromptGenerator{} }

// Generate returns the text prompt to inject into the voice backend.
// If profile.Voice.PromptTemplate is set, it takes precedence over auto-generation.
func (g *PromptGenerator) Generate(p *core.Profile) string {
	if p.Voice != nil && p.Voice.PromptTemplate != "" {
		return p.Voice.PromptTemplate
	}
	tone := toneMap[p.Persona.Tone]
	style := styleMap[p.Persona.Style]
	risk := riskMap[p.Persona.Risk]
	return fmt.Sprintf("You are %s. Your communication style is %s and %s. %s",
		p.DisplayName, tone, style, risk)
}
