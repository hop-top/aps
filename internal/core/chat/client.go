package chat

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"hop.top/kit/go/ai/llm"
	_ "hop.top/kit/go/ai/llm/anthropic"
	llmerrors "hop.top/kit/go/ai/llm/errors"
	_ "hop.top/kit/go/ai/llm/google"
	_ "hop.top/kit/go/ai/llm/ollama"
	_ "hop.top/kit/go/ai/llm/openai" // registers openai; explicit, not via routellm
	_ "hop.top/kit/go/ai/llm/routellm"
	_ "hop.top/kit/go/ai/llm/triton"
)

func NewLLMClient(ctx context.Context, profileID string, resolved ResolvedLLMConfig) (*llm.Client, error) {
	if resolved.ProviderURI == "" {
		return nil, fmt.Errorf("chat llm provider is not configured")
	}

	primary, err := buildProvider(ctx, profileID, resolved.ProviderURI, resolved.Model, resolved)
	if err != nil {
		return nil, err
	}

	var opts []llm.Option
	for _, fbURI := range resolved.FallbackURIs {
		fb, err := buildProvider(ctx, profileID, fbURI, "", resolved)
		if err != nil {
			// A missing credential for a fallback provider is non-fatal:
			// the chain still works as long as the primary (or another
			// fallback) has one. Skip the unauthorized fallback rather
			// than failing the whole client construction.
			var unauth *UnauthorizedError
			if errors.As(err, &unauth) {
				continue
			}
			return nil, err
		}
		opts = append(opts, llm.WithFallback(fb))
	}
	return llm.NewClient(primary, opts...), nil
}

// buildProvider resolves the credential for providerURI specifically (not
// reusing the primary's key) and constructs the provider via llm.Resolve with
// URL-encoded query parameters.
func buildProvider(ctx context.Context, profileID, providerURI, model string, resolved ResolvedLLMConfig) (llm.Provider, error) {
	if err := checkProviderRegistered(providerURI); err != nil {
		return nil, err
	}
	key, err := ResolveProviderKeyFor(ctx, profileID, providerURI)
	if err != nil {
		return nil, err
	}
	uri, err := buildProviderURI(providerURI, model, key.Value, resolved.Config.BaseURL)
	if err != nil {
		return nil, err
	}
	provider, err := llm.Resolve(uri)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %s: %w", providerURI, err)
	}
	return provider, nil
}

// checkProviderRegistered fails with the scheme and the registered schemes
// when providerURI names no adapter, before any credential lookup would mask
// the real problem. Resolution only; nothing is dialed.
func checkProviderRegistered(providerURI string) error {
	parsed, err := llm.ParseURI(providerURI)
	if err != nil {
		return fmt.Errorf("invalid chat llm provider %q: %w", providerURI, err)
	}
	_, err = llm.Resolve(parsed.Scheme + "://?api_key=probe")
	var nf *llmerrors.ErrProviderNotFound
	if errors.As(err, &nf) {
		return fmt.Errorf("unknown chat llm provider %q (available: %s)",
			parsed.Scheme, strings.Join(llm.Schemes(), ", "))
	}
	return nil
}

func buildProviderURI(providerURI, model, apiKey, baseURL string) (string, error) {
	uri := providerURI
	if model != "" {
		if parsed, err := llm.ParseURI(uri); err == nil && parsed.Model == "" {
			uri = parsed.Scheme + "://" + model
		}
	}

	base, query := splitURIQuery(uri)
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", fmt.Errorf("parse provider uri query: %w", err)
	}
	if apiKey != "" {
		values.Set("api_key", apiKey)
	}
	if baseURL != "" {
		values.Set("base_url", baseURL)
	}
	encoded := values.Encode()
	if encoded == "" {
		return base, nil
	}
	return base + "?" + encoded, nil
}

func splitURIQuery(uri string) (string, string) {
	if idx := strings.Index(uri, "?"); idx >= 0 {
		return uri[:idx], uri[idx+1:]
	}
	return uri, ""
}
