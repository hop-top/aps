package skills_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/skills"
)

// MockSecretStore implements the SecretStore interface for testing
type MockSecretStore struct {
	secrets map[string]string
}

func (m *MockSecretStore) Get(key string) (string, error) {
	if val, ok := m.secrets[key]; ok {
		return val, nil
	}
	return "", assert.AnError
}

func (m *MockSecretStore) List() ([]string, error) {
	keys := make([]string, 0, len(m.secrets))
	for k := range m.secrets {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestSecretReplacer_SimpleReplacement(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		LocalModels:        []string{},
		LocalOnly:          false,
		PlaceholderPattern: `\$\{SECRET:([A-Z_]+)\}`,
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"API_KEY": "sk-1234567890abcdef",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test simple replacement (single placeholder)
	args := map[string]interface{}{
		"api_key": "${SECRET:API_KEY}",
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)
	assert.Equal(t, "sk-1234567890abcdef", replaced["api_key"])
}

func TestSecretReplacer_NestedReplacement(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		LocalModels:        []string{},
		LocalOnly:          false,
		PlaceholderPattern: `\$\{SECRET:([A-Z_]+)\}`,
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"API_KEY":    "sk-1234567890abcdef",
			"API_SECRET": "secret-xyz",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test nested structure
	args := map[string]interface{}{
		"auth": map[string]interface{}{
			"key":    "${SECRET:API_KEY}",
			"secret": "${SECRET:API_SECRET}",
		},
		"endpoint": "https://api.example.com",
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)

	auth := replaced["auth"].(map[string]interface{})
	assert.Equal(t, "sk-1234567890abcdef", auth["key"])
	assert.Equal(t, "secret-xyz", auth["secret"])
	assert.Equal(t, "https://api.example.com", replaced["endpoint"])
}

func TestSecretReplacer_ArrayReplacement(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		LocalModels:        []string{},
		LocalOnly:          false,
		PlaceholderPattern: `\$\{SECRET:([A-Z0-9_]+)\}`,
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"TOKEN_1": "token-111",
			"TOKEN_2": "token-222",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test array with placeholders
	args := map[string]interface{}{
		"tokens": []interface{}{
			"${SECRET:TOKEN_1}",
			"${SECRET:TOKEN_2}",
			"plain-text",
		},
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)

	tokens := replaced["tokens"].([]interface{})
	assert.Equal(t, "token-111", tokens[0])
	assert.Equal(t, "token-222", tokens[1])
	assert.Equal(t, "plain-text", tokens[2])
}

func TestSecretReplacer_NoReplacement(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		LocalModels:        []string{},
		LocalOnly:          false,
		PlaceholderPattern: `\$\{SECRET:([A-Z_]+)\}`,
	}

	store := &MockSecretStore{
		secrets: map[string]string{},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with no placeholders
	args := map[string]interface{}{
		"plain": "text",
		"number": 42,
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)
	assert.Equal(t, "text", replaced["plain"])
	assert.Equal(t, 42, replaced["number"])
}

func TestSecretReplacer_Disabled(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled: false,
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"API_KEY": "sk-1234567890abcdef",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test that placeholders are NOT replaced when disabled
	args := map[string]interface{}{
		"api_key": "${SECRET:API_KEY}",
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)

	// Should be unchanged
	assert.Equal(t, "${SECRET:API_KEY}", replaced["api_key"])
}

func TestSecretReplacer_CustomPattern(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		LocalModels:        []string{},
		LocalOnly:          false,
		PlaceholderPattern: `\{\{([A-Z_]+)\}\}`, // Custom pattern: {{KEY}}
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"API_KEY": "sk-1234567890abcdef",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Test custom pattern
	args := map[string]interface{}{
		"api_key": "{{API_KEY}}",
	}

	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", args)
	require.NoError(t, err)
	assert.Equal(t, "sk-1234567890abcdef", replaced["api_key"])
}

func TestSecretReplacer_DeepCopy(t *testing.T) {
	config := &skills.SecretReplacementConfig{
		Enabled:            true,
		PlaceholderPattern: `\$\{SECRET:([A-Z_]+)\}`,
	}

	store := &MockSecretStore{
		secrets: map[string]string{
			"API_KEY": "sk-1234567890abcdef",
		},
	}

	replacer, err := skills.NewSecretReplacer(config, store)
	require.NoError(t, err)

	ctx := context.Background()

	// Original args
	original := map[string]interface{}{
		"api_key": "${SECRET:API_KEY}",
	}

	// Intercept (should not mutate original)
	replaced, err := replacer.InterceptToolCall(ctx, "test-tool", original)
	require.NoError(t, err)

	// Verify original is unchanged
	assert.Equal(t, "${SECRET:API_KEY}", original["api_key"])

	// Verify replaced has real value
	assert.Equal(t, "sk-1234567890abcdef", replaced["api_key"])
}
