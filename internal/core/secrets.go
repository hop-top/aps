// Package core secrets bridge.
//
// Profile secrets are routed through hop.top/kit/go/storage/secret so the
// backend is configurable via Config.Secrets.Backend: file, env, keyring,
// onepassword, openbao, infisical, or ghsecrets (see SecretsBackends). The
// default "file" backend preserves the legacy per-profile secrets.env layout
// (godotenv format); every other backend is opened through the kit registry.
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"hop.top/kit/go/storage/secret"
	_ "hop.top/kit/go/storage/secret/env"         // register "env"
	_ "hop.top/kit/go/storage/secret/ghsecrets"   // register "ghsecrets"
	_ "hop.top/kit/go/storage/secret/infisical"   // register "infisical"
	_ "hop.top/kit/go/storage/secret/keyring"     // register "keyring"
	_ "hop.top/kit/go/storage/secret/onepassword" // register "onepassword"
	"hop.top/kit/go/storage/secret/openbao"
)

// kit ships the openbao store but registers no opener for it, unlike every
// other backend. Register it here so it is selectable via secrets.backend.
func init() {
	secret.RegisterBackend(SecretsBackendOpenBao, func(cfg secret.Config) (secret.MutableStore, error) {
		if cfg.Addr == "" {
			return nil, fmt.Errorf("secret: openbao backend requires Addr")
		}
		if cfg.Token == "" {
			return nil, fmt.Errorf("secret: openbao backend requires Token")
		}
		return openbao.New(cfg.Addr, cfg.Token, cfg.Mount)
	})
}

// SecretsBackend* are the canonical backend identifiers for SecretsConfig.
const (
	SecretsBackendFile        = "file"
	SecretsBackendEnv         = "env"
	SecretsBackendKeyring     = "keyring"
	SecretsBackendOnePassword = "onepassword"
	SecretsBackendOpenBao     = "openbao"
	SecretsBackendInfisical   = "infisical"
	SecretsBackendGHSecrets   = "ghsecrets"
)

// SecretsBackends lists every backend selectable via Config.Secrets.Backend.
var SecretsBackends = []string{
	SecretsBackendFile,
	SecretsBackendEnv,
	SecretsBackendKeyring,
	SecretsBackendOnePassword,
	SecretsBackendOpenBao,
	SecretsBackendInfisical,
	SecretsBackendGHSecrets,
}

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
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		fmt.Fprintf(os.Stderr,
			"WARNING: Secrets file %s has insecure permissions (%o). It should be 0600.\n",
			path, mode)
	}
	return godotenv.Read(path)
}

// LoadProfileSecrets returns the env-style key→value map for a profile,
// honoring Config.Secrets.Backend. The file backend reads secrets.env from
// the per-profile dir; every other backend is opened through the kit registry
// and drained via List/Get.
func LoadProfileSecrets(profileID string) (map[string]string, error) {
	cfg, _ := LoadConfig()
	backend := cfg.Secrets.Backend
	if backend == "" {
		backend = SecretsBackendFile
	}
	// The file backend reads the legacy per-profile secrets.env directly.
	if backend == SecretsBackendFile {
		dir, err := GetProfileDir(profileID)
		if err != nil {
			return nil, err
		}
		return LoadSecrets(filepath.Join(dir, "secrets.env"))
	}
	// Every other backend drains through the kit registry, so newly
	// registered backends work here without further changes.
	store, err := openProfileStore(cfg, profileID)
	if err != nil {
		return nil, err
	}
	return drainStore(context.Background(), store)
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
	// The file backend predates the kit registry and keeps the legacy
	// per-profile secrets.env layout, so it is opened directly.
	if backend == SecretsBackendFile {
		dir, err := GetProfileDir(profileID)
		if err != nil {
			return nil, err
		}
		return newDotenvStore(filepath.Join(dir, "secrets.env")), nil
	}

	kitCfg, err := secretBackendConfig(cfg.Secrets, backend, profileID)
	if err != nil {
		return nil, err
	}
	store, err := secret.Open(kitCfg)
	if err != nil {
		return nil, fmt.Errorf("opening %q secrets backend: %w", backend, err)
	}
	return store, nil
}

// secretBackendConfig maps aps SecretsConfig onto the kit backend config,
// applying aps-specific defaults and validating the fields each backend
// requires. Backends ignore fields they do not use.
func secretBackendConfig(sc SecretsConfig, backend, profileID string) (secret.Config, error) {
	kitCfg := secret.Config{
		Backend:    backend,
		Prefix:     sc.Prefix,
		Service:    sc.Service,
		Addr:       sc.Addr,
		Token:      secretBackendToken(sc),
		Mount:      sc.Mount,
		Project:    sc.Project,
		Env:        sc.Env,
		Repo:       sc.Repo,
		Vault:      sc.Vault,
		ConnectURL: sc.ConnectURL,
	}

	switch backend {
	case SecretsBackendEnv:
		if kitCfg.Prefix == "" {
			kitCfg.Prefix = "APS_SECRET_"
		}
	case SecretsBackendKeyring:
		if kitCfg.Service == "" {
			kitCfg.Service = "aps/" + profileID
		}
	case SecretsBackendOnePassword:
		if kitCfg.Vault == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.vault", backend)
		}
	case SecretsBackendOpenBao:
		if kitCfg.Addr == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.addr", backend)
		}
		if kitCfg.Token == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.token or secrets.token_env", backend)
		}
	case SecretsBackendInfisical:
		if kitCfg.Addr == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.addr", backend)
		}
		if kitCfg.Project == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.project", backend)
		}
		if kitCfg.Token == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.token or secrets.token_env", backend)
		}
		if kitCfg.Env == "" {
			return secret.Config{}, fmt.Errorf("secrets backend %q requires secrets.env", backend)
		}
	case SecretsBackendGHSecrets:
		// Repo may be empty: the backend falls back to the current repo.
	default:
		return secret.Config{}, fmt.Errorf("unknown secrets backend %q", backend)
	}
	return kitCfg, nil
}

// secretBackendToken prefers an explicit token, then the environment variable
// named by TokenEnv, so vault credentials need not live in the config file.
func secretBackendToken(sc SecretsConfig) string {
	if strings.TrimSpace(sc.Token) != "" {
		return sc.Token
	}
	if name := strings.TrimSpace(sc.TokenEnv); name != "" {
		return os.Getenv(name)
	}
	return ""
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
	if err := os.MkdirAll(filepath.Dir(d.path), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	for k, v := range m {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return os.WriteFile(d.path, []byte(b.String()), 0o600)
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
