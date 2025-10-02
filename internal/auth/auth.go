// Package auth provides authorization management for Dirvana projects.
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Auth manages authorized project paths
type Auth struct {
	path       string
	mu         sync.RWMutex
	authorized map[string]bool
}

// New creates a new auth manager
func New(path string) (*Auth, error) {
	a := &Auth{
		path:       path,
		authorized: make(map[string]bool),
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Load existing authorized paths if file exists
	if err := a.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return a, nil
}

// Allow adds a path to authorized list
func (a *Auth) Allow(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	normalized := normalizePath(path)
	a.authorized[normalized] = true
	return a.persist()
}

// IsAllowed checks if a path is authorized
func (a *Auth) IsAllowed(path string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	normalized := normalizePath(path)
	return a.authorized[normalized], nil
}

// Revoke removes a path from authorized list
func (a *Auth) Revoke(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	normalized := normalizePath(path)
	delete(a.authorized, normalized)
	return a.persist()
}

// List returns all authorized paths
func (a *Auth) List() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	paths := make([]string, 0, len(a.authorized))
	for path := range a.authorized {
		paths = append(paths, path)
	}
	return paths
}

// Clear removes all authorized paths
func (a *Auth) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.authorized = make(map[string]bool)
	return a.persist()
}

// load reads authorized paths from disk
func (a *Auth) load() error {
	data, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}

	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		return err
	}

	for _, path := range paths {
		a.authorized[normalizePath(path)] = true
	}

	return nil
}

// persist writes authorized paths to disk
func (a *Auth) persist() error {
	paths := make([]string, 0, len(a.authorized))
	for path := range a.authorized {
		paths = append(paths, path)
	}

	data, err := json.MarshalIndent(paths, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.path, data, 0600)
}

// normalizePath removes trailing slashes and cleans the path
func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.TrimSuffix(cleaned, "/")
}
