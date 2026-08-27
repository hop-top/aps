package chat

import (
	"testing"
)

func TestLLMResolveOptionsCarriesSamplingOverrides(t *testing.T) {
	temperature := 0.0
	maxTokens := 512
	opts := Options{
		Model:       "gpt-4o",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Effort:      "high",
		Verbosity:   "low",
	}

	resolve := llmResolveOptions(opts)
	if resolve.ModelOverride != "gpt-4o" {
		t.Fatalf("ModelOverride = %q", resolve.ModelOverride)
	}
	if resolve.TemperatureOverride == nil || *resolve.TemperatureOverride != 0 {
		t.Fatalf("TemperatureOverride = %v, want explicit 0", resolve.TemperatureOverride)
	}
	if resolve.MaxTokensOverride == nil || *resolve.MaxTokensOverride != 512 {
		t.Fatalf("MaxTokensOverride = %v, want 512", resolve.MaxTokensOverride)
	}
	if resolve.EffortOverride != "high" {
		t.Fatalf("EffortOverride = %q", resolve.EffortOverride)
	}
	if resolve.VerbosityOverride != "low" {
		t.Fatalf("VerbosityOverride = %q", resolve.VerbosityOverride)
	}
}

func TestLLMResolveOptionsUnsetMeansNoOverride(t *testing.T) {
	resolve := llmResolveOptions(Options{})
	if resolve.TemperatureOverride != nil || resolve.MaxTokensOverride != nil {
		t.Fatalf("unset flags produced overrides: %+v", resolve)
	}
	if resolve.EffortOverride != "" || resolve.VerbosityOverride != "" {
		t.Fatalf("unset string flags produced overrides: %+v", resolve)
	}
}
