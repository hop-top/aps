package chat

import (
	"reflect"
	"testing"

	corechat "hop.top/aps/internal/core/chat"
)

func TestParseParticipantsCombinesPrimaryShorthandAndInvites(t *testing.T) {
	got, err := ParseParticipants("noor,sami", []string{"reza", "aps://profile/kai, noor"})
	if err != nil {
		t.Fatalf("ParseParticipants: %v", err)
	}
	want := []string{"noor", "sami", "reza", "kai"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseParticipants = %#v, want %#v", got, want)
	}
}

func TestParseParticipantsRejectsInvalidProfileRef(t *testing.T) {
	if _, err := ParseParticipants("http://profile/noor", nil); err == nil {
		t.Fatal("expected invalid profile ref error")
	}
}

func TestParseMetaCommand(t *testing.T) {
	tests := []struct {
		line string
		want corechat.MetaCommand
		ok   bool
	}{
		{line: " :auto ", want: corechat.MetaCommandAuto, ok: true},
		{line: ":human", want: corechat.MetaCommandHuman, ok: true},
		{line: ":done", want: corechat.MetaCommandDone, ok: true},
		{line: ":later", want: corechat.MetaCommandNone, ok: false},
		{line: "hello", want: corechat.MetaCommandNone, ok: false},
	}

	for _, tt := range tests {
		got, ok := ParseMetaCommand(tt.line)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("ParseMetaCommand(%q) = %q, %v; want %q, %v", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}
