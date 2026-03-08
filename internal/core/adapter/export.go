package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ExportBundle represents an adapter export.
type ExportBundle struct {
	Adapter    Adapter         `yaml:"adapter"`
	Manifest   AdapterManifest `yaml:"manifest,omitempty"`
	ExportedAt time.Time       `yaml:"exported_at"`
}

// ExportToYAML exports an adapter to a YAML file at the given path.
func ExportToYAML(a *Adapter, outputPath string) error {
	bundle := ExportBundle{
		Adapter:    *a,
		ExportedAt: time.Now(),
	}

	data, err := yaml.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("failed to marshal adapter: %w", err)
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportFromYAML imports an adapter from a YAML file.
func ImportFromYAML(inputPath string) (*ExportBundle, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var bundle ExportBundle
	if err := yaml.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse import file: %w", err)
	}

	if bundle.Adapter.Name == "" {
		return nil, fmt.Errorf("imported adapter has no name")
	}

	return &bundle, nil
}
