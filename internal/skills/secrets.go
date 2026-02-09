package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// SecretReplacer handles intelligent secret injection into tool calls
type SecretReplacer struct {
	config      *SecretReplacementConfig
	secretStore SecretStore // Interface to profile secrets (age/secretspec)
	pattern     *regexp.Regexp
}

// SecretStore interface for retrieving secrets from profile
type SecretStore interface {
	Get(key string) (string, error)
	List() ([]string, error)
}

// NewSecretReplacer creates a new secret replacer
func NewSecretReplacer(config *SecretReplacementConfig, store SecretStore) (*SecretReplacer, error) {
	pattern := config.PlaceholderPattern
	if pattern == "" {
		pattern = `\$\{SECRET:([A-Z_]+)\}` // Default pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid placeholder pattern: %w", err)
	}

	return &SecretReplacer{
		config:      config,
		secretStore: store,
		pattern:     re,
	}, nil
}

// InterceptToolCall intercepts a tool call and replaces secret placeholders
func (sr *SecretReplacer) InterceptToolCall(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	if !sr.config.Enabled {
		return args, nil
	}

	// Deep copy args to avoid mutating original
	replaced := sr.deepCopyMap(args)

	// Find and replace placeholders
	if err := sr.replaceInMap(ctx, replaced); err != nil {
		return nil, fmt.Errorf("failed to replace secrets: %w", err)
	}

	return replaced, nil
}

// replaceInMap recursively finds and replaces secret placeholders in a map
func (sr *SecretReplacer) replaceInMap(ctx context.Context, m map[string]interface{}) error {
	for key, value := range m {
		switch v := value.(type) {
		case string:
			// Check if string contains placeholders
			if sr.pattern.MatchString(v) {
				replaced, err := sr.replaceInString(ctx, v)
				if err != nil {
					return err
				}
				m[key] = replaced
			}

		case map[string]interface{}:
			// Recursively replace in nested maps
			if err := sr.replaceInMap(ctx, v); err != nil {
				return err
			}

		case []interface{}:
			// Replace in arrays
			replaced, err := sr.replaceInSliceNew(ctx, v)
			if err != nil {
				return err
			}
			m[key] = replaced
		}
	}

	return nil
}

// replaceInSlice replaces placeholders in a slice
func (sr *SecretReplacer) replaceInSlice(ctx context.Context, slice []interface{}) error {
	for i, item := range slice {
		switch v := item.(type) {
		case string:
			if sr.pattern.MatchString(v) {
				replaced, err := sr.replaceInString(ctx, v)
				if err != nil {
					return err
				}
				slice[i] = replaced
			}

		case map[string]interface{}:
			if err := sr.replaceInMap(ctx, v); err != nil {
				return err
			}

		case []interface{}:
			if err := sr.replaceInSlice(ctx, v); err != nil {
				return err
			}
		}
	}

	return nil
}

// replaceInSliceNew replaces placeholders in a slice and returns a new slice
func (sr *SecretReplacer) replaceInSliceNew(ctx context.Context, slice []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(slice))
	for i, item := range slice {
		switch v := item.(type) {
		case string:
			if sr.pattern.MatchString(v) {
				replaced, err := sr.replaceInString(ctx, v)
				if err != nil {
					return nil, err
				}
				result[i] = replaced
			} else {
				result[i] = v
			}

		case map[string]interface{}:
			if err := sr.replaceInMap(ctx, v); err != nil {
				return nil, err
			}
			result[i] = v

		case []interface{}:
			replaced, err := sr.replaceInSliceNew(ctx, v)
			if err != nil {
				return nil, err
			}
			result[i] = replaced

		default:
			result[i] = v
		}
	}

	return result, nil
}

// replaceInString replaces secret placeholders in a single string
func (sr *SecretReplacer) replaceInString(ctx context.Context, s string) (string, error) {
	// Find all matches
	matches := sr.pattern.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return s, nil
	}

	// Check if we need intelligent replacement (multiple placeholders or embedded in text)
	needsIntelligence := len(matches) > 1 || !sr.isOnlyPlaceholder(s)

	if needsIntelligence {
		return sr.intelligentReplace(ctx, s, matches)
	}

	// Simple replacement (single placeholder, exact match)
	secretKey := matches[0][1]
	secretValue, err := sr.secretStore.Get(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretKey, err)
	}

	return secretValue, nil
}

// intelligentReplace uses LLM to intelligently replace secrets in context
func (sr *SecretReplacer) intelligentReplace(ctx context.Context, text string, matches [][]string) (string, error) {
	// Build map of available secrets
	secrets := make(map[string]string)
	for _, match := range matches {
		secretKey := match[1]
		secretValue, err := sr.secretStore.Get(secretKey)
		if err != nil {
			return "", fmt.Errorf("failed to get secret %s: %w", secretKey, err)
		}
		secrets[secretKey] = secretValue
	}

	// Try local models first
	if len(sr.config.LocalModels) > 0 {
		for _, model := range sr.config.LocalModels {
			replaced, err := sr.replaceWithOllama(ctx, text, secrets, model)
			if err == nil {
				return replaced, nil
			}
			// Log error but continue to next model
			fmt.Printf("Warning: Failed to use local model %s: %v\n", model, err)
		}
	}

	// If local_only is true and all local models failed, return error
	if sr.config.LocalOnly {
		return "", fmt.Errorf("no local models available for intelligent secret replacement")
	}

	// Fallback: Use remote model (implementation depends on APS integration)
	// For now, do simple string replacement
	result := text
	for secretKey, secretValue := range secrets {
		placeholder := fmt.Sprintf("${SECRET:%s}", secretKey)
		result = strings.ReplaceAll(result, placeholder, secretValue)
	}

	return result, nil
}

// replaceWithOllama uses Ollama to intelligently replace secrets
func (sr *SecretReplacer) replaceWithOllama(ctx context.Context, text string, secrets map[string]string, model string) (string, error) {
	// Build prompt
	prompt := sr.buildReplacementPrompt(text, secrets)

	// Call Ollama API
	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "curl", "-s", "http://localhost:11434/api/generate",
		"-H", "Content-Type: application/json",
		"-d", string(payloadBytes))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}

	// Parse response
	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return strings.TrimSpace(response.Response), nil
}

// buildReplacementPrompt creates a prompt for the LLM to replace secrets
func (sr *SecretReplacer) buildReplacementPrompt(text string, secrets map[string]string) string {
	var sb strings.Builder

	sb.WriteString("Replace the secret placeholders in the following text with their actual values.\n\n")
	sb.WriteString("Available secrets:\n")
	for key := range secrets {
		sb.WriteString(fmt.Sprintf("- ${SECRET:%s}\n", key))
	}
	sb.WriteString("\nText to process:\n")
	sb.WriteString(text)
	sb.WriteString("\n\nReturn ONLY the processed text with placeholders replaced. Do not include explanations.")

	return sb.String()
}

// isOnlyPlaceholder checks if string is exactly one placeholder
func (sr *SecretReplacer) isOnlyPlaceholder(s string) bool {
	return sr.pattern.MatchString(s) && s == sr.pattern.FindString(s)
}

// deepCopyMap creates a deep copy of a map
func (sr *SecretReplacer) deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = sr.deepCopyMap(val)
		case []interface{}:
			result[k] = sr.deepCopySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

// deepCopySlice creates a deep copy of a slice
func (sr *SecretReplacer) deepCopySlice(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = sr.deepCopyMap(val)
		case []interface{}:
			result[i] = sr.deepCopySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}
