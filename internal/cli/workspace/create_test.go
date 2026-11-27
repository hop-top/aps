package workspace

import (
	"testing"

	collab "hop.top/aps/internal/core/collaboration"
)

func TestCreate_NonTTYDefaultsWithoutPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	prev := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = prev })

	cmd := NewCreateCmd()
	cmd.Flags().String("profile", "", "")
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
