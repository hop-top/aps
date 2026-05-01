// Package core secrets bridge.
//
// Profile secrets are routed through hop.top/kit/go/storage/secret so the
// backend is configurable via Config.Secrets.Backend (file, env, keyring;
// future: openbao, agefile, onepassword). The default "file" backend
// preserves the legacy per-profile secrets.env layout (godotenv format).
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"hop.top/kit/go/storage/secret"
	_ "hop.top/kit/go/storage/secret/env"     // register "env"
	_ "hop.top/kit/go/storage/secret/keyring" // register "keyring"
)

// SecretsBackend* are the canonical backend identifiers for SecretsConfig.
const (
	SecretsBackendFile    = "file"
	SecretsBackendEnv     = "env"
	SecretsBackendKeyring = "keyring"
)

// LoadSecrets loads secrets for the file backend from path (a secrets.env
// file). Kept for backward compatibility with execution.go and isolation
// handlers that pass an explicit secrets.env path. Returns nil for missing
// files. Warns on insecure permissions.
func LoadSecrets(path string) (map[string]string, error) {
	info, err := os.Stat(path) // #nosec G304 -- path comes from per-profile dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if mode := info.Mode().Perm(); mode&0077 != 0 {
		fmt.Fprintf(os.Stderr,
			"WARNING: Secrets file %s has insecure permissions (%o). It should be 0600.\n",
			path, mode)
	}
	return godotenv.Read(path)
}

// LoadProfileSecrets returns the env-style key→value map for a profile,
// honoring Config.Secrets.Backend. The file backend reads secrets.env from
// the per-profile dir; env reads APS_SECRET_<KEY> from the environment;
// keyring lists keys under Service (default "aps/<profileID>").
func LoadProfileSecrets(profileID string) (map[string]string, error) {
	cfg, _ := LoadConfig()
	switch cfg.Secrets.Backend {
	case "", SecretsBackendFile:
		dir, err := GetProfileDir(profileID)
		if err != nil {
			return nil, err
		}
		return LoadSecrets(filepath.Join(dir, "secrets.env"))
	case SecretsBackendEnv, SecretsBackendKeyring:
		store, err := openProfileStore(cfg, profileID)
		if err != nil {
			return nil, err
		}
		return drainStore(context.Background(), store)
	default:
		return nil, fmt.Errorf("unknown secrets backend %q", cfg.Secrets.Backend)
	}
}

// OpenProfileSecretStore opens a kit/storage/secret store for the given
// profile using the configured backend. Callers may use Get/Set/Delete to
// manage individual secrets. Returns ErrNotSupported semantics from the
// underlying backend.
func OpenProfileSecretStore(profileID string) (secret.MutableStore, error) {
	cfg, _ := LoadConfig()
	return openProfileStore(cfg, profileID)
}

func openProfileStore(cfg *Config, profileID string) (secret.MutableStore, error) {
	backend := cfg.Secrets.Backend
	if backend == "" {
		backend = SecretsBackendFile
	}
	switch backend {
	case SecretsBackendFile:
		dir, err := GetProfileDir(profileID)
		if err != nil {
			return nil, err
		}
		// Returns a wrapper around secrets.env (godotenv format).
		return newDotenvStore(filepath.Join(dir, "secrets.env")), nil
	case SecretsBackendEnv:
		prefix := cfg.Secrets.Prefix
		if prefix == "" {
			prefix = "APS_SECRET_"
		}
		return secret.Open(secret.Config{Backend: SecretsBackendEnv, Prefix: prefix})
	case SecretsBackendKeyring:
		svc := cfg.Secrets.Service
		if svc == "" {
			svc = "aps/" + profileID
		}
		return secret.Open(secret.Config{Backend: SecretsBackendKeyring, Service: svc})
	default:
		return nil, fmt.Errorf("unknown secrets backend %q", backend)
	}
}

func drainStore(ctx context.Context, store secret.Store) (map[string]string, error) {
	keys, err := store.List(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		s, err := store.Get(ctx, k)
		if err != nil {
			return nil, err
		}
		out[k] = string(s.Value)
	}
	return out, nil
}

// dotenvStore implements secret.MutableStore over a single secrets.env file.
// It preserves the legacy aps file layout while exposing the kit Store API.
type dotenvStore struct{ path string }

func newDotenvStore(path string) *dotenvStore { return &dotenvStore{path: path} }

func (d *dotenvStore) read() (map[string]string, error) {
	m, err := LoadSecrets(d.path)
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

func (d *dotenvStore) write(m map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(d.path), 0700); err != nil {
		return err
	}
	var b strings.Builder
	for k, v := range m {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return os.WriteFile(d.path, []byte(b.String()), 0600)
}

func (d *dotenvStore) Get(_ context.Context, key string) (*secret.Secret, error) {
	m, err := d.read()
	if err != nil {
		return nil, err
	}
	v, ok := m[key]
	if !ok {
		return nil, secret.ErrNotFound
	}
	return &secret.Secret{Key: key, Value: []byte(v)}, nil
}

func (d *dotenvStore) List(_ context.Context, prefix string) ([]string, error) {
	m, err := d.read()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (d *dotenvStore) Exists(_ context.Context, key string) (bool, error) {
	m, err := d.read()
	if err != nil {
		return false, err
	}
	_, ok := m[key]
	return ok, nil
}

func (d *dotenvStore) Set(_ context.Context, key string, value []byte) error {
	m, err := d.read()
	if err != nil {
		return err
	}
	m[key] = string(value)
	return d.write(m)
}

func (d *dotenvStore) Delete(_ context.Context, key string) error {
	m, err := d.read()
	if err != nil {
		return err
	}
	if _, ok := m[key]; !ok {
		return secret.ErrNotFound
	}
	delete(m, key)
	return d.write(m)
}
