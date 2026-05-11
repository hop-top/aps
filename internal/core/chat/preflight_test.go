package chat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func TestResolveProviderKeyUsesProfileSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("LLM_API_KEY", "")
	requireNoError(t, core.CreateProfile("agent", core.Profile{DisplayName: "Agent"}))
	requireNoError(t, os.WriteFile(filepath.Join(home, "profiles", "agent", "secrets.env"), []byte("OPENAI_API_KEY=sk-test\n"), 0600))

	key, err := ResolveProviderKey(context.Background(), "agent", []string{"openai://gpt-4o"})
	requireNoError(t, err)
	if key.EnvKey != "OPENAI_API_KEY" || key.Value != "sk-test" {
		t.Fatalf("key = %#v, want OPENAI_API_KEY from profile", key)
	}
}

func TestResolveProviderKeyUnauthorizedListsExpectedKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("LLM_API_KEY", "")
	requireNoError(t, core.CreateProfile("agent", core.Profile{DisplayName: "Agent"}))

	_, err := ResolveProviderKey(context.Background(), "agent", []string{"anthropic://claude-3-5", "openai://gpt-4o"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	for _, want := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "LLM_API_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, missing %s", err, want)
		}
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
