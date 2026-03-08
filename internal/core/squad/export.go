package squad

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/core"
)

// ExportBundle represents a squad export with its member profiles.
type ExportBundle struct {
	Squad      Squad          `yaml:"squad"`
	Profiles   []*core.Profile `yaml:"profiles"`
	ExportedAt time.Time      `yaml:"exported_at"`
}

// Export creates a gzip tarball containing the squad definition and all
// member profiles serialized as a single YAML bundle.
func Export(mgr *Manager, squadID string) ([]byte, error) {
	s, err := mgr.Get(squadID)
	if err != nil {
		return nil, err
	}

	bundle := ExportBundle{
		Squad:      *s,
		ExportedAt: time.Now(),
	}

	for _, profileID := range s.Members {
		profile, err := core.LoadProfile(profileID)
		if err != nil {
			return nil, fmt.Errorf("load member profile %s: %w", profileID, err)
		}
		bundle.Profiles = append(bundle.Profiles, profile)
	}

	squadData, err := yaml.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("marshal bundle: %w", err)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:    "squad-bundle.yaml",
		Size:    int64(len(squadData)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(squadData); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Import reads a gzip tarball produced by Export and returns the bundle.
func Import(r io.Reader) (*ExportBundle, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == "squad-bundle.yaml" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}

			var bundle ExportBundle
			if err := yaml.Unmarshal(data, &bundle); err != nil {
				return nil, fmt.Errorf("parse bundle: %w", err)
			}
			return &bundle, nil
		}
	}

	return nil, fmt.Errorf("squad-bundle.yaml not found in archive")
}
