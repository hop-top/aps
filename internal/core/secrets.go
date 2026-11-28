// Package core secrets bridge.
//
// Profile secrets are routed through hop.top/kit/go/storage/secret so the
// backend is configurable via Config.Secrets.Backend: file, env, keyring,
// onepassword, infisical, or ghsecrets (see SecretsBackends). The default
// "file" backend preserves the legacy per-profile secrets.env layout (godotenv
// format); every other backend is opened through the kit registry.
//
// The openbao backend is opt-in and requires `-tags openbao`, since it pulls
// the Vault API client and its transitive modules. Use
// AvailableSecretsBackends to list what a given binary can actually open.
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hop.top/aps/internal/fsperm"
	"hop.top/kit/go/storage/secret"
	_ "hop.top/kit/go/storage/secret/env"         // register "env"
	_ "hop.top/kit/go/storage/secret/ghsecrets"   // register "ghsecrets"
	_ "hop.top/kit/go/storage/secret/infisical"   // register "infisical"
	_ "hop.top/kit/go/storage/secret/keyring"     // register "keyring"
	_ "hop.top/kit/go/storage/secret/onepassword" // register "onepassword"
)

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

// SecretsBackends lists every backend aps knows about. Some are opt-in at
// build time (see optionalBackends); AvailableSecretsBackends reports the
// subset this binary can actually open.
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
	data, found, err := readSecretsFile(path)
	if err != nil || !found {
		return nil, err
	}
	return parseSecretsEnv(path, data)
}

// readSecretsFile reads the raw secrets.env bytes, warning on insecure
// permissions. found is false for a missing file.
func readSecretsFile(path string) (data []byte, found bool, err error) {
	warning, err := fsperm.CheckSecret(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("check secrets %s: %w", path, err)
	}
	if warning != "" {
		fmt.Fprintf(os.Stderr, "WARNING: Secrets file %s has %s\n", path, warning)
	}
	data, err = os.ReadFile(path) // #nosec G304 -- path is the profile's secrets file
	if err != nil {
		return nil, false, fmt.Errorf("read secrets %s: %w", path, err)
	}
	return data, true, nil
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
	if tag, optional := optionalBackends[backend]; optional && !backendAvailable(backend) {
		return secret.Config{}, fmt.Errorf(
			"secrets backend %q is not compiled into this build; rebuild with -tags %s", backend, tag)
	}
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

// edit sets key to *val (deletes it when val is nil), changing only that
// key's statement in the file. The file is replaced atomically (temp file,
// sync, rename; owner-only throughout), so a failed write leaves the old
// content intact.
func (d *dotenvStore) edit(key string, val *string) error {
	data, _, err := readSecretsFile(d.path)
	if err != nil {
		return err
	}
	out, changed, err := editSecretsEnv(d.path, data, key, val)
	if err != nil || !changed {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.path), 0o700); err != nil {
		return err
	}
	if err := fsperm.WriteSecret(d.path, out, 0o600); err != nil {
		return fmt.Errorf("write secrets %s: %w", d.path, err)
	}
	return nil
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
	v := string(value)
	return d.edit(key, &v)
}

func (d *dotenvStore) Delete(_ context.Context, key string) error {
	return d.edit(key, nil)
}

// ProfileSecretCount returns the number of secret keys the profile holds,
// through the configured backend. It is the one definition of "has
// secrets": a profile has secrets iff this count is above zero. A fresh
// profile's secrets.env holds only a comment template (0 keys), and a
// missing file is also 0. An unreadable or malformed store is an error
// (location only, never content) with a count of 0.
func ProfileSecretCount(profileID string) (int, error) {
	secrets, err := LoadProfileSecrets(profileID)
	if err != nil {
		return 0, err
	}
	return len(secrets), nil
}

// ProfileHasSecrets reports whether ProfileSecretCount is above zero. An
// unreadable store reports false; callers that must tell "none" from
// "unreadable" use ProfileSecretCount.
func ProfileHasSecrets(profileID string) bool {
	n, err := ProfileSecretCount(profileID)
	return err == nil && n > 0
}
