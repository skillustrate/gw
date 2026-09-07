package cache

import (
	"strings"
	"sync"
	"time"
)

// Entry holds a cached value and the timestamp when it was observed.
type Entry[T any] struct {
	Value      T
	ObservedAt time.Time
}

// Cache provides a thread-safe in-process TTL cache for read workflows.
type Cache[T any] struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]Entry[T]
}

// New instantiates a new Cache with the given TTL duration.
func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		ttl:     ttl,
		entries: make(map[string]Entry[T]),
	}
}

// Get returns the cached value if present and unexpired.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		var zero T
		return zero, false
	}

	if c.ttl > 0 && time.Since(entry.ObservedAt) > c.ttl {
		var zero T
		return zero, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with the given observation timestamp.
func (c *Cache[T]) Set(key string, val T, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry[T]{
		Value:      val,
		ObservedAt: now,
	}
}

// Invalidate removes a cached entry.
func (c *Cache[T]) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// InvalidatePrefix removes all cached entries whose keys start with prefix.
func (c *Cache[T]) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
}

// Clear removes all cached entries.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]Entry[T])
}