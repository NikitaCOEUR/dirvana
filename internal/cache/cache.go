// Package cache provides persistent and in-memory caching for Dirvana configuration.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a cached configuration entry
type Entry struct {
	Path      string    `json:"path"`
	Hash      string    `json:"hash"`
	ShellCode string    `json:"shell_code"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	LocalOnly bool      `json:"local_only"`
	// Track what was defined for cleanup
	Aliases   []string `json:"aliases,omitempty"`
	Functions []string `json:"functions,omitempty"`
	EnvVars   []string `json:"env_vars,omitempty"`
	// Map of alias/function name to actual command (for dirvana exec)
	CommandMap map[string]string `json:"command_map,omitempty"`
	// Map of alias name to completion command (overrides CommandMap for completion)
	// Example: k -> kubectl (when k executes kubecolor but completes with kubectl)
	CompletionMap map[string]string `json:"completion_map,omitempty"`

	// NEW: Merged configuration cache for fast completion/exec
	// This stores the merged result after applying hierarchy, auth, global config, etc.
	MergedCommandMap    map[string]string `json:"merged_command_map,omitempty"`
	MergedCompletionMap map[string]string `json:"merged_completion_map,omitempty"`
	// Hash of the full hierarchy (all config files that contributed to the merge)
	// Format: "hash1:hash2:hash3:..." from root to leaf
	HierarchyHash string `json:"hierarchy_hash,omitempty"`
	// Paths of all configs in the hierarchy that contributed to this merge
	HierarchyPaths []string `json:"hierarchy_paths,omitempty"`
}

// Cache manages persistent and in-memory cache
type Cache struct {
	path    string
	mu      sync.RWMutex
	entries map[string]*Entry
}

// New creates a new cache instance
func New(path string) (*Cache, error) {
	c := &Cache{
		path:    path,
		entries: make(map[string]*Entry),
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Load existing cache if it exists
	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return c, nil
}

// Get retrieves an entry from cache
func (c *Cache) Get(path string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[path]
	return entry, found
}

// Set stores an entry in cache and persists it
func (c *Cache) Set(entry *Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[entry.Path] = entry
	return c.persist()
}

// Delete removes an entry from cache
func (c *Cache) Delete(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, path)
	return c.persist()
}

// Clear removes all entries from cache
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*Entry)
	return c.persist()
}

// ClearHierarchy removes cache entries for the given directory and its hierarchy
func (c *Cache) ClearHierarchy(dir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Normalize path
	dir = filepath.Clean(dir)

	// Track which entries to delete
	toDelete := []string{}

	for path := range c.entries {
		// Delete if path is the directory or is within its hierarchy
		cleanPath := filepath.Clean(path)
		if cleanPath == dir || isParentOf(dir, cleanPath) || isParentOf(cleanPath, dir) {
			toDelete = append(toDelete, path)
		}
	}

	// Delete the entries
	for _, path := range toDelete {
		delete(c.entries, path)
	}

	return c.persist()
}

// DeleteWithSubdirs removes cache entries for the given directory and all its subdirectories
// This is useful when revoking authorization - we want to invalidate the revoked directory
// and all subdirectories, but not parent directories
func (c *Cache) DeleteWithSubdirs(dir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Normalize path
	dir = filepath.Clean(dir)

	// Track which entries to delete
	toDelete := []string{}

	for path := range c.entries {
		// Delete if path is the directory or is a subdirectory
		cleanPath := filepath.Clean(path)
		if cleanPath == dir || isParentOf(dir, cleanPath) {
			toDelete = append(toDelete, path)
		}
	}

	// Delete the entries
	for _, path := range toDelete {
		delete(c.entries, path)
	}

	return c.persist()
}

// isParentOf checks if parent is a parent directory of child
func isParentOf(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// If the relative path doesn't start with "..", child is under parent
	return len(rel) > 0 && rel[0] != '.' && rel[:2] != ".."
}

// IsValid checks if cached entry is valid for given hash and version
func (c *Cache) IsValid(path, hash, version string) bool {
	entry, found := c.Get(path)
	if !found {
		return false
	}

	return entry.Hash == hash && entry.Version == version
}

// load reads cache from disk
func (c *Cache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	var entries map[string]*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	c.entries = entries
	return nil
}

// persist writes cache to disk
func (c *Cache) persist() error {
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}
