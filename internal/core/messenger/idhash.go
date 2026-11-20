package messenger

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/fsperm"
)

// Keyed ID hashes.
//
// IDs of phone-number platforms (WhatsApp, SMS) and email are PII, so
// persisted records keep them redacted. To keep records joinable and
// searchable, a sibling "<key>_hmac" field carries a stable keyed hash:
// hex of the first 16 bytes (128 bits) of HMAC-SHA256(installationKey,
// value). A keyed hash is required because phone numbers are
// low-entropy: a bare SHA-256 can be brute-forced. 128 bits is ample
// for collision resistance on lookup and keeps records short.
//
// The installation key is 32 random bytes in
// <aps data dir>/messenger-id-hmac.key (0600). When it cannot be read
// or created, no hash is written and the ID stays redacted (fail closed).

const (
	idKeyFile  = "messenger-id-hmac.key"
	idKeyBytes = 32
	idHMACSize = 16
)

// idKeyPath resolves the key file location; a variable so tests can
// point it at a temp dir.
var idKeyPath = func() (string, error) {
	dir, err := core.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("aps data dir: %w", err)
	}
	return filepath.Join(dir, idKeyFile), nil
}

var (
	idKeyMu    sync.Mutex
	idKeyCache = map[string][]byte{} // path -> key; failures are not cached
)

// loadIDKey returns the installation key at path, creating it on first
// use. Creation is race-safe across goroutines and processes: the key
// is written to a unique temp file and hard-linked into place, which
// fails if another creator won; the loser reads the winner's key.
func loadIDKey(path string) ([]byte, error) {
	idKeyMu.Lock()
	defer idKeyMu.Unlock()
	if k, ok := idKeyCache[path]; ok {
		return k, nil
	}
	k, err := readIDKey(path)
	if errors.Is(err, os.ErrNotExist) {
		k, err = createIDKey(path)
	}
	if err != nil {
		return nil, err
	}
	idKeyCache[path] = k
	return k, nil
}

func readIDKey(path string) ([]byte, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- fixed installation key path
	if err != nil {
		return nil, fmt.Errorf("read id hmac key: %w", err)
	}
	if len(b) != idKeyBytes {
		return nil, fmt.Errorf("id hmac key %s: want %d bytes, got %d", path, idKeyBytes, len(b))
	}
	return b, nil
}

func createIDKey(path string) ([]byte, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create key dir: %w", err)
	}
	key := make([]byte, idKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	tmp, err := os.CreateTemp(dir, idKeyFile+".*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create key temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := fsperm.WriteSecret(tmpPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}
	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return readIDKey(path) // lost the race; use the winner's key
		}
		return nil, fmt.Errorf("install key: %w", err)
	}
	return key, nil
}

// idHMAC returns the keyed hash of v, or "" when v is empty or the key
// is unavailable.
func idHMAC(v string) string {
	if v == "" {
		return ""
	}
	path, err := idKeyPath()
	if err != nil {
		return ""
	}
	key, err := loadIDKey(path)
	if err != nil {
		return ""
	}
	m := hmac.New(sha256.New, key)
	m.Write([]byte(v))
	return hex.EncodeToString(m.Sum(nil)[:idHMACSize])
}

// withIDHashes adds "<key>_hmac" siblings for every ID field of entry
// whose platform keeps IDs redacted. Verbatim platforms need no hash.
func withIDHashes(platform string, entry map[string]any) {
	if verbatimIDPlatforms[MessengerPlatform(platform)] {
		return
	}
	for _, k := range idKeys {
		if s, ok := entry[k].(string); ok {
			if h := idHMAC(s); h != "" {
				entry[k+"_hmac"] = h
			}
		}
	}
}
