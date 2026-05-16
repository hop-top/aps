package chat

import "testing"

func TestRoundRobinPolicy(t *testing.T) {
	participants := []Participant{
		{ID: "noor"},
		{ID: "reza"},
		{ID: "sami"},
	}
	policy := NewRoundRobinPolicy()

	tests := []struct {
		last string
		want string
	}{
		{last: "", want: "noor"},
		{last: "noor", want: "reza"},
		{last: "reza", want: "sami"},
		{last: "sami", want: "noor"},
		{last: "missing", want: "noor"},
	}

	for _, tt := range tests {
		got, err := policy.Next(TurnState{Participants: participants, LastSpeakerID: tt.last})
		if err != nil {
			t.Fatalf("Next(%q): %v", tt.last, err)
		}
		if got.ID != tt.want {
			t.Fatalf("Next(%q) = %q, want %q", tt.last, got.ID, tt.want)
		}
	}
}

func TestAutoSessionCapsAutoTurnsAndHonorsMetaCommands(t *testing.T) {
	session := NewAutoSession(2)
	if session.MaxAutoTurns != 2 {
		t.Fatalf("MaxAutoTurns = %d, want 2", session.MaxAutoTurns)
	}

	session.Apply(MetaCommandAuto)
	if !session.ShouldAutoContinue() {
		t.Fatal("expected auto mode to continue after :auto")
	}
	if !session.RecordAutoTurn() {
		t.Fatal("expected first auto turn to continue")
	}
	if session.RecordAutoTurn() {
		t.Fatal("expected second auto turn to hit cap")
	}
	if session.Mode != ControlModeHuman {
		t.Fatalf("Mode = %q, want human after cap", session.Mode)
	}

	session.Apply(MetaCommandAuto)
	session.Apply(MetaCommandDone)
	if session.ShouldAutoContinue() {
		t.Fatal("expected :done to stop auto mode")
	}
	if !session.Done {
		t.Fatal("expected Done after :done")
	}
}

func TestNewAutoSessionDefaultCap(t *testing.T) {
	session := NewAutoSession(0)
	if session.MaxAutoTurns != DefaultMaxAutoTurns {
		t.Fatalf("MaxAutoTurns = %d, want %d", session.MaxAutoTurns, DefaultMaxAutoTurns)
	}
}
