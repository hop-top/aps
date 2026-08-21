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

// The full aps path for an always-available vault backend: a service bound to
// secret:NAME must resolve that name through the configured store.
func TestResolveSecretValueThroughInfisical(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	srv := infisicalStub(t, map[string]string{"twilio_sid": "AC-from-infisical"})
	writeSecretsConfig(t, "secrets:\n  backend: infisical\n  addr: "+srv.URL+
		"\n  token: test-token\n  project: proj\n  env: prod\n")

	got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
	if !found || got != "AC-from-infisical" {
		t.Fatalf("got (%q, %v), want (AC-from-infisical, true)", got, found)
	}
}

// A vault-backed profile must still refuse to substitute an ambient variable
// named after the binding key when the secret is absent.
func TestVaultBackedResolutionStillRefusesBindingKeyFallback(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	srv := infisicalStub(t, map[string]string{})
	writeSecretsConfig(t, "secrets:\n  backend: infisical\n  addr: "+srv.URL+
		"\n  token: test-token\n  project: proj\n  env: prod\n")
	t.Setenv("TWILIO_ACCOUNT_SID", "WRONG-ambient-value")

	got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
	if found || got != "" {
		t.Fatalf("got (%q, %v), want empty and not-found", got, found)
	}
}

// A vault credential supplied via token_env must authenticate a real request.
func TestInfisicalAuthenticatesViaTokenEnv(t *testing.T) {
	t.Setenv("APS_VAULT_TOKEN", "test-token")

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
}
