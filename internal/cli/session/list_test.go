package session

import (
	"testing"

	"hop.top/aps/internal/core/session"
)

func TestMatchesTypeFilter(t *testing.T) {
	std := &session.SessionInfo{ID: "a"} // zero-value Type → standard
	voc := &session.SessionInfo{ID: "b", Type: session.SessionTypeVoice}

	cases := []struct {
		name    string
		s       *session.SessionInfo
		filter  string
		matches bool
	}{
		{"empty filter matches standard", std, "", true},
		{"empty filter matches voice", voc, "", true},
		{"standard filter matches zero-value", std, "standard", true},
		{"standard filter excludes voice", voc, "standard", false},
		{"voice filter matches voice", voc, "voice", true},
		{"voice filter excludes standard", std, "voice", false},
		{"unknown filter excludes both std", std, "bogus", false},
		{"unknown filter excludes both voc", voc, "bogus", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesTypeFilter(tc.s, tc.filter); got != tc.matches {
				t.Errorf("matchesTypeFilter(%q) = %v, want %v", tc.filter, got, tc.matches)
			}
		})
	}
}

func TestValidateTypeFilter(t *testing.T) {
	cases := []struct {
		filter  string
		wantErr bool
	}{
		{"", false},
		{"standard", false},
		{"voice", false},
		{"bogus", true},
		{"VOICE", true}, // case-sensitive by design
	}
	for _, tc := range cases {
		err := validateTypeFilter(tc.filter)
		if tc.wantErr && err == nil {
			t.Errorf("validateTypeFilter(%q) want error, got nil", tc.filter)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateTypeFilter(%q) want nil, got %v", tc.filter, err)
		}
	}
}

func TestFilterSessions_ByType(t *testing.T) {
	in := []*session.SessionInfo{
		{ID: "a", ProfileID: "p1"},
		{ID: "b", ProfileID: "p1", Type: session.SessionTypeVoice},
		{ID: "c", ProfileID: "p1", Type: session.SessionTypeVoice},
	}
	if got := filterSessions(in, "", "", "", "", ""); len(got) != 3 {
		t.Errorf("no filter: got %d, want 3", len(got))
	}
	if got := filterSessions(in, "", "", "", "", "voice"); len(got) != 2 {
		t.Errorf("voice filter: got %d, want 2", len(got))
	}
	if got := filterSessions(in, "", "", "", "", "standard"); len(got) != 1 {
		t.Errorf("standard filter: got %d, want 1", len(got))
	}
}
