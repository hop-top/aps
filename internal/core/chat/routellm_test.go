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

func TestResolveLLMConfigSamplingPrecedence(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.yaml")
	requireNoError(t, os.WriteFile(userPath, []byte("temperature: 0.9\nmax_tokens: 512\nreasoning_effort: low\nverbosity: low\n"), 0o600))

	zero := 0.0
	profile := &core.Profile{
		ID: "agent",
		LLM: &core.LLMConfig{
			Temperature:     &zero,
			MaxTokens:       2048,
			ReasoningEffort: "high",
		},
	}

	resolved, err := ResolveLLMConfig(profile, LLMResolveOptions{
		SystemConfigPath: filepath.Join(dir, "missing-system.yaml"),
		UserConfigPath:   userPath,
	})
	requireNoError(t, err)

	if resolved.Config.Temperature == nil || *resolved.Config.Temperature != 0 {
		t.Fatalf("Temperature = %v, want explicit 0 from profile", resolved.Config.Temperature)
	}
	if resolved.Config.MaxTokens != 2048 {
		t.Fatalf("MaxTokens = %d, want 2048", resolved.Config.MaxTokens)
	}
	if resolved.Config.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", resolved.Config.ReasoningEffort)
	}
	if resolved.Config.Verbosity != "low" {
		t.Fatalf("Verbosity = %q, want low from user config", resolved.Config.Verbosity)
	}
}

func TestResolveLLMConfigSamplingOverrides(t *testing.T) {
	dir := t.TempDir()
	fileTemp := 0.9
	profile := &core.Profile{
		ID: "agent",
		LLM: &core.LLMConfig{
			Temperature:     &fileTemp,
			MaxTokens:       2048,
			ReasoningEffort: "high",
			Verbosity:       "high",
		},
	}

	zero := 0.0
	maxTokens := 128
	resolved, err := ResolveLLMConfig(profile, LLMResolveOptions{
		SystemConfigPath:    filepath.Join(dir, "missing-system.yaml"),
		UserConfigPath:      filepath.Join(dir, "missing-user.yaml"),
		TemperatureOverride: &zero,
		MaxTokensOverride:   &maxTokens,
		EffortOverride:      "minimal",
		VerbosityOverride:   "medium",
	})
	requireNoError(t, err)

	if resolved.Config.Temperature == nil || *resolved.Config.Temperature != 0 {
		t.Fatalf("Temperature = %v, want flag override 0", resolved.Config.Temperature)
	}
	if resolved.Config.MaxTokens != 128 {
		t.Fatalf("MaxTokens = %d, want flag override 128", resolved.Config.MaxTokens)
	}
	if resolved.Config.ReasoningEffort != "minimal" {
		t.Fatalf("ReasoningEffort = %q, want minimal", resolved.Config.ReasoningEffort)
	}
	if resolved.Config.Verbosity != "medium" {
		t.Fatalf("Verbosity = %q, want medium", resolved.Config.Verbosity)
	}
}

func TestResolveLLMConfigRejectsInvalidSampling(t *testing.T) {
	dir := t.TempDir()
	opts := func() LLMResolveOptions {
		return LLMResolveOptions{
			SystemConfigPath: filepath.Join(dir, "missing-system.yaml"),
			UserConfigPath:   filepath.Join(dir, "missing-user.yaml"),
		}
	}
	badTemp := 2.5
	negTemp := -0.1
	cases := []struct {
		name string
		llm  core.LLMConfig
	}{
		{"temperature above range", core.LLMConfig{Temperature: &badTemp}},
		{"temperature below range", core.LLMConfig{Temperature: &negTemp}},
		{"negative max_tokens", core.LLMConfig{MaxTokens: -1}},
		{"unknown reasoning_effort", core.LLMConfig{ReasoningEffort: "frantic"}},
		{"unknown verbosity", core.LLMConfig{Verbosity: "loud"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			llmCfg := tc.llm
			_, err := ResolveLLMConfig(&core.Profile{ID: "agent", LLM: &llmCfg}, opts())
			if err == nil {
				t.Fatalf("ResolveLLMConfig accepted %+v", tc.llm)
			}
		})
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
