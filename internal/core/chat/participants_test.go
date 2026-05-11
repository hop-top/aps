package chat

import (
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func TestComposeSystemPromptUsesRenderedPromptsAndActiveSpeaker(t *testing.T) {
	noor := &core.Profile{
		ID:          "noor",
		DisplayName: "Noor",
		Persona:     core.Persona{Tone: "direct", Style: "concise", Risk: "low"},
		Roles:       []string{"reviewer"},
	}
	reza := &core.Profile{
		ID:          "reza",
		DisplayName: "Reza",
		Persona:     core.Persona{Tone: "skeptical", Style: "evidence-first"},
	}

	participants, err := NewParticipants([]*core.Profile{noor, reza})
	if err != nil {
		t.Fatalf("NewParticipants: %v", err)
	}
	got, err := ComposeSystemPrompt(participants, "reza")
	if err != nil {
		t.Fatalf("ComposeSystemPrompt: %v", err)
	}

	for _, want := range []string{
		"Active speaker: Reza (aps://profile/reza)",
		"## Speaker: Noor (aps://profile/noor)",
		"## Speaker: Reza (aps://profile/reza) [active]",
		"Tone: direct",
		"Style: evidence-first",
		"Roles: reviewer",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
}

func TestComposeSystemPromptRejectsUnknownActiveSpeaker(t *testing.T) {
	participants := []Participant{{ID: "noor", URI: "aps://profile/noor", DisplayName: "Noor"}}
	if _, err := ComposeSystemPrompt(participants, "missing"); err == nil {
		t.Fatal("expected unknown active speaker error")
	}
}
