package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil cache")
	}

	// Verify cache directory was created
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		t.Errorf("cache directory not created: %s", c.dir)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	c := newTestCache(t)

	// Test setting and getting a value
	c.Set("san francisco, ca", 37.7749, -122.4194)

	lat, lng, ok := c.Get("san francisco, ca")
	if !ok {
		t.Error("Get() returned ok=false for existing key")
	}
	if lat != 37.7749 || lng != -122.4194 {
		t.Errorf("Get() returned wrong coordinates: got (%.4f, %.4f), want (37.7749, -122.4194)", lat, lng)
	}
}

func TestCacheGetMissing(t *testing.T) {
	c := newTestCache(t)

	lat, lng, ok := c.Get("nonexistent address")
	if ok {
		t.Error("Get() returned ok=true for missing key")
	}
	if lat != 0 || lng != 0 {
		t.Errorf("Get() returned non-zero coordinates for missing key: (%.4f, %.4f)", lat, lng)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := newTestCache(t)

	// Set a value with timestamp in the past (expired)
	c.data["old address"] = CacheEntry{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Timestamp: time.Now().UTC().Add(-31 * 24 * time.Hour), // 31 days ago
	}

	lat, lng, ok := c.Get("old address")
	if ok {
		t.Error("Get() returned ok=true for expired entry")
	}
	if lat != 0 || lng != 0 {
		t.Errorf("Get() returned non-zero coordinates for expired entry: (%.4f, %.4f)", lat, lng)
	}

	// Verify entry was removed
	if _, exists := c.data["old address"]; exists {
		t.Error("expired entry was not removed from cache")
	}
}

func TestCacheEvict(t *testing.T) {
	c := newTestCache(t)

	c.Set("address1", 37.7749, -122.4194)
	c.Set("address2", 34.0522, -118.2437)

	c.Evict("address1")

	_, _, ok1 := c.Get("address1")
	_, _, ok2 := c.Get("address2")

	if ok1 {
		t.Error("Evict() failed to remove entry")
	}
	if !ok2 {
		t.Error("Evict() removed wrong entry")
	}
}

func TestCacheEvictExpired(t *testing.T) {
	c := newTestCache(t)

	// Add expired entry
	c.data["old1"] = CacheEntry{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Timestamp: time.Now().UTC().Add(-31 * 24 * time.Hour),
	}

	// Add fresh entry
	c.Set("fresh", 34.0522, -118.2437)

	count := c.EvictExpired()

	if count != 1 {
		t.Errorf("EvictExpired() returned %d, want 1", count)
	}

	_, _, okOld := c.Get("old1")
	_, _, okFresh := c.Get("fresh")

	if okOld {
		t.Error("expired entry was not evicted")
	}
	if !okFresh {
		t.Error("fresh entry was incorrectly evicted")
	}
}

func TestCacheClear(t *testing.T) {
	c := newTestCache(t)

	c.Set("address1", 37.7749, -122.4194)
	c.Set("address2", 34.0522, -118.2437)

	c.Clear()

	_, _, ok1 := c.Get("address1")
	_, _, ok2 := c.Get("address2")

	if ok1 || ok2 {
		t.Error("Clear() did not remove all entries")
	}
}

func TestCacheSaveAndLoad(t *testing.T) {
	// Create first cache and save data
	c1 := newTestCache(t)
	c1.Set("san francisco, ca", 37.7749, -122.4194)
	c1.Set("los angeles, ca", 34.0522, -118.2437)

	if err := c1.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Create second cache pointing to same directory
	c2 := &Cache{
		dir:        c1.dir,
		data:       make(map[string]CacheEntry),
		dirty:      false,
		defaultTTL: DefaultTTL,
	}

	if err := c2.load(); err != nil {
		t.Fatalf("load() error: %v", err)
	}

	// Verify data was loaded
	lat1, lng1, ok1 := c2.Get("san francisco, ca")
	lat2, lng2, ok2 := c2.Get("los angeles, ca")

	if !ok1 || lat1 != 37.7749 || lng1 != -122.4194 {
		t.Error("failed to load san francisco from cache")
	}
	if !ok2 || lat2 != 34.0522 || lng2 != -118.2437 {
		t.Error("failed to load los angeles from cache")
	}
}

func TestCacheSaveNotDirty(t *testing.T) {
	c := newTestCache(t)

	// Save without modifications should be no-op
	if err := c.Save(); err != nil {
		t.Errorf("Save() on clean cache returned error: %v", err)
	}
}

func TestCacheStats(t *testing.T) {
	c := newTestCache(t)

	// Add fresh entry
	c.Set("fresh", 37.7749, -122.4194)

	// Add expired entry
	c.data["old"] = CacheEntry{
		Latitude:  34.0522,
		Longitude: -118.2437,
		Timestamp: time.Now().UTC().Add(-31 * 24 * time.Hour),
	}

	total, expired := c.Stats()

	if total != 2 {
		t.Errorf("Stats() total = %d, want 2", total)
	}
	if expired != 1 {
		t.Errorf("Stats() expired = %d, want 1", expired)
	}
}

func TestCachePath(t *testing.T) {
	c := newTestCache(t)

	path := c.Path()
	if !strings.HasSuffix(path, CacheFileName) {
		t.Errorf("Path() = %s, expected to end with %s", path, CacheFileName)
	}
}

func TestCacheKeyNormalization(t *testing.T) {
	c := newTestCache(t)

	// Set with lowercase
	c.Set("san francisco", 37.7749, -122.4194)

	// Get with mixed case should NOT match (case-sensitive by design)
	_, _, ok := c.Get("San Francisco")
	if ok {
		t.Error("cache keys should be case-sensitive")
	}
}

// Helper to create a test cache with temporary directory
func newTestCache(t *testing.T) *Cache {
	t.Helper()

	// Create temporary directory for test cache
	tmpDir := t.TempDir()

	return &Cache{
		dir:        tmpDir,
		data:       make(map[string]CacheEntry),
		dirty:      false,
		defaultTTL: DefaultTTL,
	}
}

func TestCacheLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	c := &Cache{
		dir:        tmpDir,
		data:       make(map[string]CacheEntry),
		dirty:      false,
		defaultTTL: DefaultTTL,
	}

	// Should not error when cache file doesn't exist
	if err := c.load(); err != nil {
		t.Errorf("load() on non-existent cache returned error: %v", err)
	}

	if len(c.data) != 0 {
		t.Error("load() should start with empty cache when file doesn't exist")
	}
}

func TestCacheLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid JSON to cache file
	cachePath := filepath.Join(tmpDir, CacheFileName)
	if err := os.WriteFile(cachePath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test cache file: %v", err)
	}

	c := &Cache{
		dir:        tmpDir,
		data:       make(map[string]CacheEntry),
		dirty:      false,
		defaultTTL: DefaultTTL,
	}

	// Should error on invalid JSON
	if err := c.load(); err == nil {
		t.Error("load() should return error for invalid JSON")
	}
}

func TestCacheSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent", "subdir")

	c := &Cache{
		dir:        nonExistentDir,
		data:       make(map[string]CacheEntry),
		dirty:      true,
		defaultTTL: DefaultTTL,
	}

	c.Set("test", 37.7749, -122.4194)

	// This should create the directory
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}

	if err := c.Save(); err != nil {
		t.Errorf("Save() after creating directory error: %v", err)
	}
}
