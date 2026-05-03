package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hop.top/kit/go/runtime/domain"
)

// writeInstancesFile sets APS_INSTANCES_PATH to a temp file containing
// the supplied YAML. Returns the path; t.Cleanup restores the env.
func writeInstancesFile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "instances.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	prev, had := os.LookupEnv("APS_INSTANCES_PATH")
	os.Setenv("APS_INSTANCES_PATH", path)
	t.Cleanup(func() {
		if had {
			os.Setenv("APS_INSTANCES_PATH", prev)
		} else {
			os.Unsetenv("APS_INSTANCES_PATH")
		}
	})
	return path
}

// pointInstancesPathAt sets APS_INSTANCES_PATH to a path that does not
// exist (used to verify the missing-file branch).
func pointInstancesPathAt(t *testing.T, path string) {
	t.Helper()
	prev, had := os.LookupEnv("APS_INSTANCES_PATH")
	os.Setenv("APS_INSTANCES_PATH", path)
	t.Cleanup(func() {
		if had {
			os.Setenv("APS_INSTANCES_PATH", prev)
		} else {
			os.Unsetenv("APS_INSTANCES_PATH")
		}
	})
}

func TestResolve_EmptyName(t *testing.T) {
	// No env override; empty name must be a no-op default regardless of
	// whether instances.yaml exists.
	inst, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve(\"\") err = %v, want nil", err)
	}
	if inst == nil {
		t.Fatal("Resolve(\"\") returned nil instance")
	}
	if (*inst != Instance{}) {
		t.Errorf("Resolve(\"\") = %+v, want zero-value", inst)
	}
}

func TestResolve_KnownInstance(t *testing.T) {
	writeInstancesFile(t, `instances:
  prod:
    directory_endpoint: https://dir.prod.example.com
    a2a_registry_endpoint: https://a2a.prod.example.com
  staging:
    directory_endpoint: https://dir.staging.example.com
`)

	inst, err := Resolve("prod")
	if err != nil {
		t.Fatalf("Resolve(\"prod\") err = %v, want nil", err)
	}
	if inst.Name != "prod" {
		t.Errorf("Name = %q, want %q", inst.Name, "prod")
	}
	if inst.DirectoryEndpoint != "https://dir.prod.example.com" {
		t.Errorf("DirectoryEndpoint = %q", inst.DirectoryEndpoint)
	}
	if inst.A2ARegistryEndpoint != "https://a2a.prod.example.com" {
		t.Errorf("A2ARegistryEndpoint = %q", inst.A2ARegistryEndpoint)
	}
}

func TestResolve_UnknownInstance(t *testing.T) {
	writeInstancesFile(t, `instances:
  prod:
    directory_endpoint: https://dir.prod.example.com
`)

	_, err := Resolve("nonsuch")
	if err == nil {
		t.Fatal("Resolve(\"nonsuch\") err = nil, want not-found")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Resolve(\"nonsuch\") err = %v, want errors.Is(domain.ErrNotFound)", err)
	}
}

func TestResolve_MissingFile(t *testing.T) {
	dir := t.TempDir()
	pointInstancesPathAt(t, filepath.Join(dir, "does-not-exist.yaml"))

	// Empty name still fine.
	if _, err := Resolve(""); err != nil {
		t.Errorf("Resolve(\"\") with missing file err = %v, want nil", err)
	}

	// Non-empty name should be not-found.
	_, err := Resolve("prod")
	if err == nil {
		t.Fatal("Resolve(\"prod\") with missing file err = nil, want not-found")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want errors.Is(domain.ErrNotFound)", err)
	}
}

func TestResolve_MalformedYAML(t *testing.T) {
	writeInstancesFile(t, "this: is: not: valid")

	_, err := Resolve("prod")
	if err == nil {
		t.Fatal("Resolve on malformed YAML err = nil, want parse error")
	}
	if errors.Is(err, domain.ErrNotFound) {
		t.Errorf("malformed YAML err should not be domain.ErrNotFound, got %v", err)
	}
}
