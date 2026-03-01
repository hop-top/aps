package trust

import (
	"fmt"

	"hop.top/aps/internal/core"
)

// LoadTrustPolicy loads the trust config from a profile.
func LoadTrustPolicy(profile *core.Profile) *core.TrustConfig {
	if profile == nil || profile.Trust == nil {
		return nil
	}
	return profile.Trust
}

// ValidateTrustConfig validates a trust configuration.
func ValidateTrustConfig(cfg *core.TrustConfig) error {
	if cfg == nil {
		return fmt.Errorf("trust config is nil")
	}

	// If allowed issuers are specified without require_identity, warn but allow
	// The verifier will enforce require_identity at runtime

	return nil
}
