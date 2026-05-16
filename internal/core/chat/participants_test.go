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
		"You are Reza.",
		"Style: evidence-first",
		"## Other speakers in this conversation",
		"- Noor (aps://profile/noor)",
		"You are the active speaker for this turn (Reza).",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
	// Inactive personas must NOT appear as direct directives.
	for _, banned := range []string{
		"You are Noor.",
	} {
		if strings.Contains(got, banned) {
			t.Fatalf("prompt unexpectedly contains directive %q for inactive speaker:\n%s", banned, got)
		}
	}
}

func TestNewParticipantsRejectsDuplicateID(t *testing.T) {
	noor := &core.Profile{ID: "noor", DisplayName: "Noor"}
	_, err := NewParticipants([]*core.Profile{noor, noor})
	if err == nil {
		t.Fatal("expected duplicate-id error")
	}
}

func TestComposeSystemPromptRejectsUnknownActiveSpeaker(t *testing.T) {
	participants := []Participant{{ID: "noor", URI: "aps://profile/noor", DisplayName: "Noor"}}
	if _, err := ComposeSystemPrompt(participants, "missing"); err == nil {
		t.Fatal("expected unknown active speaker error")
	}
}
