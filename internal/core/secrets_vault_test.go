package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/kit/go/storage/secret"
)

// infisicalStub speaks enough of the Infisical v3 raw-secrets API for the kit
// store to read from it, so the backend is exercised over real HTTP rather
// than merely constructed.
func infisicalStub(t *testing.T, secrets map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v3/secrets/raw") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/api/v3/secrets/raw")
		key = strings.TrimPrefix(key, "/")

		w.Header().Set("Content-Type", "application/json")
		if key == "" { // list
			type rawSecret struct {
				SecretKey   string `json:"secretKey"`
				SecretValue string `json:"secretValue"`
			}
			out := struct {
				Secrets []rawSecret `json:"secrets"`
			}{}
			for k, v := range secrets {
				out.Secrets = append(out.Secrets, rawSecret{SecretKey: k, SecretValue: v})
			}
			_ = json.NewEncoder(w).Encode(out)
			return
		}
		value, ok := secrets[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"secret": map[string]string{"secretKey": key, "secretValue": value},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// openBaoStub speaks enough of the OpenBao/Vault KV v2 API to serve reads.
func openBaoStub(t *testing.T, mount string, secrets map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Vault-Token"); got != "test-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// KV v2 list: /v1/<mount>/metadata/<prefix>?list=true
		if strings.Contains(r.URL.Path, "/metadata/") || r.URL.Query().Get("list") == "true" {
			keys := make([]string, 0, len(secrets))
			for k := range secrets {
				keys = append(keys, k)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"keys": keys}})
			return
		}

		// KV v2 read: /v1/<mount>/data/<key>
		prefix := "/v1/" + mount + "/data/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, prefix)
		value, ok := secrets[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": map[string]any{"value": value}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOpenBaoBackendReadsSecrets(t *testing.T) {
	srv := openBaoStub(t, "secret", map[string]string{"twilio_sid": "AC-from-openbao"})

	cfg := SecretsConfig{Backend: SecretsBackendOpenBao, Addr: srv.URL, Token: "test-token"}
	kitCfg, err := secretBackendConfig(cfg, SecretsBackendOpenBao, "prof")
	if err != nil {
		t.Fatalf("secretBackendConfig: %v", err)
	}
	store, err := secret.Open(kitCfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	got, err := store.Get(context.Background(), "twilio_sid")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Value) != "AC-from-openbao" {
		t.Fatalf("value = %q, want AC-from-openbao", got.Value)
	}

	// LoadProfileSecrets drains via List, so List must work too.
	keys, err := store.List(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0] != "twilio_sid" {
		t.Fatalf("keys = %v, want [twilio_sid]", keys)
	}
}

func TestOpenBaoBackendCustomMount(t *testing.T) {
	srv := openBaoStub(t, "kv", map[string]string{"tok": "from-custom-mount"})

	kitCfg, err := secretBackendConfig(
		SecretsConfig{Addr: srv.URL, Token: "test-token", Mount: "kv"},
		SecretsBackendOpenBao, "prof")
	if err != nil {
		t.Fatal(err)
	}
	store, err := secret.Open(kitCfg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), "tok")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Value) != "from-custom-mount" {
		t.Fatalf("value = %q, want from-custom-mount", got.Value)
	}
}

func TestOpenBaoBackendMissingKeyIsNotFound(t *testing.T) {
	srv := openBaoStub(t, "secret", map[string]string{})

	kitCfg, _ := secretBackendConfig(
		SecretsConfig{Addr: srv.URL, Token: "test-token"}, SecretsBackendOpenBao, "prof")
	store, err := secret.Open(kitCfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), "absent"); err == nil {
		t.Fatal("expected an error for a missing key")
	}
}

func TestInfisicalBackendReadsSecrets(t *testing.T) {
	srv := infisicalStub(t, map[string]string{"twilio_sid": "AC-from-infisical"})

	cfg := SecretsConfig{
		Backend: SecretsBackendInfisical,
		Addr:    srv.URL,
		Token:   "test-token",
		Project: "proj",
		Env:     "prod",
	}
	kitCfg, err := secretBackendConfig(cfg, SecretsBackendInfisical, "prof")
	if err != nil {
		t.Fatalf("secretBackendConfig: %v", err)
	}
	store, err := secret.Open(kitCfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	got, err := store.Get(context.Background(), "twilio_sid")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Value) != "AC-from-infisical" {
		t.Fatalf("value = %q, want AC-from-infisical", got.Value)
	}

	keys, err := store.List(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0] != "twilio_sid" {
		t.Fatalf("keys = %v, want [twilio_sid]", keys)
	}
}

