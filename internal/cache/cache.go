package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultTTL is the default cache expiration time (30 days)
	DefaultTTL = 30 * 24 * time.Hour
	// CacheFileName is the name of the cache file
	CacheFileName = "geocode_cache.json"
)

// Cache provides persistent storage for geocoded address results
type Cache struct {
	dir        string
	data       map[string]CacheEntry
	dirty      bool
	defaultTTL time.Duration
}

// CacheEntry stores a single cached geocode result
type CacheEntry struct {
	Latitude  float64   `json:"lat"`
	Longitude float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new Cache instance using the OS default cache directory
func New() (*Cache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get user cache dir: %w", err)
	}

	appCacheDir := filepath.Join(cacheDir, "ams")
	if err := os.MkdirAll(appCacheDir, 0750); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	c := &Cache{
		dir:        appCacheDir,
		data:       make(map[string]CacheEntry),
		dirty:      false,
		defaultTTL: DefaultTTL,
	}

	// Load existing cache if present
	if err := c.load(); err != nil {
		// Non-fatal: start with empty cache
		c.data = make(map[string]CacheEntry)
	}

	return c, nil
}

// Get retrieves a cached entry if it exists and is not expired
func (c *Cache) Get(key string) (lat, lng float64, ok bool) {
	entry, exists := c.data[key]
	if !exists {
		return 0, 0, false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > c.defaultTTL {
		delete(c.data, key)
		c.dirty = true
		return 0, 0, false
	}

	return entry.Latitude, entry.Longitude, true
}

// Set stores a geocode result in the cache
func (c *Cache) Set(key string, lat, lng float64) {
	c.data[key] = CacheEntry{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: time.Now().UTC(),
	}
	c.dirty = true
}

// Evict removes a specific entry from the cache
func (c *Cache) Evict(key string) {
	delete(c.data, key)
	c.dirty = true
}

// EvictExpired removes all expired entries
func (c *Cache) EvictExpired() int {
	count := 0
	now := time.Now().UTC()
	for key, entry := range c.data {
		if now.Sub(entry.Timestamp) > c.defaultTTL {
			delete(c.data, key)
			count++
		}
	}
	if count > 0 {
		c.dirty = true
	}
	return count
}

// Clear removes all entries
func (c *Cache) Clear() {
	c.data = make(map[string]CacheEntry)
	c.dirty = true
}

// Save persists the cache to disk
func (c *Cache) Save() error {
	if !c.dirty {
		return nil
	}

	path := filepath.Join(c.dir, CacheFileName)
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	// Write atomically to avoid corruption
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil { // #nosec G306 - cache file needs user-only access
		return fmt.Errorf("write cache temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename cache file: %w", err)
	}

	c.dirty = false
	return nil
}

// load reads the cache from disk
func (c *Cache) load() error {
	path := filepath.Join(c.dir, CacheFileName)
	data, err := os.ReadFile(path) // #nosec G304 - path is constructed from trusted cache directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache yet, that's fine
		}
		return fmt.Errorf("read cache file: %w", err)
	}

	var entries map[string]CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal cache: %w", err)
	}

	c.data = entries
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() (total, expired int) {
	now := time.Now().UTC()
	for _, entry := range c.data {
		total++
		if now.Sub(entry.Timestamp) > c.defaultTTL {
			expired++
		}
	}
	return total, expired
}

// Path returns the cache file path
func (c *Cache) Path() string {
	return filepath.Join(c.dir, CacheFileName)
}
