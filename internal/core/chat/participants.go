package chat

import (
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
)

// Participant is the chat-ready view of a profile.
type Participant struct {
	ID           string
	URI          string
	DisplayName  string
	SystemPrompt string
}

// NewParticipant renders a profile into the reusable chat participant shape.
func NewParticipant(profile *core.Profile) (Participant, error) {
	if profile == nil {
		return Participant{}, fmt.Errorf("profile is nil")
	}
	if strings.TrimSpace(profile.ID) == "" {
		return Participant{}, fmt.Errorf("profile id is required")
	}

	return Participant{
		ID:           profile.ID,
		URI:          profile.URI(),
		DisplayName:  participantName(profile),
		SystemPrompt: RenderSystemPrompt(profile),
	}, nil
}

// NewParticipants renders profiles into chat participants, preserving order.
func NewParticipants(profiles []*core.Profile) ([]Participant, error) {
	participants := make([]Participant, 0, len(profiles))
	for _, profile := range profiles {
		participant, err := NewParticipant(profile)
		if err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}
	return participants, nil
}

// ComposeSystemPrompt joins each participant's rendered prompt under speaker
// headers and marks the one participant that should answer this turn.
func ComposeSystemPrompt(participants []Participant, activeSpeakerID string) (string, error) {
	if len(participants) == 0 {
		return "", fmt.Errorf("at least one participant is required")
	}

	active, err := FindParticipant(participants, activeSpeakerID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Active speaker: %s (%s)\n", active.DisplayName, active.URI)
	b.WriteString("Only the active speaker should respond for this turn.\n")

	for _, participant := range participants {
		b.WriteString("\n## Speaker: ")
		b.WriteString(participant.DisplayName)
		b.WriteString(" (")
		b.WriteString(participant.URI)
		b.WriteString(")")
		if participant.ID == active.ID {
			b.WriteString(" [active]")
		}
		b.WriteByte('\n')
		if participant.ID == active.ID {
			b.WriteString("Active speaker: yes\n")
		} else {
			b.WriteString("Active speaker: no\n")
		}
		b.WriteString(strings.TrimSpace(participant.SystemPrompt))
		b.WriteByte('\n')
	}

	return strings.TrimSpace(b.String()), nil
}

// FindParticipant returns the participant with id. Empty id selects the first
// participant, which is the initial speaker for round-robin v1.
func FindParticipant(participants []Participant, id string) (Participant, error) {
	if len(participants) == 0 {
		return Participant{}, fmt.Errorf("at least one participant is required")
	}
	if strings.TrimSpace(id) == "" {
		return participants[0], nil
	}
	for _, participant := range participants {
		if participant.ID == id {
			return participant, nil
		}
	}
	return Participant{}, fmt.Errorf("participant %q not found", id)
}

func participantName(profile *core.Profile) string {
	if name := strings.TrimSpace(profile.DisplayName); name != "" {
		return name
	}
	return profile.ID
}
