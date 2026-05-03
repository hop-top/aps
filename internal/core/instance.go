// Package core — instance resolver (T-0412).
//
// An Instance is a named bundle of backend endpoints aps can target. It
// covers the multi-host concerns aps surfaces today: AGNTCY Directory,
// A2A registries, observability collectors, and the bus hub. A user
// types `aps --instance prod directory discover ...` and the relevant
// subcommand resolves the name through Resolve to pick the right URL.
//
// Config lives at $XDG_CONFIG_HOME/aps/instances.yaml:
//
//	instances:
//	  prod:
//	    directory_endpoint: https://dir.example.com
//	    a2a_registry_endpoint: https://a2a.example.com
//	    observability_endpoint: http://otel.example.com:4317
//	    bus_hub_endpoint: wss://bus.example.com
//	  staging:
//	    directory_endpoint: https://dir.staging.example.com
//
// Resolve("") is the no-op default — returns a zero-value *Instance and
// nil error so callers can ignore the value when no --instance was set.
// Resolve("missing") returns a wrapped domain.ErrNotFound (exit code 3
// per internal/cli/exit). Empty fields on a known instance fall through
// to the consumer's existing per-profile or hardcoded defaults — the
// resolver never fabricates URLs.
package core

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"hop.top/kit/go/core/xdg"
	"hop.top/kit/go/runtime/domain"
)

// Instance is a named bundle of backend endpoints. Empty fields mean
// "no override" — the consumer keeps its existing default.
type Instance struct {
	Name                  string `yaml:"-"`
	DirectoryEndpoint     string `yaml:"directory_endpoint,omitempty"`
	A2ARegistryEndpoint   string `yaml:"a2a_registry_endpoint,omitempty"`
	ObservabilityEndpoint string `yaml:"observability_endpoint,omitempty"`
	BusHubEndpoint        string `yaml:"bus_hub_endpoint,omitempty"`
}

// instancesFile mirrors the YAML on disk.
type instancesFile struct {
	Instances map[string]*Instance `yaml:"instances"`
}

// instancesPath returns the path to instances.yaml. Override-able for
// tests via APS_INSTANCES_PATH.
func instancesPath() (string, error) {
	if p := os.Getenv("APS_INSTANCES_PATH"); p != "" {
		return p, nil
	}
	dir, err := xdg.ConfigDir("aps")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "instances.yaml"), nil
}

// Resolve returns the named Instance from instances.yaml.
//
// Empty name → zero-value *Instance, nil error (no-op default; preserves
// pre-T-0412 single-host behavior).
//
// Unknown name → wrapped domain.ErrNotFound (exit code 3).
//
// Missing instances.yaml is treated as "no instances declared": empty
// name still succeeds; any non-empty name is not found.
func Resolve(name string) (*Instance, error) {
	if name == "" {
		return &Instance{}, nil
	}

	path, err := instancesPath()
	if err != nil {
		return nil, fmt.Errorf("resolve instance %q: %w", name, err)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- path resolved via xdg or test env override
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("instance %q: %w", name, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var file instancesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	inst, ok := file.Instances[name]
	if !ok || inst == nil {
		return nil, fmt.Errorf("instance %q: %w", name, domain.ErrNotFound)
	}
	inst.Name = name
	return inst, nil
}
