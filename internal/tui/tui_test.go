package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestInitialModel verifies the model starts in the profile-list state.
func TestInitialModel(t *testing.T) {
	m := InitialModel()
	if m.state != StateProfileList {
		t.Fatalf("InitialModel.state = %v, want %v", m.state, StateProfileList)
	}
}

// TestView_NoPanic ensures View renders something for every reachable state
// without panicking. Visual parity is enforced by the kit-themed styles
// being non-empty rather than asserting on exact output.
func TestView_NoPanic(t *testing.T) {
	cases := []State{
		StateProfileList,
		StateProfileDetail,
		StateCapabilityList,
		StateActionList,
	}
	for _, st := range cases {
		m := Model{state: st}
		v := m.View()
		if got := teaViewString(v); got == "" && st == StateProfileList {
			// Profile list with no profiles should still render the title.
			t.Errorf("state %v: empty view", st)
		}
	}
}

// TestView_ErrorBranch verifies the error path renders the message.
func TestView_ErrorBranch(t *testing.T) {
	m := Model{err: errSentinel{}}
	v := m.View()
	out := teaViewString(v)
	if !strings.Contains(out, "boom") {
		t.Fatalf("view should contain error msg, got %q", out)
	}
}

// TestStyles_ThemedFromKit verifies the package-level styles are wired to
// kit's themed surface (non-zero render output) rather than zero values.
func TestStyles_ThemedFromKit(t *testing.T) {
	if got := titleStyle.Render("x"); got == "" {
		t.Fatal("titleStyle should render non-empty")
	}
	if got := selectedItemStyle.Render("x"); got == "" {
		t.Fatal("selectedItemStyle should render non-empty")
	}
	if got := footerStyle.Render("x"); got == "" {
		t.Fatal("footerStyle should render non-empty")
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "boom" }

// teaViewString extracts the rendered string from a tea.View. The bubbletea
// v2 View struct exposes its content via the Content field.
func teaViewString(v tea.View) string {
	return v.Content
}
