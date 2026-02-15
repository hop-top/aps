package multidevice

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// OfflineQueue stores workspace events locally when a device is offline.
// Events are persisted as JSON lines in
// ~/.aps/workspaces/{workspace_id}/offline/{device_id}.jsonl so they
// survive process restarts.
type OfflineQueue struct {
	deviceID    string
	workspaceID string
	mu          sync.Mutex
}

// NewOfflineQueue creates a queue for the given device and workspace.
func NewOfflineQueue(deviceID, workspaceID string) *OfflineQueue {
	return &OfflineQueue{
		deviceID:    deviceID,
		workspaceID: workspaceID,
	}
}

// queueDir returns the directory for offline queues.
func (q *OfflineQueue) queueDir() (string, error) {
	wsDir, err := GetWorkspaceDir(q.workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "offline"), nil
}

// queuePath returns the path to this device's queue file.
func (q *OfflineQueue) queuePath() (string, error) {
	dir, err := q.queueDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, q.deviceID+".jsonl"), nil
}

// ensureDir creates the offline queue directory if it does not exist.
func (q *OfflineQueue) ensureDir() error {
	dir, err := q.queueDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// Enqueue appends an event to the offline queue.
func (q *OfflineQueue) Enqueue(event *WorkspaceEvent) error {
	if event == nil {
		return fmt.Errorf("event must not be nil")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.ensureDir(); err != nil {
		return fmt.Errorf("creating offline queue directory: %w", err)
	}

	path, err := q.queuePath()
	if err != nil {
		return fmt.Errorf("resolving queue path: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening offline queue: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing to offline queue: %w", err)
	}

	return nil
}

// Dequeue returns and removes all queued events. The queue file is
// deleted after the events are read.
func (q *OfflineQueue) Dequeue() ([]*WorkspaceEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	events, err := q.readAll()
	if err != nil {
		return nil, err
	}

	path, err := q.queuePath()
	if err != nil {
		return nil, fmt.Errorf("resolving queue path: %w", err)
	}

	// Remove the queue file after successfully reading.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing offline queue: %w", err)
	}

	return events, nil
}

// Peek returns queued events without removing them.
func (q *OfflineQueue) Peek() ([]*WorkspaceEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.readAll()
}

// Size returns the number of queued events.
func (q *OfflineQueue) Size() (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	events, err := q.readAll()
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

// Clear removes all queued events by deleting the queue file.
func (q *OfflineQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	path, err := q.queuePath()
	if err != nil {
		return fmt.Errorf("resolving queue path: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing offline queue: %w", err)
	}
	return nil
}

// readAll reads all events from the queue file. Must be called with the
// mutex held.
func (q *OfflineQueue) readAll() ([]*WorkspaceEvent, error) {
	path, err := q.queuePath()
	if err != nil {
		return nil, fmt.Errorf("resolving queue path: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening offline queue: %w", err)
	}
	defer f.Close()

	var events []*WorkspaceEvent
	scanner := bufio.NewScanner(f)

	// Increase buffer size for large payloads.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var event WorkspaceEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines rather than failing the whole read.
			continue
		}
		events = append(events, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading offline queue: %w", err)
	}

	return events, nil
}
