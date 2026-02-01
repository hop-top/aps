package a2a

import (
	"context"
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)
	assert.NotNil(t, resolver)
}

func TestResolver_ResolveProfile_InvalidProfile(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)
	ctx := context.Background()

	// Try to resolve non-existent profile
	card, err := resolver.ResolveProfile(ctx, "nonexistent-profile")
	assert.Error(t, err)
	assert.Nil(t, card)
}

func TestResolver_ResolveProfile_EmptyProfileID(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)
	ctx := context.Background()

	card, err := resolver.ResolveProfile(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, card)
	assert.Equal(t, ErrInvalidConfig, err)
}

func TestResolver_ResolveFromCard_ValidCard(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)
	ctx := context.Background()

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "http://127.0.0.1:8081",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		Skills: []a2a.AgentSkill{
			{ID: "execute", Name: "Execute"},
		},
	}

	client, err := resolver.ResolveFromCard(ctx, card)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestResolver_ResolveFromCard_NilCard(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)
	ctx := context.Background()

	client, err := resolver.ResolveFromCard(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestResolver_InvalidateCache_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)

	// Save an agent card
	card := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
	}
	err = storage.SaveAgentCard("test-profile", card)
	require.NoError(t, err)

	// Verify it exists
	_, err = storage.GetAgentCard("test-profile")
	require.NoError(t, err)

	// Invalidate cache
	err = resolver.InvalidateCache("test-profile")
	assert.NoError(t, err)

	// Verify it's gone
	_, err = storage.GetAgentCard("test-profile")
	assert.Error(t, err)
	assert.Equal(t, ErrAgentCardNotFound, err)
}

func TestResolver_InvalidateCache_EmptyProfileID(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)

	err = resolver.InvalidateCache("")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidConfig, err)
}

func TestResolver_InvalidateCache_NonExistentProfile(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	resolver := NewResolver(storage)

	// Should not error when invalidating non-existent profile
	err = resolver.InvalidateCache("nonexistent-profile")
	assert.NoError(t, err)
}

func TestResolver_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	_ = NewResolver(storage)

	// Manually save a card
	originalCard := &a2a.AgentCard{
		Name: "Cached Agent",
		URL:  "http://127.0.0.1:8082",
		Skills: []a2a.AgentSkill{
			{ID: "execute", Name: "Execute"},
		},
	}
	err = storage.SaveAgentCard("cached-profile", originalCard)
	require.NoError(t, err)

	// Retrieve it
	retrievedCard, err := storage.GetAgentCard("cached-profile")
	assert.NoError(t, err)
	assert.Equal(t, originalCard.Name, retrievedCard.Name)
	assert.Equal(t, originalCard.URL, retrievedCard.URL)
}
