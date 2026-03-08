package bundle

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed all:assets
var builtinFS embed.FS

// LoadBuiltins loads all built-in bundle definitions embedded in the binary.
// Returns an empty slice (not an error) if no YAML files are embedded yet.
func LoadBuiltins() ([]Bundle, error) {
	var bundles []Bundle

	entries, err := fs.ReadDir(builtinFS, "assets")
	if err != nil {
		// Directory exists but is unreadable — surface the error.
		return nil, fmt.Errorf("bundle: failed to read embedded bundles dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := "assets/" + name
		data, err := builtinFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("bundle: failed to read embedded file %s: %w", name, err)
		}

		var b Bundle
		if err := yaml.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("bundle: failed to parse embedded bundle %s: %w", name, err)
		}

		bundles = append(bundles, b)
	}

	return bundles, nil
}

// LoadUserOverrides loads bundle definitions from ~/.config/aps/bundles/*.yaml.
// User-defined bundles override built-ins with the same name.
// Returns an empty slice (not an error) if the directory does not exist.
func LoadUserOverrides() ([]Bundle, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to determine user config dir: %w", err)
	}

	bundlesDir := filepath.Join(configDir, "aps", "bundles")

	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("bundle: failed to read user bundles dir %s: %w", bundlesDir, err)
	}

	var bundles []Bundle
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(bundlesDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("bundle: failed to read user bundle %s: %w", name, err)
		}

		var b Bundle
		if err := yaml.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("bundle: failed to parse user bundle %s: %w", name, err)
		}

		bundles = append(bundles, b)
	}

	return bundles, nil
}
