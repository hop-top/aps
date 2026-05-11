package chat

import (
	"fmt"
	"strings"
)

const DefaultMaxAutoTurns = 10

// TurnPolicy chooses exactly one participant for the next model turn.
type TurnPolicy interface {
	Next(TurnState) (Participant, error)
}

// TurnState is the policy input shared by CLI, TUI, and future transports.
type TurnState struct {
	Participants  []Participant
	LastSpeakerID string
}

// RoundRobinPolicy advances through participants in order.
type RoundRobinPolicy struct{}

func NewRoundRobinPolicy() RoundRobinPolicy {
	return RoundRobinPolicy{}
}

func (RoundRobinPolicy) Next(state TurnState) (Participant, error) {
	if len(state.Participants) == 0 {
		return Participant{}, fmt.Errorf("at least one participant is required")
	}
	if strings.TrimSpace(state.LastSpeakerID) == "" {
		return state.Participants[0], nil
	}
	for i, participant := range state.Participants {
		if participant.ID == state.LastSpeakerID {
			return state.Participants[(i+1)%len(state.Participants)], nil
		}
	}
	return state.Participants[0], nil
}

type MetaCommand string

const (
	MetaCommandNone  MetaCommand = ""
	MetaCommandAuto  MetaCommand = "auto"
	MetaCommandHuman MetaCommand = "human"
	MetaCommandDone  MetaCommand = "done"
)

type ControlMode string

const (
	ControlModeHuman ControlMode = "human"
	ControlModeAuto  ControlMode = "auto"
)

// AutoSession tracks autonomous turn-taking and enforces the auto-turn cap.
type AutoSession struct {
	Mode         ControlMode
	AutoTurns    int
	MaxAutoTurns int
	Done         bool
}

func NewAutoSession(maxAutoTurns int) AutoSession {
	if maxAutoTurns <= 0 {
		maxAutoTurns = DefaultMaxAutoTurns
	}
	return AutoSession{
		Mode:         ControlModeHuman,
		MaxAutoTurns: maxAutoTurns,
	}
}

func (s *AutoSession) Apply(command MetaCommand) {
	switch command {
	case MetaCommandAuto:
		s.Mode = ControlModeAuto
		s.AutoTurns = 0
		s.Done = false
	case MetaCommandHuman:
		s.Mode = ControlModeHuman
		s.AutoTurns = 0
	case MetaCommandDone:
		s.Mode = ControlModeHuman
		s.Done = true
	}
}

func (s *AutoSession) ShouldAutoContinue() bool {
	return s.Mode == ControlModeAuto && !s.Done && s.AutoTurns < s.maxAutoTurns()
}

// RecordAutoTurn records one completed autonomous speaker turn. It returns
// false when this turn exhausted the cap and control has returned to human.
func (s *AutoSession) RecordAutoTurn() bool {
	if s.Mode != ControlModeAuto || s.Done {
		return false
	}
	s.AutoTurns++
	if s.AutoTurns >= s.maxAutoTurns() {
		s.Mode = ControlModeHuman
		return false
	}
	return true
}

func (s *AutoSession) maxAutoTurns() int {
	if s.MaxAutoTurns <= 0 {
		return DefaultMaxAutoTurns
	}
	return s.MaxAutoTurns
}
