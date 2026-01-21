package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/docs/*
var docsFS embed.FS

// GenerateDocs extracts embedded documentation to the destination directory
func GenerateDocs(dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	entries, err := docsFS.ReadDir("assets/docs")
	if err != nil {
		return fmt.Errorf("failed to read embedded docs: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := "assets/docs/" + entry.Name()
		data, err := docsFS.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", srcPath, err)
		}

		destPath := filepath.Join(dest, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write doc file %s: %w", destPath, err)
		}
		fmt.Printf("Generated %s\n", destPath)
	}

	return nil
}
