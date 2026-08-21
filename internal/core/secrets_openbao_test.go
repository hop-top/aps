//go:build openbao

package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hop.top/kit/go/storage/secret"
)

// openBaoStub speaks enough of the OpenBao/Vault KV v2 API to serve reads, so
// the backend is exercised over real HTTP rather than merely constructed.
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

// The tag must actually register the backend; without it the build reports
// how to enable it (see TestUnavailableBackendReportsBuildTag).
func TestOpenBaoIsAvailableUnderBuildTag(t *testing.T) {
	if !backendAvailable(SecretsBackendOpenBao) {
		t.Fatal("built with -tags openbao but the backend is not registered")
	}
	if !contains(AvailableSecretsBackends(), SecretsBackendOpenBao) {
		t.Fatal("openbao compiled in but missing from AvailableSecretsBackends")
	}
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

func TestOpenBaoAuthenticatesViaTokenEnv(t *testing.T) {
	t.Setenv("APS_VAULT_TOKEN", "test-token")

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
}

// The full aps path: a service bound to secret:NAME resolves through openbao.
func TestResolveSecretValueThroughOpenBao(t *testing.T) {
	ResetProfileSecretCache()
	t.Cleanup(ResetProfileSecretCache)

	srv := openBaoStub(t, "secret", map[string]string{"twilio_sid": "AC-from-openbao"})
	writeSecretsConfig(t, "secrets:\n  backend: openbao\n  addr: "+srv.URL+"\n  token: test-token\n")

	got, found := ResolveSecretValue("secret:twilio_sid", ProfileSecretLookup("p1"), EnvSecretLookup)
	if !found || got != "AC-from-openbao" {
		t.Fatalf("got (%q, %v), want (AC-from-openbao, true)", got, found)
	}
}

func TestOpenBaoRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SecretsConfig
		wantErr string
	}{
		{name: "without addr", cfg: SecretsConfig{Token: "tok"}, wantErr: "secrets.addr"},
		{name: "without token", cfg: SecretsConfig{Addr: "https://x"}, wantErr: "secrets.token"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := secretBackendConfig(tc.cfg, SecretsBackendOpenBao, "prof")
			if err == nil {
				t.Fatalf("expected an error mentioning %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}
