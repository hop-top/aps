package chat

import (
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func TestRenderSystemPromptElidesEmptyFields(t *testing.T) {
	prompt := RenderSystemPrompt(&core.Profile{
		DisplayName: "Reza",
		Persona: core.Persona{
			Tone: "direct",
			Risk: "cautious",
		},
	})

	if !strings.Contains(prompt, "You are Reza.") {
		t.Fatalf("prompt = %q, want display name", prompt)
	}
	if !strings.Contains(prompt, "Tone: direct.") {
		t.Fatalf("prompt = %q, want tone", prompt)
	}
	if strings.Contains(prompt, "Style: .") {
		t.Fatalf("prompt = %q, contains dangling empty style", prompt)
	}
	if !strings.Contains(prompt, "Risk posture: cautious.") {
		t.Fatalf("prompt = %q, want risk posture", prompt)
	}
}

func TestRenderSystemPromptIncludesScopeAndLimits(t *testing.T) {
	prompt := RenderSystemPrompt(&core.Profile{
		DisplayName: "Ops",
		Scope: &core.ScopeConfig{
			FilePatterns: []string{"internal/**"},
			Operations:   []string{"read", "write"},
			Tools:        []string{"go test"},
			Secrets:      []string{"OPENAI_API_KEY"},
			Networks:     []string{"api.openai.com"},
		},
		Limits: core.Limits{
			MaxConcurrency:    2,
			MaxRuntimeMinutes: 30,
		},
	})

	for _, want := range []string{
		"Scope file patterns: internal/**.",
		"Allowed operations: read, write.",
		"Allowed tools: go test.",
		"Allowed secrets: OPENAI_API_KEY.",
		"Allowed networks: api.openai.com.",
		"Max concurrency: 2.",
		"Max runtime minutes: 30.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt = %q, missing %q", prompt, want)
		}
	}
}
