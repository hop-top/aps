package transport

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"hop.top/aps/internal/core"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthAPIKey AuthType = "apikey"
	AuthMTLS   AuthType = "mtls"
	AuthOpenID AuthType = "openid"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type     AuthType
	APIKey   string
	CertPath string
	KeyPath  string
	Token    string
}

// DefaultAuthConfig returns default authentication config
func DefaultAuthConfig(securityScheme string) *AuthConfig {
	switch securityScheme {
	case "apikey":
		return &AuthConfig{
			Type: AuthAPIKey,
		}
	case "mtls":
		return &AuthConfig{
			Type: AuthMTLS,
		}
	case "openid":
		return &AuthConfig{
			Type: AuthOpenID,
		}
	default:
		return &AuthConfig{
			Type: AuthNone,
		}
	}
}

// GetAuthFromProfile extracts auth config from profile
func GetAuthFromProfile(profile *core.Profile) (*AuthConfig, error) {
	if profile.A2A == nil {
		return &AuthConfig{Type: AuthNone}, nil
	}

	securityScheme := profile.A2A.SecurityScheme
	if securityScheme == "" {
		securityScheme = "apikey"
	}

	config := DefaultAuthConfig(securityScheme)

	if config.Type == AuthAPIKey {
		apiKey, err := getAPIKeyFromProfile(profile)
		if err != nil {
			return nil, err
		}
		config.APIKey = apiKey
	}

	if config.Type == AuthMTLS {
		certPath, keyPath, err := getMTLSPathsFromProfile(profile)
		if err != nil {
			return nil, err
		}
		config.CertPath = certPath
		config.KeyPath = keyPath
	}

	return config, nil
}

// ApplyAuth applies authentication to HTTP request
func (a *AuthConfig) ApplyAuth(req *http.Request, ctx context.Context) error {
	switch a.Type {
	case AuthAPIKey:
		if a.APIKey != "" {
			req = req.WithContext(ctx)
			req.Header.Set("X-API-Key", a.APIKey)
		}
	case AuthNone:
		req = req.WithContext(ctx)
	default:
		return fmt.Errorf("unsupported auth type: %s", a.Type)
	}

	return nil
}

// ValidateAuth validates authentication configuration
func (a *AuthConfig) ValidateAuth() error {
	switch a.Type {
	case AuthAPIKey:
		if a.APIKey == "" {
			return fmt.Errorf("API key is required for API key authentication")
		}
	case AuthMTLS:
		if a.CertPath == "" || a.KeyPath == "" {
			return fmt.Errorf("cert and key paths are required for mTLS authentication")
		}
	case AuthNone:
		return nil
	default:
		return fmt.Errorf("unsupported authentication type: %s", a.Type)
	}

	return nil
}

// getAPIKeyFromProfile retrieves API key from profile secrets
func getAPIKeyFromProfile(profile *core.Profile) (string, error) {
	profileDir, err := core.GetProfileDir(profile.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile directory: %w", err)
	}

	secretsPath := profileDir + "/secrets.env"

	secrets, err := os.ReadFile(secretsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secrets file: %w", err)
	}

	apiKey := extractAPIKey(string(secrets))
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in secrets")
	}

	return apiKey, nil
}

// getMTLSPathsFromProfile retrieves mTLS cert and key paths from profile secrets
func getMTLSPathsFromProfile(profile *core.Profile) (string, string, error) {
	profileDir, err := core.GetProfileDir(profile.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get profile directory: %w", err)
	}

	secretsPath := profileDir + "/secrets.env"

	secrets, err := os.ReadFile(secretsPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read secrets file: %w", err)
	}

	certPath := extractMTLSCert(string(secrets))
	keyPath := extractMTLSKey(string(secrets))

	if certPath == "" || keyPath == "" {
		return "", "", fmt.Errorf("mTLS paths not found in secrets")
	}

	return certPath, keyPath, nil
}

// extractAPIKey extracts API key from secrets file content
func extractAPIKey(content string) string {
	lines := parseSecretsFile(content)
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			if key, value, ok := parseSecretLine(line); ok && key == "A2A_API_KEY" {
				return value
			}
		}
	}
	return ""
}

// extractMTLSCert extracts mTLS certificate path from secrets
func extractMTLSCert(content string) string {
	lines := parseSecretsFile(content)
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			if key, value, ok := parseSecretLine(line); ok && key == "A2A_MTLS_CERT" {
				return value
			}
		}
	}
	return ""
}

// extractMTLSKey extracts mTLS key path from secrets
func extractMTLSKey(content string) string {
	lines := parseSecretsFile(content)
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			if key, value, ok := parseSecretLine(line); ok && key == "A2A_MTLS_KEY" {
				return value
			}
		}
	}
	return ""
}

// parseSecretsFile splits content into lines
func parseSecretsFile(content string) []string {
	var lines []string
	for _, line := range splitLines(content) {
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}

// parseSecretLine parses a KEY=VALUE line
func parseSecretLine(line string) (string, string, bool) {
	var key, value string
	var found bool

	for i, c := range line {
		if c == '=' && !found {
			key = line[:i]
			value = line[i+1:]
			found = true
		}
	}

	return key, value, found
}

// splitLines splits content into lines
func splitLines(content string) []string {
	var lines []string
	var current string

	for _, c := range content {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}

	if len(current) > 0 {
		lines = append(lines, current)
	}

	return lines
}
