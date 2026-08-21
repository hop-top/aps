package core

import (
	"strings"
	"testing"

	"hop.top/kit/go/storage/secret"
)

// Every advertised backend must actually be openable. Guards against listing a
// backend whose opener kit never registers (as "openbao" is not).
func TestSecretsBackendsAreRegistered(t *testing.T) {
	for _, backend := range SecretsBackends {
		if backend == SecretsBackendFile {
			continue // opened directly, not via the kit registry
		}
		if !backendAvailable(backend) {
			// Opt-in backend compiled out of this build; the unavailable
			// path is covered by TestUnavailableBackendReportsBuildTag.
			continue
		}
		t.Run(backend, func(t *testing.T) {
			cfg := SecretsConfig{
				Backend: backend,
				Vault:   "test-vault",
				Addr:    "https://vault.example",
				Mount:   "secret",
				Project: "test-project",
				Env:     "prod",
				Token:   "test-token",
			}
			kitCfg, err := secretBackendConfig(cfg, backend, "prof")
			if err != nil {
				t.Fatalf("secretBackendConfig: %v", err)
			}
			if _, err := secret.Open(kitCfg); err != nil {
				t.Fatalf("backend %q is advertised but not openable: %v", backend, err)
			}
		})
	}
}

func TestSecretBackendConfigDefaults(t *testing.T) {
	t.Run("env gets aps prefix", func(t *testing.T) {
		got, err := secretBackendConfig(SecretsConfig{Backend: SecretsBackendEnv}, SecretsBackendEnv, "prof")
		if err != nil {
			t.Fatal(err)
		}
		if got.Prefix != "APS_SECRET_" {
			t.Fatalf("Prefix = %q, want APS_SECRET_", got.Prefix)
		}
	})

	t.Run("env prefix override respected", func(t *testing.T) {
		got, err := secretBackendConfig(SecretsConfig{Prefix: "CUSTOM_"}, SecretsBackendEnv, "prof")
		if err != nil {
			t.Fatal(err)
		}
		if got.Prefix != "CUSTOM_" {
			t.Fatalf("Prefix = %q, want CUSTOM_", got.Prefix)
		}
	})

	t.Run("keyring service defaults per profile", func(t *testing.T) {
		got, err := secretBackendConfig(SecretsConfig{}, SecretsBackendKeyring, "my-agent")
		if err != nil {
			t.Fatal(err)
		}
		if got.Service != "aps/my-agent" {
			t.Fatalf("Service = %q, want aps/my-agent", got.Service)
		}
	})

	t.Run("onepassword carries vault and connect url", func(t *testing.T) {
		got, err := secretBackendConfig(
			SecretsConfig{Vault: "Private", ConnectURL: "https://connect.example", Token: "tok"},
			SecretsBackendOnePassword, "prof")
		if err != nil {
			t.Fatal(err)
		}
		if got.Vault != "Private" || got.ConnectURL != "https://connect.example" || got.Token != "tok" {
			t.Fatalf("got %+v, want vault/connect/token preserved", got)
		}
	})
}

func TestSecretBackendConfigRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		cfg     SecretsConfig
		wantErr string
	}{
		{
			name:    "onepassword without vault",
			backend: SecretsBackendOnePassword,
			wantErr: "secrets.vault",
		},
		{
			name:    "infisical without addr",
			backend: SecretsBackendInfisical,
			cfg:     SecretsConfig{Token: "tok", Project: "proj", Env: "prod"},
			wantErr: "secrets.addr",
		},
		{
			name:    "infisical without project",
			backend: SecretsBackendInfisical,
			cfg:     SecretsConfig{Token: "tok", Addr: "https://x", Env: "prod"},
			wantErr: "secrets.project",
		},
		{
			name:    "infisical without token",
			backend: SecretsBackendInfisical,
			cfg:     SecretsConfig{Project: "proj", Addr: "https://x", Env: "prod"},
			wantErr: "secrets.token",
		},
		{
			name:    "infisical without env",
			backend: SecretsBackendInfisical,
			cfg:     SecretsConfig{Project: "proj", Addr: "https://x", Token: "tok"},
			wantErr: "secrets.env",
		},
		{
			name:    "unknown backend",
			backend: "nosuchvault",
			wantErr: `unknown secrets backend "nosuchvault"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := secretBackendConfig(tc.cfg, tc.backend, "prof")
			if err == nil {
				t.Fatalf("expected error mentioning %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

// A vault credential should be readable from the environment so it need not be
// written into the config file.
func TestSecretBackendTokenFromEnv(t *testing.T) {
	t.Setenv("APS_TEST_VAULT_TOKEN", "token-from-env")

	got := secretBackendToken(SecretsConfig{TokenEnv: "APS_TEST_VAULT_TOKEN"})
	if got != "token-from-env" {
		t.Fatalf("token = %q, want token-from-env", got)
	}

	// An explicit token wins over the env indirection.
	got = secretBackendToken(SecretsConfig{Token: "literal", TokenEnv: "APS_TEST_VAULT_TOKEN"})
	if got != "literal" {
		t.Fatalf("token = %q, want literal", got)
	}

	if got := secretBackendToken(SecretsConfig{}); got != "" {
		t.Fatalf("token = %q, want empty", got)
	}
}
