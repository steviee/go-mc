package mojang

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultCacheSize is the default number of entries to cache.
	DefaultCacheSize = 1000

	// DefaultCacheTTL is the default time-to-live for cache entries.
	DefaultCacheTTL = 24 * time.Hour
)

// Cache is a thread-safe LRU cache for UUID lookups.
type Cache struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	lru      *list.List
}

type cacheItem struct {
	key   string
	value CacheEntry
}

// NewCache creates a new LRU cache.
func NewCache(capacity int, ttl time.Duration) *Cache {
	if capacity <= 0 {
		capacity = DefaultCacheSize
	}
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	return &Cache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a value from the cache.
// Returns nil if the key doesn't exist or has expired.
func (c *Cache) Get(username string) *CacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := strings.ToLower(username)
	elem, exists := c.items[key]
	if !exists {
		return nil
	}

	item := elem.Value.(*cacheItem)

	// Check if entry has expired
	if time.Since(item.value.Timestamp) > c.ttl {
		return nil
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)

	return &item.value
}

// Set adds or updates a value in the cache.
func (c *Cache) Set(username string, entry CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := strings.ToLower(username)
	entry.Timestamp = time.Now()

	// Check if key already exists
	if elem, exists := c.items[key]; exists {
		// Update existing entry
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheItem).value = entry
		return
	}

	// Add new entry
	item := &cacheItem{
		key:   key,
		value: entry,
	}
	elem := c.lru.PushFront(item)
	c.items[key] = elem

	// Evict oldest entry if capacity exceeded
	if c.lru.Len() > c.capacity {
		c.evictOldest()
	}
}

// evictOldest removes the least recently used entry.
// Must be called with lock held.
func (c *Cache) evictOldest() {
	elem := c.lru.Back()
	if elem == nil {
		return
	}

	c.lru.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
}

// Len returns the number of entries in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lru.Len()
}
