package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/google/uuid"
)

// Storage implements a2asrv.TaskStore interface for custom A2A task persistence
type Storage struct {
	config *StorageConfig
}

var _ a2asrv.TaskStore = (*Storage)(nil)

// NewStorage creates a new Storage instance
func NewStorage(config *StorageConfig) (*Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("storage config cannot be nil")
	}

	// Create directories
	tasksPath := config.TasksPath
	if tasksPath == "" {
		tasksPath = filepath.Join(config.BasePath, "tasks")
	}
	agentCardsPath := config.AgentCardsPath
	if agentCardsPath == "" {
		agentCardsPath = filepath.Join(config.BasePath, "agent-cards")
	}
	ipcPath := config.IPCPath
	if ipcPath == "" {
		ipcPath = filepath.Join(config.BasePath, "..", "ipc", "queues")
	}

	if err := os.MkdirAll(tasksPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}
	if err := os.MkdirAll(agentCardsPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create agent-cards directory: %w", err)
	}
	if err := os.MkdirAll(ipcPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create ipc directory: %w", err)
	}

	config.TasksPath = tasksPath
	config.AgentCardsPath = agentCardsPath
	config.IPCPath = ipcPath

	return &Storage{config: config}, nil
}

// Save implements a2asrv.TaskStore interface
func (s *Storage) Save(ctx context.Context, task *a2a.Task, event a2a.Event, prev a2a.TaskVersion) (a2a.TaskVersion, error) {
	taskDir := filepath.Join(s.config.TasksPath, string(task.ID))

	// Create task directory if it doesn't exist
	if err := os.MkdirAll(taskDir, 0700); err != nil {
		return 0, ErrStorageFailed("create task directory", err)
	}

	// Save task metadata
	metaPath := filepath.Join(taskDir, "meta.json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return 0, ErrStorageFailed("marshal task", err)
	}
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return 0, ErrStorageFailed("write task", err)
	}

	// Save the event that triggered this update
	eventData, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return 0, ErrStorageFailed("marshal event", err)
	}
	eventPath := filepath.Join(taskDir, fmt.Sprintf("event_%d_%s.json", time.Now().UnixNano(), uuid.New().String()))
	if err := os.WriteFile(eventPath, eventData, 0600); err != nil {
		return 0, ErrStorageFailed("write event", err)
	}

	// Generate new version - increment previous version
	var newVersion a2a.TaskVersion
	if prev == 0 {
		newVersion = 1
	} else {
		newVersion = prev + 1
	}

	return newVersion, nil
}

// Get implements a2asrv.TaskStore interface
func (s *Storage) Get(ctx context.Context, taskID a2a.TaskID) (*a2a.Task, a2a.TaskVersion, error) {
	taskDir := filepath.Join(s.config.TasksPath, string(taskID))
	metaPath := filepath.Join(taskDir, "meta.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, a2a.ErrTaskNotFound
		}
		return nil, 0, ErrStorageFailed("read task", err)
	}

	var task a2a.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, 0, ErrStorageFailed("unmarshal task", err)
	}

	return &task, 1, nil
}

// List implements a2asrv.TaskStore interface
func (s *Storage) List(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	entries, err := os.ReadDir(s.config.TasksPath)
	if err != nil {
		return nil, ErrStorageFailed("read tasks directory", err)
	}

	tasks := make([]*a2a.Task, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := a2a.TaskID(entry.Name())
		task, _, err := s.Get(ctx, taskID)
		if err != nil {
			if err == a2a.ErrTaskNotFound {
				continue
			}
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return &a2a.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: "",
	}, nil
}

// SaveAgentCard saves an Agent Card to be cache
func (s *Storage) SaveAgentCard(agentID string, card *a2a.AgentCard) error {
	cardPath := filepath.Join(s.config.AgentCardsPath, fmt.Sprintf("%s.json", agentID))
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return ErrStorageFailed("marshal agent card", err)
	}
	if err := os.WriteFile(cardPath, data, 0600); err != nil {
		return ErrStorageFailed("write agent card", err)
	}
	return nil
}

// GetAgentCard retrieves an Agent Card from be cache
func (s *Storage) GetAgentCard(agentID string) (*a2a.AgentCard, error) {
	cardPath := filepath.Join(s.config.AgentCardsPath, fmt.Sprintf("%s.json", agentID))
	data, err := os.ReadFile(cardPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAgentCardNotFound
		}
		return nil, ErrStorageFailed("read agent card", err)
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, ErrStorageFailed("unmarshal agent card", err)
	}

	return &card, nil
}

// DeleteAgentCard removes an Agent Card from be cache
func (s *Storage) DeleteAgentCard(agentID string) error {
	cardPath := filepath.Join(s.config.AgentCardsPath, fmt.Sprintf("%s.json", agentID))
	if err := os.Remove(cardPath); err != nil && !os.IsNotExist(err) {
		return ErrStorageFailed("delete agent card", err)
	}
	return nil
}

// CreateMessageFile saves a message to task directory
func (s *Storage) CreateMessageFile(taskID a2a.TaskID, message *a2a.Message) error {
	taskDir := filepath.Join(s.config.TasksPath, string(taskID))
	messagesDir := filepath.Join(taskDir, "messages")
	if err := os.MkdirAll(messagesDir, 0700); err != nil {
		return ErrStorageFailed("create messages directory", err)
	}

	messagePath := filepath.Join(messagesDir, fmt.Sprintf("%s.json", message.ID))
	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return ErrStorageFailed("marshal message", err)
	}
	if err := os.WriteFile(messagePath, data, 0600); err != nil {
		return ErrStorageFailed("write message", err)
	}
	return nil
}

// GetBasePath returns base storage path
func (s *Storage) GetBasePath() string {
	return s.config.BasePath
}

// GetTasksPath returns tasks storage path
func (s *Storage) GetTasksPath() string {
	return s.config.TasksPath
}
