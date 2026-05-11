package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
	"hop.top/kit/go/ai/llm"
	"hop.top/kit/go/storage/secret"
)

var ErrUnauthorized = errors.New("unauthorized")

type UnauthorizedError struct {
	Expected []string
}

func (e *UnauthorizedError) Error() string {
	if len(e.Expected) == 0 {
		return ErrUnauthorized.Error()
	}
	return fmt.Sprintf("%s: missing one of %s", ErrUnauthorized, strings.Join(e.Expected, ", "))
}

func (e *UnauthorizedError) Is(target error) bool {
	return target == ErrUnauthorized
}

type ProviderKey struct {
	ProviderURI string
	EnvKey      string
	Value       string
}

func ResolveProviderKey(ctx context.Context, profileID string, providerURIs []string) (ProviderKey, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(providerURIs) == 0 {
		providerURIs = DefaultProviderURIs()
	}

	store, err := core.OpenProfileSecretStore(profileID)
	if err != nil {
		return ProviderKey{}, err
	}

	expected := expectedEnvKeys(providerURIs)
	for _, providerURI := range providerURIs {
		value, err := llm.SecretFor(ctx, store, providerURI)
		if err == nil {
			return ProviderKey{
				ProviderURI: providerURI,
				EnvKey:      llm.EnvKeyFor(providerURI),
				Value:       value,
			}, nil
		}
		if !errors.Is(err, secret.ErrNotFound) {
			return ProviderKey{}, err
		}
	}

	return ProviderKey{}, &UnauthorizedError{Expected: expected}
}

func expectedEnvKeys(providerURIs []string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, providerURI := range providerURIs {
		key := llm.EnvKeyFor(providerURI)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	if !seen[llm.FallbackEnvKey] {
		keys = append(keys, llm.FallbackEnvKey)
	}
	return keys
}
