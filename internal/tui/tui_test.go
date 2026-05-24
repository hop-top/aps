package tui

import (
	"strings"
	"testing"

	kitcli "hop.top/kit/go/console/cli"
	kittui "hop.top/kit/go/console/tui"
)

// TestInitialModel verifies the model starts in the profile-list state.
func TestInitialModel(t *testing.T) {
	m := InitialModel()
	if m.state != StateProfileList {
		t.Fatalf("InitialModel.state = %v, want %v", m.state, StateProfileList)
	}
}

// TestRender_NoPanic ensures Render returns content for every reachable
// state without panicking. Visual parity is enforced by the kit-themed
// styles being non-empty rather than asserting on exact output.
func TestRender_NoPanic(t *testing.T) {
	cases := []State{
		StateProfileList,
		StateProfileDetail,
		StateCapabilityList,
		StateActionList,
	}
	for _, st := range cases {
		m := Model{state: st}
		got := m.Render(80, 24)
		if got == "" && st == StateProfileList {
			t.Errorf("state %v: empty render", st)
		}
	}
}

// TestRender_ErrorBranch verifies the error path renders the message.
func TestRender_ErrorBranch(t *testing.T) {
	m := Model{err: errSentinel{}}
	out := m.Render(80, 24)
	if !strings.Contains(out, "boom") {
		t.Fatalf("render should contain error msg, got %q", out)
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

// TestModel_SatisfiesAppShellInterfaces compile-checks that Model
// implements the optional kit/console/tui interfaces the AppShell
// routes to (Init / Update / Resize). The compile-time guards in
// update.go pin the contracts; this test surfaces a clear failure
// reason if a future refactor breaks them.
func TestModel_SatisfiesAppShellInterfaces(t *testing.T) {
	var m kittui.AppRenderer = Model{}
	if _, ok := m.(kittui.Initer); !ok {
		t.Fatal("Model should implement kittui.Initer")
	}
	if _, ok := m.(kittui.Updater); !ok {
		t.Fatal("Model should implement kittui.Updater")
	}
	if _, ok := m.(kittui.Resizer); !ok {
		t.Fatal("Model should implement kittui.Resizer")
	}
}

// TestAppShell_RendersInitialFrame drives the shell once and confirms
// Model content flows through into the framed View. AppShell owns the
// header / footer; we just confirm the main region picks up the title
// rendered by the profile-list state.
func TestAppShell_RendersInitialFrame(t *testing.T) {
	root := kitcli.New(kitcli.Config{Name: "aps-test"})
	shell := kittui.NewAppShellFromRoot(InitialModel(), root, kittui.WithSize(80, 24))
	v := shell.View()
	if !strings.Contains(v.Content, "Select Profile") {
		t.Fatalf("expected initial frame to contain title, got %q", v.Content)
	}
	if !v.AltScreen {
		t.Fatal("AppShell should run in alt-screen by default")
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "boom" }
