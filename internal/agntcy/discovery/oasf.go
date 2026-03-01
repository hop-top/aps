package discovery

import (
	"fmt"

	"hop.top/aps/internal/core"
)

// GenerateOASFRecord generates an OASF-compatible record from an APS profile.
// The record is a map suitable for JSON serialization and Directory registration.
func GenerateOASFRecord(profile *core.Profile) (map[string]interface{}, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}

	if profile.ID == "" {
		return nil, fmt.Errorf("profile ID is required")
	}

	record := map[string]interface{}{
		"schema_version": "1.0",
		"type":           "agent",
		"name":           profile.DisplayName,
		"id":             profile.ID,
	}

	// Capabilities
	if len(profile.Capabilities) > 0 {
		record["capabilities"] = profile.Capabilities
	}

	// Transport endpoints
	endpoints := map[string]interface{}{}
	if profile.A2A != nil {
		a2aEndpoint := map[string]interface{}{
			"protocol": profile.A2A.ProtocolBinding,
		}
		if profile.A2A.PublicEndpoint != "" {
			a2aEndpoint["url"] = profile.A2A.PublicEndpoint
		} else if profile.A2A.ListenAddr != "" {
			a2aEndpoint["url"] = fmt.Sprintf("http://%s", profile.A2A.ListenAddr)
		}
		endpoints["a2a"] = a2aEndpoint
	}
	if profile.ACP != nil && profile.ACP.Enabled {
		endpoints["acp"] = map[string]interface{}{
			"transport": profile.ACP.Transport,
		}
	}
	if len(endpoints) > 0 {
		record["endpoints"] = endpoints
	}

	// Identity (DID) — populated when identity is configured
	if profile.Identity != nil && profile.Identity.DID != "" {
		record["identity"] = map[string]interface{}{
			"did": profile.Identity.DID,
		}
	}

	return record, nil
}

// ValidateOASFRecord checks that a generated OASF record has required fields.
func ValidateOASFRecord(record map[string]interface{}) error {
	required := []string{"schema_version", "type", "name", "id"}
	for _, field := range required {
		if _, ok := record[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	if t, ok := record["type"].(string); !ok || t != "agent" {
		return fmt.Errorf("type must be 'agent'")
	}

	return nil
}
