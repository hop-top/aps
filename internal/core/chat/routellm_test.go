package chat

import (
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"
)

func TestResolveLLMConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	systemPath := filepath.Join(dir, "system.yaml")
	userPath := filepath.Join(dir, "user.yaml")
	requireNoError(t, os.WriteFile(systemPath, []byte("default_model: openai://system\nrouters: [system]\nrouter_config:\n  threshold: 0.2\n"), 0600))
	requireNoError(t, os.WriteFile(userPath, []byte("default_model: openai://user\nrouters: [user]\nrouter_config:\n  threshold: 0.3\n"), 0600))

	profile := &core.Profile{
		ID: "agent",
		LLM: &core.LLMConfig{
			DefaultModel: "claude-sonnet-4-5",
			Routers:      []string{"profile"},
			RouterConfig: map[string]any{"threshold": 0.8},
			Fallback:     []string{"gpt-4o-mini"},
		},
	}

	resolved, err := ResolveLLMConfig(profile, LLMResolveOptions{
		SystemConfigPath: systemPath,
		UserConfigPath:   userPath,
		ModelOverride:    "gemini://cli-model",
	})
	requireNoError(t, err)

	if resolved.ProviderURI != "routellm://profile:0.8" {
		t.Fatalf("ProviderURI = %q", resolved.ProviderURI)
	}
	if resolved.Model != "gemini://cli-model" {
		t.Fatalf("Model = %q", resolved.Model)
	}
	if len(resolved.FallbackURIs) != 1 || resolved.FallbackURIs[0] != "openai://gpt-4o-mini" {
		t.Fatalf("FallbackURIs = %#v", resolved.FallbackURIs)
	}
	candidates := CandidateProviderURIs(resolved)
	if len(candidates) == 0 || candidates[0] != "gemini://cli-model" {
		t.Fatalf("CandidateProviderURIs = %#v", candidates)
	}
}

func TestResolveLLMConfigInfersProviderFromModel(t *testing.T) {
	resolved, err := ResolveLLMConfig(&core.Profile{
		ID:  "agent",
		LLM: &core.LLMConfig{DefaultModel: "claude-sonnet-4-5"},
	}, LLMResolveOptions{
		SystemConfigPath: filepath.Join(t.TempDir(), "missing-system.yaml"),
		UserConfigPath:   filepath.Join(t.TempDir(), "missing-user.yaml"),
	})
	requireNoError(t, err)

	if resolved.ProviderURI != "anthropic://claude-sonnet-4-5" {
		t.Fatalf("ProviderURI = %q", resolved.ProviderURI)
	}
	if resolved.Model != "claude-sonnet-4-5" {
		t.Fatalf("Model = %q", resolved.Model)
	}
}
