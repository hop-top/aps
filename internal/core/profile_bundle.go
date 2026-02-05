package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const ProfileBundleVersion = "v1"

type ProfileBundle struct {
	Version   string         `yaml:"version"`
	SourceID  string         `yaml:"source_id,omitempty"`
	CreatedAt string         `yaml:"created_at,omitempty"`
	Profile   Profile        `yaml:"profile"`
	Actions   []BundleAction `yaml:"actions,omitempty"`
}

type BundleAction struct {
	ID           string `yaml:"id"`
	Title        string `yaml:"title,omitempty"`
	Entrypoint   string `yaml:"entrypoint"`
	AcceptsStdin bool   `yaml:"accepts_stdin,omitempty"`
	Content      string `yaml:"content"`
}

func ExportProfileBundle(profileID, destPath string) (*ProfileBundle, error) {
	if profileID == "" {
		return nil, fmt.Errorf("profile id is required")
	}
	if destPath == "" {
		return nil, fmt.Errorf("destination path is required")
	}

	profile, err := LoadProfile(profileID)
	if err != nil {
		return nil, err
	}

	actions, err := LoadActions(profileID)
	if err != nil {
		return nil, err
	}

	bundleActions := make([]BundleAction, 0, len(actions))
	for _, action := range actions {
		content, err := os.ReadFile(action.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read action %s: %w", action.ID, err)
		}

		bundleActions = append(bundleActions, BundleAction{
			ID:           action.ID,
			Title:        action.Title,
			Entrypoint:   action.Entrypoint,
			AcceptsStdin: action.AcceptsStdin,
			Content:      string(content),
		})
	}

	bundle := &ProfileBundle{
		Version:   ProfileBundleVersion,
		SourceID:  profileID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Profile:   *profile,
		Actions:   bundleActions,
	}

	data, err := yaml.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile bundle: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create bundle directory: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write bundle: %w", err)
	}

	return bundle, nil
}

func ImportProfileBundle(bundlePath, newID string, force bool) (*Profile, *ProfileBundle, error) {
	if bundlePath == "" {
		return nil, nil, fmt.Errorf("bundle path is required")
	}

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read bundle: %w", err)
	}

	var bundle ProfileBundle
	if err := yaml.Unmarshal(data, &bundle); err != nil {
		return nil, nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	targetID := newID
	if targetID == "" {
		targetID = bundle.Profile.ID
	}
	if targetID == "" {
		return nil, nil, fmt.Errorf("profile id is required")
	}

	profileDir, err := GetProfileDir(targetID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(profileDir); err == nil {
		if !force {
			return nil, nil, fmt.Errorf("profile '%s' already exists", targetID)
		}
		if err := os.RemoveAll(profileDir); err != nil {
			return nil, nil, fmt.Errorf("failed to remove existing profile: %w", err)
		}
	}

	profile := bundle.Profile
	profile.ID = targetID
	if err := CreateProfile(targetID, profile); err != nil {
		return nil, nil, err
	}

	if len(bundle.Actions) > 0 {
		actionsDir := filepath.Join(profileDir, "actions")
		for _, action := range bundle.Actions {
			if err := validateBundleAction(action); err != nil {
				return nil, nil, err
			}
			actionPath := filepath.Join(actionsDir, action.Entrypoint)
			if err := os.WriteFile(actionPath, []byte(action.Content), 0644); err != nil {
				return nil, nil, fmt.Errorf("failed to write action %s: %w", action.ID, err)
			}
		}

		manifest := ActionManifest{Actions: make([]Action, 0, len(bundle.Actions))}
		for _, action := range bundle.Actions {
			manifest.Actions = append(manifest.Actions, Action{
				ID:           action.ID,
				Title:        action.Title,
				Entrypoint:   action.Entrypoint,
				AcceptsStdin: action.AcceptsStdin,
			})
		}

		manifestBytes, err := yaml.Marshal(&manifest)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal actions manifest: %w", err)
		}
		if err := os.WriteFile(filepath.Join(profileDir, "actions.yaml"), manifestBytes, 0644); err != nil {
			return nil, nil, fmt.Errorf("failed to write actions manifest: %w", err)
		}
	}

	return &profile, &bundle, nil
}

func validateBundleAction(action BundleAction) error {
	if action.ID == "" {
		return fmt.Errorf("action id is required")
	}
	if action.Entrypoint == "" {
		return fmt.Errorf("entrypoint is required for action %s", action.ID)
	}
	if filepath.IsAbs(action.Entrypoint) {
		return fmt.Errorf("entrypoint must be relative for action %s", action.ID)
	}
	clean := filepath.Clean(action.Entrypoint)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("entrypoint must stay within actions directory for action %s", action.ID)
	}
	if filepath.Base(clean) != clean {
		return fmt.Errorf("entrypoint must be a file name for action %s", action.ID)
	}

	return nil
}
