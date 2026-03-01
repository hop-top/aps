package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
)

// DIDDocument represents a minimal DID Document.
type DIDDocument struct {
	ID                 string              `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
}

// VerificationMethod represents a DID verification method.
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
}

// multicodecEd25519 is the multicodec prefix for Ed25519 public keys.
var multicodecEd25519 = []byte{0xed, 0x01}

// GenerateDID generates a DID and saves the key pair for a profile.
// Supported methods: "did:key" (default), "did:web".
func GenerateDID(method, profileID string) (string, string, error) {
	if profileID == "" {
		return "", "", fmt.Errorf("profile ID is required")
	}

	pub, priv, err := GenerateKeyPair()
	if err != nil {
		return "", "", err
	}

	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get profile directory: %w", err)
	}

	keyPath := profileDir + "/identity.key"
	if err := SaveKeyPair(keyPath, pub, priv); err != nil {
		return "", "", err
	}

	var did string
	switch method {
	case "did:key", "":
		did = encodeDIDKey(pub)
	case "did:web":
		did = fmt.Sprintf("did:web:localhost:agents:%s", profileID)
	default:
		return "", "", fmt.Errorf("unsupported DID method: %s (use: did:key, did:web)", method)
	}

	return did, keyPath, nil
}

// encodeDIDKey encodes an Ed25519 public key as a did:key identifier.
func encodeDIDKey(pub ed25519.PublicKey) string {
	// Prepend multicodec prefix for Ed25519
	mcKey := append(multicodecEd25519, pub...)
	// Encode with base58btc multibase prefix 'z'
	encoded := "z" + base58Encode(mcKey)
	return "did:key:" + encoded
}

// ResolveDID resolves a DID to its DID Document.
// Currently supports did:key resolution (self-describing).
func ResolveDID(did string) (*DIDDocument, error) {
	if did == "" {
		return nil, fmt.Errorf("DID is required")
	}

	if strings.HasPrefix(did, "did:key:") {
		return resolveDIDKey(did)
	}

	if strings.HasPrefix(did, "did:web:") {
		return &DIDDocument{
			ID: did,
		}, nil
	}

	return nil, fmt.Errorf("unsupported DID method: %s", did)
}

// resolveDIDKey resolves a did:key to extract the public key.
func resolveDIDKey(did string) (*DIDDocument, error) {
	// Extract the multibase-encoded key
	parts := strings.SplitN(did, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid did:key format")
	}

	mbKey := parts[2]
	if len(mbKey) == 0 || mbKey[0] != 'z' {
		return nil, fmt.Errorf("expected base58btc multibase prefix 'z'")
	}

	decoded, err := base58Decode(mbKey[1:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	// Verify multicodec prefix
	if len(decoded) < 2 || decoded[0] != 0xed || decoded[1] != 0x01 {
		return nil, fmt.Errorf("not an Ed25519 key (wrong multicodec prefix)")
	}

	pubKeyBytes := decoded[2:]
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
	}

	pubKeyMultibase := "z" + base58Encode(append(multicodecEd25519, pubKeyBytes...))

	return &DIDDocument{
		ID: did,
		VerificationMethod: []VerificationMethod{
			{
				ID:                 did + "#keys-1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         did,
				PublicKeyMultibase: pubKeyMultibase,
			},
		},
	}, nil
}

// ExtractPublicKeyFromDID extracts the Ed25519 public key from a did:key.
func ExtractPublicKeyFromDID(did string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(did, "did:key:") {
		return nil, fmt.Errorf("can only extract public key from did:key, got: %s", did)
	}

	parts := strings.SplitN(did, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid did:key format")
	}

	mbKey := parts[2]
	if len(mbKey) == 0 || mbKey[0] != 'z' {
		return nil, fmt.Errorf("expected base58btc multibase prefix 'z'")
	}

	decoded, err := base58Decode(mbKey[1:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	if len(decoded) < 2 || decoded[0] != 0xed || decoded[1] != 0x01 {
		return nil, fmt.Errorf("not an Ed25519 key")
	}

	pubKeyBytes := decoded[2:]
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
	}

	return ed25519.PublicKey(pubKeyBytes), nil
}

// LoadDID loads the DID from a profile's identity config.
func LoadDID(profile *core.Profile) (string, error) {
	if profile.Identity == nil || profile.Identity.DID == "" {
		return "", fmt.Errorf("no DID configured for profile %s", profile.ID)
	}
	return profile.Identity.DID, nil
}

// base58Encode encodes bytes to base58btc (Bitcoin alphabet).
func base58Encode(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Count leading zeros
	zeros := 0
	for _, b := range input {
		if b != 0 {
			break
		}
		zeros++
	}

	// Convert to base58
	size := len(input)*138/100 + 1
	buf := make([]byte, size)
	high := size - 1

	for _, b := range input {
		carry := int(b)
		j := size - 1
		for ; j > high || carry != 0; j-- {
			carry += 256 * int(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}
		high = j
	}

	// Skip leading zeros in buf
	start := 0
	for start < size && buf[start] == 0 {
		start++
	}

	// Build result
	result := make([]byte, zeros+size-start)
	for i := 0; i < zeros; i++ {
		result[i] = alphabet[0]
	}
	for i := start; i < size; i++ {
		result[zeros+i-start] = alphabet[buf[i]]
	}

	return string(result)
}

// base58Decode decodes a base58btc string.
func base58Decode(input string) ([]byte, error) {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Build reverse lookup
	lookup := [256]int{}
	for i := range lookup {
		lookup[i] = -1
	}
	for i, c := range alphabet {
		lookup[c] = i
	}

	// Count leading '1's (zeros)
	zeros := 0
	for _, c := range input {
		if c != '1' {
			break
		}
		zeros++
	}

	// Allocate enough space
	size := len(input)*733/1000 + 1
	buf := make([]byte, size)

	for _, c := range input {
		val := lookup[c]
		if val == -1 {
			return nil, fmt.Errorf("invalid base58 character: %c", c)
		}

		carry := val
		for j := size - 1; j >= 0; j-- {
			carry += 58 * int(buf[j])
			buf[j] = byte(carry % 256)
			carry /= 256
		}
	}

	// Skip leading zeros in buf
	start := 0
	for start < size && buf[start] == 0 {
		start++
	}

	result := make([]byte, zeros+size-start)
	copy(result[zeros:], buf[start:])

	return result, nil
}

// Ensure base64 is available for badge operations (used in badge.go).
var _ = base64.StdEncoding
