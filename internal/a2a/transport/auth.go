package transport

import (
	"context"
	"fmt"
	"net/http"

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

// getAPIKeyFromProfile retrieves API key from profile secrets via the
// configured kit/storage/secret backend.
func getAPIKeyFromProfile(profile *core.Profile) (string, error) {
	secrets, err := core.LoadProfileSecrets(profile.ID)
	if err != nil {
		return "", fmt.Errorf("failed to read secrets: %w", err)
	}
	apiKey := secrets["A2A_API_KEY"]
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in secrets")
	}
	return apiKey, nil
}

// getMTLSPathsFromProfile retrieves mTLS cert and key paths from profile
// secrets via the configured kit/storage/secret backend.
func getMTLSPathsFromProfile(profile *core.Profile) (string, string, error) {
	secrets, err := core.LoadProfileSecrets(profile.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to read secrets: %w", err)
	}
	certPath := secrets["A2A_MTLS_CERT"]
	keyPath := secrets["A2A_MTLS_KEY"]
	if certPath == "" || keyPath == "" {
		return "", "", fmt.Errorf("mTLS paths not found in secrets")
	}
	return certPath, keyPath, nil
}

