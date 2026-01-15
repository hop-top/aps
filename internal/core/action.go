package core

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Action represents an executable task within a profile
type Action struct {
	ID           string `yaml:"id"`
	Title        string `yaml:"title"`
	Entrypoint   string `yaml:"entrypoint"`
	AcceptsStdin bool   `yaml:"accepts_stdin"`
	Path         string `yaml:"-"` // Resolved absolute path
	Type         string `yaml:"-"` // Resolved type (sh, py, js)
}

type ActionManifest struct {
	Actions []Action `yaml:"actions"`
}

// LoadActions discovers actions for a profile
func LoadActions(profileID string) ([]Action, error) {
	profileDir, err := GetProfileDir(profileID)
	if err != nil {
		return nil, err
	}
	actionsDir := filepath.Join(profileDir, "actions")

	// 1. Try to load actions.yaml
	manifestPath := filepath.Join(profileDir, "actions.yaml")
	var manifestActions []Action
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest ActionManifest
		if err := yaml.Unmarshal(data, &manifest); err == nil {
			manifestActions = manifest.Actions
		}
	}

	// 2. Map known actions for quick lookup
	known := make(map[string]Action)
	for _, a := range manifestActions {
		// Resolve path relative to actions dir
		a.Path = filepath.Join(actionsDir, a.Entrypoint)
		a.Type = resolveType(a.Entrypoint)
		known[a.ID] = a
	}

	// 3. Scan directory for implicit actions (files not in manifest)
	entries, err := os.ReadDir(actionsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			id := name // default ID is filename (maybe strip extension?)
			// simple strip extension for ID if implicit
			ext := filepath.Ext(name)
			if ext != "" {
				id = name[:len(name)-len(ext)]
			}

			if _, exists := known[id]; !exists {
				// Implicit action
				known[id] = Action{
					ID:           id,
					Title:        name,
					Entrypoint:   name,
					AcceptsStdin: false, // Default false for implicit? Spec doesn't say. Let's assume false or maybe we should detect? Safe default false.
					Path:         filepath.Join(actionsDir, name),
					Type:         resolveType(name),
				}
			}
		}
	}

	// Convert map back to slice
	var results []Action
	for _, a := range known {
		results = append(results, a)
	}

	return results, nil
}

func resolveType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".sh":
		return "sh"
	case ".py":
		return "py"
	case ".js":
		return "js"
	default:
		return "unknown"
	}
}

func GetAction(profileID, actionID string) (*Action, error) {
	actions, err := LoadActions(profileID)
	if err != nil {
		return nil, err
	}
	for _, a := range actions {
		if a.ID == actionID {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("action %s not found in profile %s", actionID, profileID)
}
