package workspace

import (
	"testing"

	"github.com/spf13/cobra"

	collab "hop.top/aps/internal/core/collaboration"
)

// newCreateWithProfile mimics the global --profile flag the root command
// declares persistently.
func newCreateWithProfile() *cobra.Command {
	c := NewCreateCmd()
	c.Flags().String("profile", "", "")
	return c
}

func TestCreate_NonTTYDefaultsWithoutPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	prev := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = prev })

	cmd := newCreateWithProfile()
	cmd.SetArgs([]string{"demo-ws", "--profile", "alice"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("create without TTY: %v", err)
	}

	mgr, err := getManager()
	if err != nil {
		t.Fatal(err)
	}
	list, err := mgr.List(cmd.Context(), collab.ListOptions{})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	ws := list[0]
	if ws.Config.Description != "" || ws.Config.DefaultPolicy != collab.ResolutionStrategy("priority") {
		t.Fatalf("defaults: desc=%q policy=%q", ws.Config.Description, ws.Config.DefaultPolicy)
	}
}

func TestShow_ByNameAfterCreate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	prev := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = prev })

	create := newCreateWithProfile()
	create.SetArgs([]string{"demo-ws", "--profile", "alice"})
	if err := create.Execute(); err != nil {
		t.Fatalf("create: %v", err)
	}
	mgr, err := getManager()
	if err != nil {
		t.Fatal(err)
	}
	id, err := mgr.Resolve(create.Context(), "demo-ws")
	if err != nil {
		t.Fatal(err)
	}

	for _, ref := range []string{"demo-ws", id} {
		show := NewShowCmd()
		show.SetArgs([]string{ref, "--json"})
		if err := show.Execute(); err != nil {
			t.Fatalf("show %q: %v", ref, err)
		}
	}

	show := NewShowCmd()
	show.SetArgs([]string{"nope"})
	if err := show.Execute(); err == nil {
		t.Fatal("show nope: want not-found error")
	}
}
