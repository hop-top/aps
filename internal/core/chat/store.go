package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Store struct {
	root string
}

func NewStore(dataDir string) *Store {
	return &Store{root: filepath.Join(dataDir, "sessions", "chat")}
}

func (s *Store) Load(sessionID string) (*Transcript, error) {
	path, err := s.path(sessionID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- sessionID validated; path rooted under APS data dir.
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
	path, err := s.path(transcript.SessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return fmt.Errorf("create chat transcript dir: %w", err)
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal chat transcript: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write chat transcript: %w", err)
	}
	return nil
}

// safeSessionIDPattern restricts sessionIDs to a conservative set that cannot
// contain path separators, "..", NUL bytes, or other characters that would let
// a caller escape the chat transcript directory.
var safeSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (s *Store) path(sessionID string) (string, error) {
	if !safeSessionIDPattern.MatchString(sessionID) {
		return "", fmt.Errorf("invalid chat session id %q", sessionID)
	}
	return filepath.Join(s.root, sessionID+".json"), nil
}