func TestInfisicalBackendMissingKeyIsNotFound(t *testing.T) {
	srv := infisicalStub(t, map[string]string{})

	kitCfg, _ := secretBackendConfig(
		SecretsConfig{Addr: srv.URL, Token: "test-token", Project: "p", Env: "prod"},
		SecretsBackendInfisical, "prof")
	store, err := secret.Open(kitCfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), "absent"); err == nil {
		t.Fatal("expected an error for a missing key")
	}
}

// The token indirection must reach the backend: a vault credential supplied
// via token_env has to authenticate a real request.
func TestVaultBackendsAuthenticateViaTokenEnv(t *testing.T) {
	t.Setenv("APS_VAULT_TOKEN", "test-token")

	t.Run("openbao", func(t *testing.T) {
		srv := openBaoStub(t, "secret", map[string]string{"k": "v"})
		kitCfg, err := secretBackendConfig(
			SecretsConfig{Addr: srv.URL, TokenEnv: "APS_VAULT_TOKEN"}, SecretsBackendOpenBao, "prof")
		if err != nil {
			t.Fatal(err)
		}
		store, err := secret.Open(kitCfg)
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.Get(context.Background(), "k")
		if err != nil {
			t.Fatalf("get with token_env: %v", err)
		}
		if string(got.Value) != "v" {
			t.Fatalf("value = %q, want v", got.Value)
		}
	})

	t.Run("infisical", func(t *testing.T) {
		srv := infisicalStub(t, map[string]string{"k": "v"})
		kitCfg, err := secretBackendConfig(
			SecretsConfig{Addr: srv.URL, TokenEnv: "APS_VAULT_TOKEN", Project: "p", Env: "prod"},
			SecretsBackendInfisical, "prof")
		if err != nil {
			t.Fatal(err)
		}
		store, err := secret.Open(kitCfg)
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.Get(context.Background(), "k")
		if err != nil {
			t.Fatalf("get with token_env: %v", err)
		}
		if string(got.Value) != "v" {
			t.Fatalf("value = %q, want v", got.Value)
		}
	})
}

// writeSecretsConfig points LoadConfig at a throwaway project config. It uses
// the cwd-scoped .aps.yaml layer rather than XDG_CONFIG_HOME: t.Chdir is
// isolated per test, whereas the XDG variable is process-global and races
// tests that set it via os.Setenv while parallel.
func writeSecretsConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".aps.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
}

// The full aps path: a service bound to secret:NAME must resolve that name
// through the configured vault backend. Guards LoadProfileSecrets against
// regaining a hardcoded backend list that omits vault backends.
func TestResolveSecretValueThroughVaultBackends(t *testing.T) {
	t.Run("openbao", func(t *testing.T) {
		ResetProfileSecretCache()
		t.Cleanup(ResetProfileSecretCache)

		srv := openBaoStub(t, "secret", map[string]string{"twilio_sid": "AC-from-openbao"})
		writeSecretsConfig(t, "secrets:\n  backend: openbao\n  addr: "+srv.URL+"\n  token: test-token\n")

		got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
		if !found || got != "AC-from-openbao" {
			t.Fatalf("got (%q, %v), want (AC-from-openbao, true)", got, found)
		}
	})

	t.Run("infisical", func(t *testing.T) {
		ResetProfileSecretCache()
		t.Cleanup(ResetProfileSecretCache)

		srv := infisicalStub(t, map[string]string{"twilio_sid": "AC-from-infisical"})
		writeSecretsConfig(t, "secrets:\n  backend: infisical\n  addr: "+srv.URL+
			"\n  token: test-token\n  project: proj\n  env: prod\n")

		got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
		if !found || got != "AC-from-infisical" {
			t.Fatalf("got (%q, %v), want (AC-from-infisical, true)", got, found)
		}
	})
}

// A vault-backed profile must still refuse to substitute an ambient variable
// named after the binding key when the secret is absent.
func TestVaultBackedResolutionStillRefusesBindingKeyFallback(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	srv := openBaoStub(t, "secret", map[string]string{})
	writeSecretsConfig(t, "secrets:\n  backend: openbao\n  addr: "+srv.URL+"\n  token: test-token\n")
	t.Setenv("TWILIO_ACCOUNT_SID", "WRONG-ambient-value")

	got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
	if found || got != "" {
		t.Fatalf("got (%q, %v), want empty and not-found", got, found)
	}
}
