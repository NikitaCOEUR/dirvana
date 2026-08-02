package completion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/fsutil"
)

// CacheEntry stores completer type with timestamp for TTL
type CacheEntry struct {
	CompleterType string    `json:"completer_type"`
	Timestamp     time.Time `json:"timestamp"`
}

// DetectionCache persists which completer type works for each tool
// Only successful completions are cached, with a TTL to allow re-detection
type DetectionCache struct {
	mu       sync.RWMutex
	path     string
	cache    map[string]CacheEntry
	modified bool
	ttl      time.Duration
}

// NewDetectionCache creates or loads a detection cache with a 24h TTL.
// A missing, unreadable or corrupt cache file yields an empty cache —
// detection results are always recomputable, so this never fails.
func NewDetectionCache(cachePath string) *DetectionCache {
	c := &DetectionCache{
		path:  cachePath,
		cache: make(map[string]CacheEntry),
		ttl:   24 * time.Hour,
	}

	if err := c.load(); err != nil {
		// Reset in case of a partially decoded corrupt file
		c.cache = make(map[string]CacheEntry)
	}

	return c
}

// Get returns the completer type for a tool, or empty string if not cached or expired
func (c *DetectionCache) Get(tool string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[tool]
	if !ok {
		return ""
	}

	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.ttl {
		return ""
	}

	return entry.CompleterType
}

// Set stores the completer type for a tool with current timestamp
func (c *DetectionCache) Set(tool string, completerType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := CacheEntry{
		CompleterType: completerType,
		Timestamp:     time.Now(),
	}

	// Only mark as modified if the completer type actually changed
	if existing, ok := c.cache[tool]; !ok || existing.CompleterType != completerType {
		c.cache[tool] = entry
		c.modified = true
	}
}

// Save persists the cache to disk if it was modified
func (c *DetectionCache) Save() error {
	c.mu.RLock()
	if !c.modified {
		c.mu.RUnlock()
		return nil // Nothing to save
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, fsutil.StateDirPerm); err != nil {
		return err
	}

	// Marshal cache to JSON
	data, err := json.Marshal(c.cache)
	if err != nil {
		return err
	}

	// Write to file
	if err := fsutil.AtomicWrite(c.path, data, fsutil.StateFilePerm); err != nil {
		return err
	}

	c.modified = false
	return nil
}

// load reads the cache from disk
func (c *DetectionCache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.cache)
}
