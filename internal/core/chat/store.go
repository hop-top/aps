package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

func NewStore(dataDir string) *Store {
	return &Store{root: filepath.Join(dataDir, "sessions", "chat")}
}

func (s *Store) Load(sessionID string) (*Transcript, error) {
	path := s.path(sessionID)
	data, err := os.ReadFile(path) // #nosec G304 -- path is rooted under APS data dir.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("read chat transcript: %w", err)
	}
	var transcript Transcript
	if err := json.Unmarshal(data, &transcript); err != nil {
		return nil, fmt.Errorf("parse chat transcript: %w", err)
	}
	return &transcript, nil
}

func (s *Store) Save(transcript *Transcript) error {
	if transcript == nil {
		return fmt.Errorf("chat transcript cannot be nil")
	}
	if transcript.SessionID == "" {
		return fmt.Errorf("chat transcript session ID cannot be empty")
	}
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return fmt.Errorf("create chat transcript dir: %w", err)
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal chat transcript: %w", err)
	}
	if err := os.WriteFile(s.path(transcript.SessionID), data, 0600); err != nil {
		return fmt.Errorf("write chat transcript: %w", err)
	}
	return nil
}

func (s *Store) path(sessionID string) string {
	return filepath.Join(s.root, sessionID+".json")
}
