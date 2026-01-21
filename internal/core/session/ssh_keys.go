package session

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"oss-aps-cli/internal/core"
)

const (
	KeysDir      = "keys"
	AdminKeyFile = "admin_key"
)

type SSHKeyType string

const (
	SSHKeyRSA     SSHKeyType = "rsa"
	SSHKeyEd25519 SSHKeyType = "ed25519"
)

type AdminSSHKey struct {
	KeyType    SSHKeyType
	PublicKey  string
	PrivateKey []byte
}

type SSHKeyManager struct{}

func NewSSHKeyManager() *SSHKeyManager {
	return &SSHKeyManager{}
}

func (m *SSHKeyManager) GenerateKeyPair(keyType SSHKeyType) (*AdminSSHKey, error) {
	switch keyType {
	case SSHKeyRSA:
		return m.generateRSAKey()
	case SSHKeyEd25519:
		return m.generateEd25519Key()
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

func (m *SSHKeyManager) generateRSAKey() (*AdminSSHKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rsa key: %w", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rsa public key: %w", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)

	rsaPrivateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	rsaPublicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	return &AdminSSHKey{
		KeyType:    SSHKeyRSA,
		PublicKey:  rsaPublicKeyPEM,
		PrivateKey: []byte(rsaPrivateKeyPEM),
	}, nil
}

func (m *SSHKeyManager) generateEd25519Key() (*AdminSSHKey, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ed25519 public key: %w", err)
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ed25519 private key: %w", err)
	}

	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	publicKeyPEM := string(publicKeyBytes)

	return &AdminSSHKey{
		KeyType:    SSHKeyEd25519,
		PublicKey:  publicKeyPEM,
		PrivateKey: []byte(privateKeyPEM),
	}, nil
}

func (m *SSHKeyManager) InstallAdminKey(sessionID string, key *AdminSSHKey) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	apsDir := filepath.Join(home, core.ApsHomeDir)
	keysDir := filepath.Join(apsDir, KeysDir)

	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	sessionKeyDir := filepath.Join(keysDir, sessionID)
	if err := os.MkdirAll(sessionKeyDir, 0700); err != nil {
		return fmt.Errorf("failed to create session key directory: %w", err)
	}

	privateKeyPath := filepath.Join(sessionKeyDir, AdminKeyFile)
	if err := os.WriteFile(privateKeyPath, key.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	publicKeyPath := filepath.Join(sessionKeyDir, AdminKeyFile+".pub")
	if err := os.WriteFile(publicKeyPath, []byte(key.PublicKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

func (m *SSHKeyManager) GetAdminKey(sessionID string) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	privateKeyPath := filepath.Join(home, core.ApsHomeDir, KeysDir, sessionID, AdminKeyFile)

	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read admin key: %w", err)
	}

	return privateKey, nil
}

func (m *SSHKeyManager) GetAdminPublicKey(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	publicKeyPath := filepath.Join(home, core.ApsHomeDir, KeysDir, sessionID, AdminKeyFile+".pub")

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	return string(publicKey), nil
}

func (m *SSHKeyManager) ListAdminSessions() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	keysDir := filepath.Join(home, core.ApsHomeDir, KeysDir)

	entries, err := os.ReadDir(keysDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read keys directory: %w", err)
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			sessions = append(sessions, entry.Name())
		}
	}

	return sessions, nil
}

func (m *SSHKeyManager) RemoveAdminKey(sessionID string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionKeyDir := filepath.Join(home, core.ApsHomeDir, KeysDir, sessionID)

	if err := os.RemoveAll(sessionKeyDir); err != nil {
		return fmt.Errorf("failed to remove admin key directory: %w", err)
	}

	return nil
}
