package a2a

import (
	"testing"

	"github.com/stretchr/testify/assert"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

func TestNewCache(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	assert.NotNil(t, cache)
	assert.Equal(t, 0, cache.Size())
}

func TestCache_GetMiss(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	card, exists := cache.Get("test-profile")

	assert.False(t, exists)
	assert.Nil(t, card)
}

func TestCache_GetHit(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	testCard := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
	}

	cache.Set("test-profile", testCard)

	card, exists := cache.Get("test-profile")

	assert.True(t, exists)
	assert.Equal(t, testCard.Name, card.Name)
}

func TestCache_Invalidate(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	testCard := &a2a.AgentCard{
		Name: "Test Agent",
		URL:  "http://127.0.0.1:8081",
	}

	cache.Set("test-profile", testCard)
	cache.Invalidate("test-profile")

	card, exists := cache.Get("test-profile")

	assert.False(t, exists)
	assert.Nil(t, card)
}

func TestCache_Clear(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	cache.Set("profile1", &a2a.AgentCard{Name: "Agent 1", URL: "http://127.0.0.1:8081"})
	cache.Set("profile2", &a2a.AgentCard{Name: "Agent 2", URL: "http://127.0.0.1:8082"})

	assert.Equal(t, 2, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())
}

func TestCache_SetTTL(t *testing.T) {
	config := DefaultStorageConfig()
	cache := NewCache(config)

	cache.SetTTL(3600)

	cache.Set("test", &a2a.AgentCard{Name: "Test", URL: "http://127.0.0.1:8081"})

	_, exists := cache.Get("test")
	assert.True(t, exists)
}
