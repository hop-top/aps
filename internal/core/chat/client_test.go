package chat

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildProviderURIEncodesQueryParams(t *testing.T) {
	uri, err := buildProviderURI("openai://gpt-4", "", "key+with&special?chars", "https://api.example.com/v1?already=set")
	if err != nil {
		t.Fatalf("buildProviderURI: %v", err)
	}

	// Result must be a parseable URI; decoded api_key must round-trip exactly.
	parts := strings.SplitN(uri, "?", 2)
	if len(parts) != 2 {
		t.Fatalf("expected query string, got %q", uri)
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		t.Fatalf("parse query %q: %v", parts[1], err)
	}
	if got := values.Get("api_key"); got != "key+with&special?chars" {
		t.Fatalf("api_key round-trip mismatch: got %q want %q", got, "key+with&special?chars")
	}
	if got := values.Get("base_url"); got != "https://api.example.com/v1?already=set" {
		t.Fatalf("base_url round-trip mismatch: got %q want %q", got, "https://api.example.com/v1?already=set")
	}
}

func TestBuildProviderURIPreservesExistingQuery(t *testing.T) {
	uri, err := buildProviderURI("anthropic://claude-3?temperature=0.7", "", "k1", "")
	if err != nil {
		t.Fatalf("buildProviderURI: %v", err)
	}
	parts := strings.SplitN(uri, "?", 2)
	if len(parts) != 2 {
		t.Fatalf("expected query, got %q", uri)
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("temperature"); got != "0.7" {
		t.Fatalf("existing temperature param dropped: got %q", got)
	}
	if got := values.Get("api_key"); got != "k1" {
		t.Fatalf("api_key missing: got %q", got)
	}
}

func TestBuildProviderURIAppliesModelOverride(t *testing.T) {
	uri, err := buildProviderURI("openai://", "gpt-4o", "", "")
	if err != nil {
		t.Fatalf("buildProviderURI: %v", err)
	}
	if !strings.HasPrefix(uri, "openai://gpt-4o") {
		t.Fatalf("model override not applied: got %q", uri)
	}
}

func TestBuildProviderURINoCredentialsLeavesBare(t *testing.T) {
	uri, err := buildProviderURI("ollama://llama3", "", "", "")
	if err != nil {
		t.Fatalf("buildProviderURI: %v", err)
	}
	if uri != "ollama://llama3" {
		t.Fatalf("expected bare URI, got %q", uri)
	}
}
