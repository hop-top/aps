package a2a

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

// Cache provides Agent Card caching functionality
type Cache struct {
	mu         sync.RWMutex
	cards      map[string]*CachedCard
	config     *StorageConfig
	ttlSeconds int
}

// CachedCard represents a cached Agent Card with metadata
type CachedCard struct {
	Card     *a2a.AgentCard
	FilePath string
	CachedAt time.Time
	ExpireAt time.Time
}

// NewCache creates a new Agent Card cache
func NewCache(config *StorageConfig) *Cache {
	return &Cache{
		cards:      make(map[string]*CachedCard),
		config:     config,
		ttlSeconds: 3600,
	}
}

// Get retrieves a cached Agent Card for a profile
func (c *Cache) Get(profileID string) (*a2a.AgentCard, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.cards[profileID]
	if !exists {
		return nil, false
	}

	if time.Now().After(cached.ExpireAt) {
		delete(c.cards, profileID)
		return nil, false
	}

	return cached.Card, true
}

// Set caches an Agent Card for a profile
func (c *Cache) Set(profileID string, card *a2a.AgentCard) error {
	now := time.Now()

	cached := &CachedCard{
		Card:     card,
		CachedAt: now,
		ExpireAt: now.Add(time.Duration(c.ttlSeconds) * time.Second),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards[profileID] = cached

	return c.persist(profileID, card)
}

// Invalidate removes a cached Agent Card for a profile
func (c *Cache) Invalidate(profileID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cards, profileID)
}

// Clear removes all cached Agent Cards
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards = make(map[string]*CachedCard)
}

// LoadFromDisk loads an Agent Card from disk and caches it
func (c *Cache) LoadFromDisk(profileID string) (*a2a.AgentCard, error) {
	cardPath := filepath.Join(c.config.AgentCardsPath, profileID+".json")

	data, err := os.ReadFile(cardPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, ErrStorageFailed("read cached card", err)
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, ErrStorageFailed("unmarshal cached card", err)
	}

	cached := &CachedCard{
		Card:     &card,
		FilePath: cardPath,
		CachedAt: time.Now(),
		ExpireAt: time.Now().Add(time.Duration(c.ttlSeconds) * time.Second),
	}

	c.mu.Lock()
	c.cards[profileID] = cached
	c.mu.Unlock()

	return &card, nil
}

// persist saves an Agent Card to disk
func (c *Cache) persist(profileID string, card *a2a.AgentCard) error {
	cardsPath := c.config.AgentCardsPath
	if cardsPath == "" {
		cardsPath = filepath.Join(c.config.BasePath, "agent-cards")
	}

	if err := os.MkdirAll(cardsPath, 0700); err != nil {
		return ErrStorageFailed("create cache directory", err)
	}

	cardPath := filepath.Join(cardsPath, profileID+".json")

	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return ErrStorageFailed("marshal card", err)
	}

	if err := os.WriteFile(cardPath, data, 0600); err != nil {
		return ErrStorageFailed("write card", err)
	}

	return nil
}

// Delete removes a cached Agent Card from disk and memory
func (c *Cache) Delete(profileID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cards, profileID)

	cardPath := filepath.Join(c.config.AgentCardsPath, profileID+".json")
	if err := os.Remove(cardPath); err != nil && !os.IsNotExist(err) {
		return ErrStorageFailed("delete cached card", err)
	}

	return nil
}

// SetTTL sets the cache time-to-live in seconds
func (c *Cache) SetTTL(seconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttlSeconds = seconds
}

// Size returns the number of cached Agent Cards
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cards)
}
