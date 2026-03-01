package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core"
)

// Badge represents a Verifiable Credential (Agent Badge).
type Badge struct {
	Context      string    `json:"@context"`
	Type         string    `json:"type"`
	Issuer       string    `json:"issuer"`
	IssuanceDate string    `json:"issuanceDate"`
	Subject      Subject   `json:"credentialSubject"`
	Proof        *Proof    `json:"proof,omitempty"`
}

// Subject represents the badge credential subject.
type Subject struct {
	ID         string `json:"id"`
	Capability string `json:"capability"`
}

// Proof represents the Ed25519 signature proof.
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	VerificationMethod string `json:"verificationMethod"`
	ProofValue         string `json:"proofValue"`
}

// BadgeVerification holds the result of badge verification.
type BadgeVerification struct {
	Valid      bool   `json:"valid"`
	Issuer     string `json:"issuer"`
	Capability string `json:"capability"`
	Subject    string `json:"subject"`
	Error      string `json:"error,omitempty"`
}

// IssueBadge creates a signed Verifiable Credential for a profile capability.
func IssueBadge(profile *core.Profile, capability string) (*Badge, error) {
	if profile.Identity == nil {
		return nil, fmt.Errorf("identity not configured for profile %s", profile.ID)
	}

	if profile.Identity.DID == "" {
		return nil, fmt.Errorf("no DID configured for profile %s", profile.ID)
	}

	if profile.Identity.KeyPath == "" {
		return nil, fmt.Errorf("no key path configured for profile %s", profile.ID)
	}

	now := time.Now().UTC()

	badge := &Badge{
		Context:      "https://www.w3.org/2018/credentials/v1",
		Type:         "VerifiableCredential",
		Issuer:       profile.Identity.DID,
		IssuanceDate: now.Format(time.RFC3339),
		Subject: Subject{
			ID:         profile.Identity.DID,
			Capability: capability,
		},
	}

	// Sign the badge
	payload, err := json.Marshal(map[string]interface{}{
		"issuer":       badge.Issuer,
		"issuanceDate": badge.IssuanceDate,
		"subject":      badge.Subject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal badge payload: %w", err)
	}

	sig, err := SignMessage(profile.Identity.KeyPath, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to sign badge: %w", err)
	}

	badge.Proof = &Proof{
		Type:               "Ed25519Signature2020",
		Created:            now.Format(time.RFC3339),
		VerificationMethod: profile.Identity.DID + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString(sig),
	}

	return badge, nil
}

// SaveBadge writes a badge to a JSON file.
func SaveBadge(badge *Badge, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create badge directory: %w", err)
	}

	data, err := json.MarshalIndent(badge, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal badge: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// VerifyBadge loads and verifies a badge from a file.
func VerifyBadge(badgePath string) (*BadgeVerification, error) {
	data, err := os.ReadFile(badgePath)
	if err != nil {
		return &BadgeVerification{Valid: false, Error: fmt.Sprintf("failed to read badge: %v", err)}, nil
	}

	var badge Badge
	if err := json.Unmarshal(data, &badge); err != nil {
		return &BadgeVerification{Valid: false, Error: fmt.Sprintf("failed to parse badge: %v", err)}, nil
	}

	if badge.Proof == nil {
		return &BadgeVerification{
			Valid:      false,
			Issuer:     badge.Issuer,
			Capability: badge.Subject.Capability,
			Subject:    badge.Subject.ID,
			Error:      "badge has no proof",
		}, nil
	}

	// Extract public key from issuer DID
	pubKey, err := ExtractPublicKeyFromDID(badge.Issuer)
	if err != nil {
		return &BadgeVerification{
			Valid:      false,
			Issuer:     badge.Issuer,
			Capability: badge.Subject.Capability,
			Subject:    badge.Subject.ID,
			Error:      fmt.Sprintf("failed to extract public key from issuer DID: %v", err),
		}, nil
	}

	// Reconstruct signed payload
	payload, err := json.Marshal(map[string]interface{}{
		"issuer":       badge.Issuer,
		"issuanceDate": badge.IssuanceDate,
		"subject":      badge.Subject,
	})
	if err != nil {
		return &BadgeVerification{
			Valid:      false,
			Issuer:     badge.Issuer,
			Capability: badge.Subject.Capability,
			Subject:    badge.Subject.ID,
			Error:      fmt.Sprintf("failed to reconstruct payload: %v", err),
		}, nil
	}

	// Decode and verify signature
	sig, err := base64.StdEncoding.DecodeString(badge.Proof.ProofValue)
	if err != nil {
		return &BadgeVerification{
			Valid:      false,
			Issuer:     badge.Issuer,
			Capability: badge.Subject.Capability,
			Subject:    badge.Subject.ID,
			Error:      fmt.Sprintf("failed to decode signature: %v", err),
		}, nil
	}

	valid := ed25519.Verify(pubKey, payload, sig)

	result := &BadgeVerification{
		Valid:      valid,
		Issuer:     badge.Issuer,
		Capability: badge.Subject.Capability,
		Subject:    badge.Subject.ID,
	}

	if !valid {
		result.Error = "signature verification failed"
	}

	return result, nil
}
