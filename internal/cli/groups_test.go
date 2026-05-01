package cli

import (
	"strings"
	"testing"
)

// TestEveryTopLevelCommandHasGroup asserts every registered top-level
// subcommand resolves to a non-empty GroupID via commandGroups. A
// command in the registered set but missing from commandGroups is a
// build-time omission — fail loudly so the help layout doesn't drift.
func TestEveryTopLevelCommandHasGroup(t *testing.T) {
	applyCommandGroups()

	for _, c := range rootCmd.Commands() {
		name := c.Name()
		// Skip kit-injected built-ins that kit assigns its own group.
		if name == "completion" || name == "help" {
			continue
		}
		if c.GroupID == "" {
			t.Errorf("top-level command %q has no GroupID — add to commandGroups in groups.go", name)
		}
	}
}

// TestCommandGroupsCoversRegistered asserts every entry in
// commandGroups maps to a real registered subcommand. A stale entry
// (typo, deleted command) is silently ignored at runtime; this test
// surfaces it.
func TestCommandGroupsCoversRegistered(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		registered[c.Name()] = true
	}
	for name := range commandGroups {
		if !registered[name] {
			t.Errorf("commandGroups entry %q has no registered top-level command", name)
		}
	}
}

// TestGroupIDsMatchHelpConfig asserts every group ID used in
// commandGroups is registered in HelpConfig.Groups (or is the
// kit-builtin "management"). A typo on a group ID otherwise produces
// commands that fall outside any rendered section.
func TestGroupIDsMatchHelpConfig(t *testing.T) {
	declared := map[string]bool{
		"management": true, // kit auto-registers
	}
	for _, g := range root.Config.Help.Groups {
		declared[g.ID] = true
	}
	for name, gid := range commandGroups {
		if !declared[gid] {
			t.Errorf("command %q assigned to undeclared group %q", name, gid)
		}
	}
}

// TestCommandGroupsContent locks the post-refactor command-to-group
// assignment. Update this list deliberately when moving a command
// between groups so the convention drift is visible in PR review.
func TestCommandGroupsContent(t *testing.T) {
	want := map[string][]string{
		"interact":   {"run", "serve", "voice", "session"},
		"organize":   {"profile", "capability", "bundle", "squad", "workspace", "contact"},
		"pipelines":  {"a2a", "acp", "adapter", "webhook", "directory", "action"},
		"security":   {"identity", "policy"},
		"instance":   {"observability"},
		"management": {"alias", "docs", "env", "migrate", "upgrade", "toolspec", "version"},
	}
	for groupID, expectMembers := range want {
		got := []string{}
		for name, gid := range commandGroups {
			if gid == groupID {
				got = append(got, name)
			}
		}
		if len(got) != len(expectMembers) {
			t.Errorf("group %q: got %d members [%s], want %d [%s]",
				groupID, len(got), strings.Join(got, ","),
				len(expectMembers), strings.Join(expectMembers, ","))
			continue
		}
		gotSet := map[string]bool{}
		for _, m := range got {
			gotSet[m] = true
		}
		for _, m := range expectMembers {
			if !gotSet[m] {
				t.Errorf("group %q missing member %q", groupID, m)
			}
		}
	}
}
