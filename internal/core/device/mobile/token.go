package mobile

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	DefaultTokenExpiry = 14 * 24 * time.Hour // 14 days
	DefaultQRExpiry    = 15 * time.Minute
	KeySize            = 2048
)

// DeviceClaims holds the JWT claims for a mobile device token
type DeviceClaims struct {
	DeviceID     string   `json:"device_id"`
	DeviceName   string   `json:"device_name"`
	DeviceOS     string   `json:"device_os"`
	ProfileID    string   `json:"profile_id"`
	Capabilities []string `json:"capabilities"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT token creation, validation, and revocation
type TokenManager struct {
	profileID  string
	keyDir     string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewTokenManager creates a new token manager for a profile
func NewTokenManager(profileID, keyDir string) (*TokenManager, error) {
	tm := &TokenManager{
		profileID: profileID,
		keyDir:    keyDir,
	}

	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	if err := tm.loadOrGenerateKeys(); err != nil {
		return nil, err
	}

	return tm, nil
}

// CreateToken generates a JWT token for a mobile device
func (tm *TokenManager) CreateToken(device *MobileDevice, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = DefaultTokenExpiry
	}

	now := time.Now()
	claims := &DeviceClaims{
		DeviceID:     device.DeviceID,
		DeviceName:   device.DeviceName,
		DeviceOS:     device.DeviceOS,
		ProfileID:    tm.profileID,
		Capabilities: device.Capabilities,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "aps",
			Subject:   device.DeviceID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			ID:        device.DeviceID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tm.profileID

	tokenString, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (tm *TokenManager) ValidateToken(tokenString string) (*DeviceClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &DeviceClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.publicKey, nil
	})
	if err != nil {
		return nil, ErrTokenInvalid(err)
	}

	claims, ok := token.Claims.(*DeviceClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid(fmt.Errorf("invalid claims"))
	}

	return claims, nil
}

// HashToken returns a SHA256 hash of a token string for storage
func HashToken(tokenString string) string {
	h := sha256.Sum256([]byte(tokenString))
	return "sha256:" + hex.EncodeToString(h[:])
}

// loadOrGenerateKeys loads existing RSA keys or generates new ones
func (tm *TokenManager) loadOrGenerateKeys() error {
	privPath := filepath.Join(tm.keyDir, "device.key")
	pubPath := filepath.Join(tm.keyDir, "device.pub")

	if _, err := os.Stat(privPath); err == nil {
		return tm.loadKeys(privPath, pubPath)
	}

	return tm.generateKeys(privPath, pubPath)
}

func (tm *TokenManager) loadKeys(privPath, pubPath string) error {
	privData, err := os.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(privData)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	tm.privateKey = privateKey
	tm.publicKey = &privateKey.PublicKey
	return nil
}

func (tm *TokenManager) generateKeys(privPath, pubPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubBytes,
	})
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	tm.privateKey = privateKey
	tm.publicKey = &privateKey.PublicKey
	return nil
}

// CertFingerprint returns the SHA256 fingerprint of the public key for QR embedding
func (tm *TokenManager) CertFingerprint() (string, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(tm.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	h := sha256.Sum256(pubBytes)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
