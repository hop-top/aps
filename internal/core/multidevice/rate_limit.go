package multidevice

import (
	"sync"
	"time"
)

// tokenBucket implements a token bucket for rate limiting.
type tokenBucket struct {
	tokens    float64
	max       float64
	rate      float64 // tokens per second
	lastCheck time.Time
}

// refill adds tokens based on elapsed time since last check.
func (tb *tokenBucket) refill(now time.Time) {
	elapsed := now.Sub(tb.lastCheck).Seconds()
	if elapsed <= 0 {
		return
	}
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.max {
		tb.tokens = tb.max
	}
	tb.lastCheck = now
}

// RateLimiter implements per-device token bucket rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket // key: deviceID:workspaceID
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*tokenBucket),
	}
}

// rateLimitKey builds a composite key for the rate limiter.
func rateLimitKey(deviceID, workspaceID string) string {
	return deviceID + ":" + workspaceID
}

// getBucket returns or creates a token bucket for the given key.
// The limit parameter is requests per minute.
func (rl *RateLimiter) getBucket(key string, limit int) *tokenBucket {
	bucket, exists := rl.buckets[key]
	if !exists {
		now := time.Now()
		maxTokens := float64(limit)
		ratePerSec := float64(limit) / 60.0
		bucket = &tokenBucket{
			tokens:    maxTokens,
			max:       maxTokens,
			rate:      ratePerSec,
			lastCheck: now,
		}
		rl.buckets[key] = bucket
	}
	return bucket
}

// Check checks if the device is within rate limits without consuming a token.
// Returns (allowed, remaining tokens, retryAfter duration).
func (rl *RateLimiter) Check(deviceID, workspaceID string, limit int) (bool, int, time.Duration) {
	if limit <= 0 {
		return true, 0, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := rateLimitKey(deviceID, workspaceID)
	bucket := rl.getBucket(key, limit)

	now := time.Now()
	bucket.refill(now)

	remaining := int(bucket.tokens)
	if remaining < 1 {
		// Calculate how long until a token is available.
		deficit := 1.0 - bucket.tokens
		retryAfter := time.Duration(deficit/bucket.rate*1000) * time.Millisecond
		return false, 0, retryAfter
	}

	return true, remaining, 0
}

// Consume attempts to consume a token from the bucket.
// Returns (allowed, remaining tokens, retryAfter duration).
func (rl *RateLimiter) Consume(deviceID, workspaceID string, limit int) (bool, int, time.Duration) {
	if limit <= 0 {
		return true, 0, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := rateLimitKey(deviceID, workspaceID)
	bucket := rl.getBucket(key, limit)

	now := time.Now()
	bucket.refill(now)

	if bucket.tokens < 1 {
		deficit := 1.0 - bucket.tokens
		retryAfter := time.Duration(deficit/bucket.rate*1000) * time.Millisecond
		return false, 0, retryAfter
	}

	bucket.tokens--
	remaining := int(bucket.tokens)

	return true, remaining, 0
}
