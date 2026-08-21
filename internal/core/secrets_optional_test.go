package core

import (
	"strings"
	"testing"
)

// Every optional backend must name a real build tag, and must appear in the
// canonical backend list. Guards against an entry that can never be enabled.
func TestOptionalBackendsAreCoherent(t *testing.T) {
	for backend, tag := range optionalBackends {
		if strings.TrimSpace(tag) == "" {
			t.Errorf("backend %q is optional but names no build tag", backend)
		}
		found := false
		for _, known := range SecretsBackends {
			if known == backend {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("optional backend %q is not in SecretsBackends", backend)
		}
	}
}

// AvailableSecretsBackends must report only what this binary can open, and
// must never advertise an opt-in backend that was compiled out.
func TestAvailableSecretsBackendsMatchesBuild(t *testing.T) {
	available := AvailableSecretsBackends()

	for _, backend := range available {
		if !backendAvailable(backend) {
			t.Errorf("AvailableSecretsBackends lists %q, which is not compiled in", backend)
		}
	}

	// Backends with no build tag are always available.
	for _, backend := range SecretsBackends {
		if _, optional := optionalBackends[backend]; optional {
			continue
		}
		if !contains(available, backend) {
			t.Errorf("always-available backend %q missing from AvailableSecretsBackends", backend)
		}
	}
}

// Selecting a backend that was compiled out must say so, and say how to fix
// it — not fail with a bare "unknown backend".
func TestUnavailableBackendReportsBuildTag(t *testing.T) {
	for backend, tag := range optionalBackends {
		if backendAvailable(backend) {
			t.Logf("%s is compiled in; skipping unavailable-path check", backend)
			continue
		}
		_, err := secretBackendConfig(
			SecretsConfig{Addr: "https://x", Token: "t", Vault: "v", Project: "p", Env: "e"},
			backend, "prof")
		if err == nil {
			t.Fatalf("backend %q is not compiled in but produced no error", backend)
		}
		if !strings.Contains(err.Error(), "-tags "+tag) {
			t.Fatalf("error for %q = %q; want it to name `-tags %s`", backend, err, tag)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
