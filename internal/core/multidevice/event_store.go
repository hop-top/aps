package multidevice

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// eventIndex tracks metadata about the event store.
type eventIndex struct {
	LatestVersion int64 `json:"latest_version"`
	EventCount    int   `json:"event_count"`
}

// EventStore persists workspace events as JSON lines.
type EventStore struct {
	mu          sync.RWMutex
	workspaceID string
}

// NewEventStore creates a new EventStore for the given workspace.
func NewEventStore(workspaceID string) *EventStore {
	return &EventStore{
		workspaceID: workspaceID,
	}
}

// eventsDir returns the events directory path for this store's workspace.
func (s *EventStore) eventsDir() (string, error) {
	wsDir, err := GetWorkspaceDir(s.workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "events"), nil
}

// eventsFilePath returns the path to the events JSONL file.
func (s *EventStore) eventsFilePath() (string, error) {
	dir, err := s.eventsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "events.jsonl"), nil
}

// indexFilePath returns the path to the index JSON file.
func (s *EventStore) indexFilePath() (string, error) {
	dir, err := s.eventsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.json"), nil
}

// loadIndex reads the current event index from disk.
func (s *EventStore) loadIndex() (*eventIndex, error) {
	indexPath, err := s.indexFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &eventIndex{LatestVersion: 0, EventCount: 0}, nil
		}
		return nil, fmt.Errorf("failed to read event index: %w", err)
	}

	var idx eventIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse event index: %w", err)
	}

	return &idx, nil
}

// saveIndex writes the event index to disk.
func (s *EventStore) saveIndex(idx *eventIndex) error {
	indexPath, err := s.indexFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event index: %w", err)
	}

	return os.WriteFile(indexPath, data, 0644)
}

// Store appends an event to the store.
func (s *EventStore) Store(event *WorkspaceEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.eventsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create events directory: %w", err)
	}

	idx, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("failed to load event index: %w", err)
	}

	idx.LatestVersion++
	event.Version = idx.LatestVersion

	eventsPath, err := s.eventsFilePath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open events file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	idx.EventCount++
	if err := s.saveIndex(idx); err != nil {
		return fmt.Errorf("failed to update event index: %w", err)
	}

	return nil
}

// GetByID returns the event with the given ID, or an error if not found.
func (s *EventStore) GetByID(eventID string) (*WorkspaceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, err := s.readAllEvents()
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if event.ID == eventID {
			return event, nil
		}
	}

	return nil, fmt.Errorf("event not found: %s", eventID)
}

// GetRange returns events with versions in the range [fromVersion, toVersion] inclusive.
func (s *EventStore) GetRange(fromVersion, toVersion int64) ([]*WorkspaceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, err := s.readAllEvents()
	if err != nil {
		return nil, err
	}

	var result []*WorkspaceEvent
	for _, event := range events {
		if event.Version >= fromVersion && event.Version <= toVersion {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetSince returns all events with a timestamp after the given time.
func (s *EventStore) GetSince(timestamp time.Time) ([]*WorkspaceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, err := s.readAllEvents()
	if err != nil {
		return nil, err
	}

	var result []*WorkspaceEvent
	for _, event := range events {
		if event.Timestamp.After(timestamp) {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetLatestVersion returns the latest event version number.
func (s *EventStore) GetLatestVersion() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, err := s.loadIndex()
	if err != nil {
		return 0, err
	}

	return idx.LatestVersion, nil
}

// Count returns the total number of events in the store.
func (s *EventStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, err := s.loadIndex()
	if err != nil {
		return 0, err
	}

	return idx.EventCount, nil
}

// QueryByResource returns all events in a workspace whose payload references
// the given resource. The resource is matched against the "resource" payload
// key, or synthesized from the event type category and "id"/"name" keys (the
// same logic used by extractResource in the conflict detector).
func (s *EventStore) QueryByResource(workspaceID, resource string) ([]*WorkspaceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, err := s.readAllEvents()
	if err != nil {
		return nil, err
	}

	var result []*WorkspaceEvent
	for _, event := range events {
		if event.WorkspaceID != workspaceID {
			continue
		}
		eventResource := extractResourceFromEvent(event)
		if eventResource == resource {
			result = append(result, event)
		}
	}

	return result, nil
}

// extractResourceFromEvent derives a resource identifier from an event's
// payload. This mirrors the extractResource function in conflict_detector.go
// but lives here to avoid a circular reference.
func extractResourceFromEvent(event *WorkspaceEvent) string {
	if event.Payload == nil {
		return ""
	}

	if res, ok := event.Payload["resource"]; ok {
		if s, ok := res.(string); ok && s != "" {
			return s
		}
	}

	category := event.EventType.Category()
	if id, ok := event.Payload["id"]; ok {
		return fmt.Sprintf("%s:%v", category, id)
	}
	if name, ok := event.Payload["name"]; ok {
		return fmt.Sprintf("%s:%v", category, name)
	}

	return ""
}

// readAllEvents reads and parses every event from the JSONL file.
func (s *EventStore) readAllEvents() ([]*WorkspaceEvent, error) {
	eventsPath, err := s.eventsFilePath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*WorkspaceEvent{}, nil
		}
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer f.Close()

	var events []*WorkspaceEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event WorkspaceEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		events = append(events, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	return events, nil
}
