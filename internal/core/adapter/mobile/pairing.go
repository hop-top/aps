package mobile

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Pairing code alphabet — excludes ambiguous characters (0/O, 1/l/I)
const pairingAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// GeneratePairingCode creates a grouped pairing code like "ABC-123-XYZ"
func GeneratePairingCode(groupSize, numGroups int) (string, error) {
	groups := make([]string, numGroups)
	for g := 0; g < numGroups; g++ {
		chars := make([]byte, groupSize)
		for i := 0; i < groupSize; i++ {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(pairingAlphabet))))
			if err != nil {
				return "", fmt.Errorf("failed to generate pairing code: %w", err)
			}
			chars[i] = pairingAlphabet[idx.Int64()]
		}
		groups[g] = string(chars)
	}
	return strings.Join(groups, "-"), nil
}

// DefaultPairingCode generates a 3-group, 3-char code: "ABC-123-XYZ"
func DefaultPairingCode() (string, error) {
	return GeneratePairingCode(3, 3)
}

// EncodePairingPayload encodes a QR payload to base64 JSON
func EncodePairingPayload(payload *QRPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal QR payload: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodePairingPayload decodes a base64 JSON QR payload
func DecodePairingPayload(encoded string) (*QRPayload, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode QR payload: %w", err)
	}
	var payload QRPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal QR payload: %w", err)
	}
	return &payload, nil
}

// NewQRPayload creates a QR payload for device pairing
func NewQRPayload(profileID, endpoint, pairingCode, certFingerprint string, capabilities []string, qrExpiry time.Duration) *QRPayload {
	return &QRPayload{
		Version:         "1.0",
		ProfileID:       profileID,
		Endpoint:        endpoint,
		PairingCode:     pairingCode,
		ExpiresAt:       time.Now().Add(qrExpiry).Format(time.RFC3339),
		CertFingerprint: certFingerprint,
		Capabilities:    capabilities,
	}
}

// IsPayloadExpired checks if the QR payload has expired
func IsPayloadExpired(payload *QRPayload) bool {
	t, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().After(t)
}
