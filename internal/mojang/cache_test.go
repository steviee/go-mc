package mojang

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		ttl      time.Duration
		wantCap  int
		wantTTL  time.Duration
	}{
		{
			name:     "default values",
			capacity: 0,
			ttl:      0,
			wantCap:  DefaultCacheSize,
			wantTTL:  DefaultCacheTTL,
		},
		{
			name:     "custom values",
			capacity: 500,
			ttl:      1 * time.Hour,
			wantCap:  500,
			wantTTL:  1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(tt.capacity, tt.ttl)

			assert.NotNil(t, cache)
			assert.Equal(t, tt.wantCap, cache.capacity)
			assert.Equal(t, tt.wantTTL, cache.ttl)
			assert.Equal(t, 0, cache.Len())
		})
	}
}

func TestCache_SetGet(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	entry := CacheEntry{
		Profile: &Profile{
			UUID:     "069a79f4-44e9-4726-a5be-fca90e38aaf5",
			Username: "Notch",
		},
	}

	// Set entry
	cache.Set("Notch", entry)

	// Get entry
	got := cache.Get("Notch")
	assert.NotNil(t, got)
	assert.Equal(t, entry.Profile.UUID, got.Profile.UUID)
	assert.Equal(t, entry.Profile.Username, got.Profile.Username)
	assert.Equal(t, 1, cache.Len())
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	got := cache.Get("NonExistent")
	assert.Nil(t, got)
}

func TestCache_CaseInsensitive(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	entry := CacheEntry{
		Profile: &Profile{
			UUID:     "069a79f4-44e9-4726-a5be-fca90e38aaf5",
			Username: "Notch",
		},
	}

	// Set with uppercase
	cache.Set("NOTCH", entry)

	// Get with lowercase
	got := cache.Get("notch")
	assert.NotNil(t, got)
	assert.Equal(t, entry.Profile.UUID, got.Profile.UUID)

	// Get with mixed case
	got = cache.Get("NoTcH")
	assert.NotNil(t, got)
	assert.Equal(t, entry.Profile.UUID, got.Profile.UUID)
}

func TestCache_Update(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	entry1 := CacheEntry{
		Profile: &Profile{
			UUID:     "uuid1",
			Username: "User1",
		},
	}

	entry2 := CacheEntry{
		Profile: &Profile{
			UUID:     "uuid2",
			Username: "User2",
		},
	}

	// Set initial entry
	cache.Set("User", entry1)
	assert.Equal(t, 1, cache.Len())

	// Update entry
	cache.Set("User", entry2)
	assert.Equal(t, 1, cache.Len())

	// Verify updated
	got := cache.Get("User")
	assert.NotNil(t, got)
	assert.Equal(t, entry2.Profile.UUID, got.Profile.UUID)
}

func TestCache_Eviction(t *testing.T) {
	cache := NewCache(3, 1*time.Hour)

	// Add 3 entries (at capacity)
	cache.Set("user1", CacheEntry{Profile: &Profile{UUID: "uuid1", Username: "user1"}})
	cache.Set("user2", CacheEntry{Profile: &Profile{UUID: "uuid2", Username: "user2"}})
	cache.Set("user3", CacheEntry{Profile: &Profile{UUID: "uuid3", Username: "user3"}})
	assert.Equal(t, 3, cache.Len())

	// Add 4th entry - should evict oldest (user1)
	cache.Set("user4", CacheEntry{Profile: &Profile{UUID: "uuid4", Username: "user4"}})
	assert.Equal(t, 3, cache.Len())

	// user1 should be evicted
	assert.Nil(t, cache.Get("user1"))

	// Others should still exist
	assert.NotNil(t, cache.Get("user2"))
	assert.NotNil(t, cache.Get("user3"))
	assert.NotNil(t, cache.Get("user4"))
}

func TestCache_LRU(t *testing.T) {
	cache := NewCache(3, 1*time.Hour)

	// Add 3 entries
	cache.Set("user1", CacheEntry{Profile: &Profile{UUID: "uuid1", Username: "user1"}})
	cache.Set("user2", CacheEntry{Profile: &Profile{UUID: "uuid2", Username: "user2"}})
	cache.Set("user3", CacheEntry{Profile: &Profile{UUID: "uuid3", Username: "user3"}})

	// Access user1 (makes it most recently used)
	cache.Get("user1")

	// Add user4 - should evict user2 (least recently used)
	cache.Set("user4", CacheEntry{Profile: &Profile{UUID: "uuid4", Username: "user4"}})

	// user2 should be evicted
	assert.Nil(t, cache.Get("user2"))

	// user1, user3, user4 should still exist
	assert.NotNil(t, cache.Get("user1"))
	assert.NotNil(t, cache.Get("user3"))
	assert.NotNil(t, cache.Get("user4"))
}

func TestCache_TTL(t *testing.T) {
	cache := NewCache(10, 50*time.Millisecond)

	entry := CacheEntry{
		Profile: &Profile{
			UUID:     "069a79f4-44e9-4726-a5be-fca90e38aaf5",
			Username: "Notch",
		},
	}

	// Set entry
	cache.Set("Notch", entry)

	// Immediate get should work
	got := cache.Get("Notch")
	assert.NotNil(t, got)

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Get should return nil (expired)
	got = cache.Get("Notch")
	assert.Nil(t, got)
}

func TestCache_NegativeResult(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	// Cache a negative result (username not found)
	entry := CacheEntry{
		Profile:  nil,
		NotFound: true,
	}

	cache.Set("NonExistent", entry)

	// Get should return the entry
	got := cache.Get("NonExistent")
	assert.NotNil(t, got)
	assert.Nil(t, got.Profile)
	assert.True(t, got.NotFound)
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(10, 1*time.Hour)

	// Add multiple entries
	cache.Set("user1", CacheEntry{Profile: &Profile{UUID: "uuid1", Username: "user1"}})
	cache.Set("user2", CacheEntry{Profile: &Profile{UUID: "uuid2", Username: "user2"}})
	cache.Set("user3", CacheEntry{Profile: &Profile{UUID: "uuid3", Username: "user3"}})
	assert.Equal(t, 3, cache.Len())

	// Clear cache
	cache.Clear()

	// Cache should be empty
	assert.Equal(t, 0, cache.Len())
	assert.Nil(t, cache.Get("user1"))
	assert.Nil(t, cache.Get("user2"))
	assert.Nil(t, cache.Get("user3"))
}

func TestCache_Concurrent(t *testing.T) {
	cache := NewCache(100, 1*time.Hour)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				username := "user" + string(rune('0'+n))
				entry := CacheEntry{
					Profile: &Profile{
						UUID:     "uuid-" + string(rune('0'+n)),
						Username: username,
					},
				}
				cache.Set(username, entry)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				username := "user" + string(rune('0'+n))
				cache.Get(username)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panics or race conditions should occur
	assert.True(t, cache.Len() > 0)
}
