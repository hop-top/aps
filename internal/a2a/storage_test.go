package a2a

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	_, err := NewStorage(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, config.TasksPath)
	assert.NotEmpty(t, config.AgentCardsPath)
}

func TestNewStorage_NilConfig(t *testing.T) {
	storage, err := NewStorage(nil)
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func TestNewStorage_CreatesDirs(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	_, err := NewStorage(config)
	require.NoError(t, err)

	assert.DirExists(t, config.TasksPath)
	assert.DirExists(t, config.AgentCardsPath)
}

func TestStorage_Save_Task(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	task := &a2a.Task{
		ID:     a2a.NewTaskID(),
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)

	version, err := storage.Save(ctx, task, event, 0)
	assert.NoError(t, err)
	assert.Equal(t, a2a.TaskVersion(1), version)

	// Verify file exists
	taskDir := filepath.Join(config.TasksPath, string(task.ID))
	assert.DirExists(t, taskDir)
	assert.FileExists(t, filepath.Join(taskDir, "meta.json"))
}

func TestStorage_Get_Task(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()
	originalTask := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)

	_, err = storage.Save(ctx, originalTask, event, 0)
	require.NoError(t, err)

	retrievedTask, version, err := storage.Get(ctx, taskID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	assert.Equal(t, taskID, retrievedTask.ID)
	assert.Greater(t, version, a2a.TaskVersion(0))
}

func TestStorage_Get_TaskNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()

	_, _, err = storage.Get(ctx, taskID)
	assert.Error(t, err)
	assert.Equal(t, a2a.ErrTaskNotFound, err)
}

func TestStorage_List_Tasks(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Save multiple tasks
	taskCount := 3
	for i := 0; i < taskCount; i++ {
		task := &a2a.Task{
			ID:     a2a.NewTaskID(),
			Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
		}
		event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
		_, err = storage.Save(ctx, task, event, 0)
		require.NoError(t, err)
	}

	// List tasks
	response, err := storage.List(ctx, &a2a.ListTasksRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Tasks, taskCount)
}

func TestStorage_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	response, err := storage.List(ctx, &a2a.ListTasksRequest{})
	assert.NoError(t, err)
	assert.Len(t, response.Tasks, 0)
}

func TestStorage_SaveAgentCard(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{
			{ID: "execute", Name: "Execute"},
		},
	}

	err = storage.SaveAgentCard("test-profile", card)
	assert.NoError(t, err)

	// Verify file exists
	cardPath := filepath.Join(config.AgentCardsPath, "test-profile.json")
	assert.FileExists(t, cardPath)
}

func TestStorage_GetAgentCard(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	originalCard := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{
			{ID: "execute", Name: "Execute"},
		},
	}

	err = storage.SaveAgentCard("test-profile", originalCard)
	require.NoError(t, err)

	retrievedCard, err := storage.GetAgentCard("test-profile")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCard)
	assert.Equal(t, "Test Agent", retrievedCard.Name)
	assert.Equal(t, "http://127.0.0.1:8081", retrievedCard.URL)
}

func TestStorage_GetAgentCard_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	_, err = storage.GetAgentCard("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrAgentCardNotFound, err)
}

func TestStorage_DeleteAgentCard(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
	}

	err = storage.SaveAgentCard("test-profile", card)
	require.NoError(t, err)

	err = storage.DeleteAgentCard("test-profile")
	assert.NoError(t, err)

	_, err = storage.GetAgentCard("test-profile")
	assert.Error(t, err)
	assert.Equal(t, ErrAgentCardNotFound, err)
}

func TestStorage_DeleteAgentCard_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	// Should not error when deleting non-existent card
	err = storage.DeleteAgentCard("nonexistent")
	assert.NoError(t, err)
}

func TestStorage_CreateMessageFile(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)

	// Create task first
	_, err = storage.Save(ctx, task, event, 0)
	require.NoError(t, err)

	// Create message
	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})
	err = storage.CreateMessageFile(taskID, message)
	assert.NoError(t, err)

	// Verify file exists
	messagePath := filepath.Join(config.TasksPath, string(taskID), "messages", string(message.ID)+".json")
	assert.FileExists(t, messagePath)
}

func TestStorage_GetBasePath(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	assert.Equal(t, tmpDir, storage.GetBasePath())
}

func TestStorage_GetTasksPath(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	tasksPath := storage.GetTasksPath()
	assert.NotEmpty(t, tasksPath)
	assert.DirExists(t, tasksPath)
}

func TestStorage_Concurrent_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup
	taskCount := 10

	// Concurrent saves
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			task := &a2a.Task{
				ID:     a2a.NewTaskID(),
				Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
			}
			event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)
			_, err := storage.Save(ctx, task, event, 0)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all tasks were saved
	response, err := storage.List(ctx, &a2a.ListTasksRequest{})
	require.NoError(t, err)
	assert.Len(t, response.Tasks, taskCount)
}

func TestStorage_Concurrent_AgentCards(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	cardCount := 5

	// Concurrent agent card saves
	for i := 0; i < cardCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			profileID := "profile-" + string('0'+rune(index))
			card := &a2a.AgentCard{
				Name: "Agent " + string('0'+rune(index)),
				URL:  "http://127.0.0.1:8081",
			}
			err := storage.SaveAgentCard(profileID, card)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all cards were saved
	for i := 0; i < cardCount; i++ {
		profileID := "profile-" + string('0'+rune(i))
		card, err := storage.GetAgentCard(profileID)
		assert.NoError(t, err)
		assert.NotNil(t, card)
	}
}

func TestStorage_VersionIncrement(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	taskID := a2a.NewTaskID()
	task := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	event := a2a.NewStatusUpdateEvent(&a2asrv.RequestContext{}, a2a.TaskStateSubmitted, nil)

	// Save task multiple times
	version1, err := storage.Save(ctx, task, event, 0)
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskVersion(1), version1)

	version2, err := storage.Save(ctx, task, event, version1)
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskVersion(2), version2)

	version3, err := storage.Save(ctx, task, event, version2)
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskVersion(3), version3)
}

func TestStorage_CustomPaths(t *testing.T) {
	tmpDir := t.TempDir()
	tasksDir := filepath.Join(tmpDir, "custom-tasks")
	cardsDir := filepath.Join(tmpDir, "custom-cards")

	config := &StorageConfig{
		BasePath:       tmpDir,
		TasksPath:      tasksDir,
		AgentCardsPath: cardsDir,
	}

	_, err := NewStorage(config)
	require.NoError(t, err)

	assert.DirExists(t, tasksDir)
	assert.DirExists(t, cardsDir)
}

func TestStorage_AgentCardValidation(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	// Test save and retrieve maintains data integrity
	originalCard := &a2a.AgentCard{
		Name:        "Complete Agent",
		Description: "A complete agent card",
		Version:     "1.0.0",
		URL:         "http://127.0.0.1:8081",
		Skills: []a2a.AgentSkill{
			{
				ID:          "execute",
				Name:        "Execute",
				Description: "Execute commands",
			},
		},
	}

	err = storage.SaveAgentCard("complete-profile", originalCard)
	require.NoError(t, err)

	retrievedCard, err := storage.GetAgentCard("complete-profile")
	require.NoError(t, err)

	assert.Equal(t, originalCard.Name, retrievedCard.Name)
	assert.Equal(t, originalCard.Description, retrievedCard.Description)
	assert.Equal(t, originalCard.Version, retrievedCard.Version)
	assert.Equal(t, originalCard.URL, retrievedCard.URL)
	assert.Equal(t, len(originalCard.Skills), len(retrievedCard.Skills))
}
