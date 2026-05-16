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
// Duplicate profile IDs are rejected so the single-active-speaker invariant
// holds and round-robin progression cannot stall on duplicates.
func NewParticipants(profiles []*core.Profile) ([]Participant, error) {
	participants := make([]Participant, 0, len(profiles))
	seen := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		participant, err := NewParticipant(profile)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[participant.ID]; dup {
			return nil, fmt.Errorf("duplicate participant id %q", participant.ID)
		}
		seen[participant.ID] = struct{}{}
		participants = append(participants, participant)
	}
	return participants, nil
}

// RenderSystemPrompt is the single-profile persona renderer used by chat.
func RenderSystemPrompt(profile *core.Profile) string {
	if profile == nil {
		return ""
	}

	var lines []string
	name := participantName(profile)
	lines = append(lines, fmt.Sprintf("You are %s.", name))
	lines = append(lines, fmt.Sprintf("Profile URI: %s", profile.URI()))

	if profile.Persona.Tone != "" {
		lines = append(lines, fmt.Sprintf("Tone: %s", profile.Persona.Tone))
	}
	if profile.Persona.Style != "" {
		lines = append(lines, fmt.Sprintf("Style: %s", profile.Persona.Style))
	}
	if profile.Persona.Risk != "" {
		lines = append(lines, fmt.Sprintf("Risk posture: %s", profile.Persona.Risk))
	}
	if len(profile.Roles) > 0 {
		lines = append(lines, fmt.Sprintf("Roles: %s", strings.Join(profile.Roles, ", ")))
	}
	if profile.Preferences.Language != "" {
		lines = append(lines, fmt.Sprintf("Language: %s", profile.Preferences.Language))
	}
	if profile.Preferences.Timezone != "" {
		lines = append(lines, fmt.Sprintf("Timezone: %s", profile.Preferences.Timezone))
	}

	return strings.Join(lines, "\n")
}

// ComposeSystemPrompt builds a single system prompt where only the active
// speaker receives their persona as direct instructions ("You are X"). Other
// participants are described as third-person metadata so the model does not
// adopt or answer as the wrong speaker.
func ComposeSystemPrompt(participants []Participant, activeSpeakerID string) (string, error) {
	if len(participants) == 0 {
		return "", fmt.Errorf("at least one participant is required")
	}

	active, err := FindParticipant(participants, activeSpeakerID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(active.SystemPrompt))
	b.WriteByte('\n')

	others := make([]Participant, 0, len(participants)-1)
	for _, participant := range participants {
		if participant.ID == active.ID {
			continue
		}
		others = append(others, participant)
	}

	if len(others) > 0 {
		b.WriteString("\n## Other speakers in this conversation\n")
		b.WriteString("These are described for context only. Do not adopt their voice or respond on their behalf.\n")
		for _, participant := range others {
			b.WriteString("\n- ")
			b.WriteString(participant.DisplayName)
			b.WriteString(" (")
			b.WriteString(participant.URI)
			b.WriteString(")\n")
			descriptors := stripDirectivesFromPrompt(participant.SystemPrompt)
			if descriptors != "" {
				b.WriteString(indent(descriptors, "  "))
				b.WriteByte('\n')
			}
		}
	}

	b.WriteString("\n## Turn\n")
	fmt.Fprintf(&b, "You are the active speaker for this turn (%s).\n", active.DisplayName)
	b.WriteString("Respond only as yourself.\n")

	return strings.TrimSpace(b.String()), nil
}

// stripDirectivesFromPrompt removes the second-person "You are X." opener and
// the "Profile URI:" line from a rendered persona, leaving only neutral
// descriptors safe to embed in a section about a non-active participant.
func stripDirectivesFromPrompt(prompt string) string {
	lines := strings.Split(prompt, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "You are ") {
			continue
		}
		if strings.HasPrefix(trimmed, "Profile URI:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func indent(s, prefix string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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
