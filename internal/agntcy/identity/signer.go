package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// GenerateKeyPair generates an Ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}
	return pub, priv, nil
}

// SaveKeyPair writes the private key to keyPath and public key to keyPath.pub.
func SaveKeyPair(keyPath string, pub ed25519.PublicKey, priv ed25519.PrivateKey) error {
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	if err := os.WriteFile(keyPath, priv.Seed(), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	pubPath := keyPath + ".pub"
	if err := os.WriteFile(pubPath, pub, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// LoadPrivateKey reads an Ed25519 private key from disk.
func LoadPrivateKey(keyPath string) (ed25519.PrivateKey, error) {
	seed, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid private key size: expected %d bytes, got %d", ed25519.SeedSize, len(seed))
	}

	return ed25519.NewKeyFromSeed(seed), nil
}

// LoadPublicKey reads an Ed25519 public key from disk.
func LoadPublicKey(keyPath string) (ed25519.PublicKey, error) {
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d bytes, got %d", ed25519.PublicKeySize, len(pub))
	}

	return ed25519.PublicKey(pub), nil
}

// SignMessage signs a message with the private key at keyPath.
func SignMessage(keyPath string, message []byte) ([]byte, error) {
	priv, err := LoadPrivateKey(keyPath)
	if err != nil {
		return nil, err
	}

	return ed25519.Sign(priv, message), nil
}

// VerifySignature verifies an Ed25519 signature.
func VerifySignature(pubKey ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pubKey, message, sig)
}
