package chat

import (
	"context"
	"fmt"
	"strings"

	"hop.top/kit/go/ai/llm"
	_ "hop.top/kit/go/ai/llm/anthropic"
	_ "hop.top/kit/go/ai/llm/google"
	_ "hop.top/kit/go/ai/llm/ollama"
	_ "hop.top/kit/go/ai/llm/routellm"
	_ "hop.top/kit/go/ai/llm/triton"
)

func NewLLMClient(ctx context.Context, profileID string, resolved ResolvedLLMConfig) (*llm.Client, error) {
	if resolved.ProviderURI == "" {
		return nil, fmt.Errorf("chat llm provider is not configured")
	}
	key, err := ResolveProviderKey(ctx, profileID, CandidateProviderURIs(resolved))
	if err != nil {
		return nil, err
	}

	primary, err := newProvider(resolved.ProviderURI, resolved.Model, key.Value, resolved)
	if err != nil {
		return nil, err
	}

	var opts []llm.Option
	for _, fbURI := range resolved.FallbackURIs {
		fb, err := newProvider(fbURI, "", key.Value, resolved)
		if err != nil {
			return nil, err
		}
		opts = append(opts, llm.WithFallback(fb))
	}
	return llm.NewClient(primary, opts...), nil
}

func newProvider(providerURI, model, apiKey string, resolved ResolvedLLMConfig) (llm.Provider, error) {
	uri := providerURI
	if model != "" {
		if parsed, err := llm.ParseURI(uri); err == nil && parsed.Model == "" {
			uri = parsed.Scheme + "://" + model
		}
	}
	var params []string
	if apiKey != "" {
		params = append(params, "api_key="+apiKey)
	}
	if resolved.Config.BaseURL != "" {
		params = append(params, "base_url="+resolved.Config.BaseURL)
	}
	if len(params) > 0 {
		sep := "?"
		if strings.Contains(uri, "?") {
			sep = "&"
		}
		uri += sep + strings.Join(params, "&")
	}
	return llm.Resolve(uri)
}
