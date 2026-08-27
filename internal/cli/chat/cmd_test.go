package chat

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestNewCommandRegistersSamplingFlags(t *testing.T) {
	cmd := NewCommand()
	for _, name := range []string{"temperature", "max-tokens", "effort", "verbosity"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("flag --%s not registered", name)
		}
	}
}

func TestApplySamplingOverridesUnsetMeansNoOverride(t *testing.T) {
	flags := pflag.NewFlagSet("chat", pflag.ContinueOnError)
	var temperature float64
	var maxTokens int
	flags.Float64Var(&temperature, "temperature", 0, "")
	flags.IntVar(&maxTokens, "max-tokens", 0, "")
	requireNoError(t, flags.Parse(nil))

	var opts Options
	applySamplingOverrides(flags, &opts, &temperature, &maxTokens)
	if opts.Temperature != nil {
		t.Fatalf("Temperature = %v, want nil when flag unset", *opts.Temperature)
	}
	if opts.MaxTokens != nil {
		t.Fatalf("MaxTokens = %v, want nil when flag unset", *opts.MaxTokens)
	}
}

func TestApplySamplingOverridesExplicitZeroOverrides(t *testing.T) {
	flags := pflag.NewFlagSet("chat", pflag.ContinueOnError)
	var temperature float64
	var maxTokens int
	flags.Float64Var(&temperature, "temperature", 0, "")
	flags.IntVar(&maxTokens, "max-tokens", 0, "")
	requireNoError(t, flags.Parse([]string{"--temperature", "0", "--max-tokens", "256"}))

	var opts Options
	applySamplingOverrides(flags, &opts, &temperature, &maxTokens)
	if opts.Temperature == nil || *opts.Temperature != 0 {
		t.Fatalf("Temperature = %v, want explicit 0", opts.Temperature)
	}
	if opts.MaxTokens == nil || *opts.MaxTokens != 256 {
		t.Fatalf("MaxTokens = %v, want 256", opts.MaxTokens)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
