// Package auth provides authorization management for Dirvana projects.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/fsutil"
)

const currentAuthVersion = 2

// GetAuth returns the DirAuth structure for a given directory path
func (a *Auth) GetAuth(path string) *DirAuth {
	a.mu.RLock()
	defer a.mu.RUnlock()
	auth, _ := a.lookup(path)
	return auth
}

// lookup finds the entry for a directory under the name given or under the one
// every symlink along it resolves to, and returns the key it was filed under -
// or the key a new entry should take.
//
// An authorization identifies a directory, not one of the names leading to it:
// a shell keeps the logical path in $PWD and os.Getwd honours it, so the same
// project reaches dirvana under whichever name the user typed. Looking the
// literal name up first keeps entries written by earlier versions working.
//
// The caller must hold the lock.
func (a *Auth) lookup(path string) (*DirAuth, string) {
	normalized := fsutil.NormalizePath(path)
	if auth := a.authorized[normalized]; auth != nil {
		return auth, normalized
	}

	resolved := fsutil.ResolvePath(path)
	if resolved != normalized {
		if auth := a.authorized[resolved]; auth != nil {
			return auth, resolved
		}
	}

	// Nothing yet: a new entry is filed under the resolved name, so it holds
	// wherever the directory is reached from - and stops holding if the
	// symlink that led to it is later pointed somewhere else
	return nil, resolved
}

// RequiresShellApproval returns true if shell command approval is needed for the directory
func (a *Auth) RequiresShellApproval(dir string, shellCmds map[string]string) bool {
	if len(shellCmds) == 0 {
		return false
	}
	auth := a.GetAuth(dir)
	if auth == nil || !auth.Allowed {
		return false // Directory authorization required first
	}
	currentHash := hashShellCommands(shellCmds)
	return auth.ShellCommandsHash == "" || auth.ShellCommandsHash != currentHash
}

// hashShellCommands computes a deterministic hash of shell commands
func hashShellCommands(cmds map[string]string) string {
	keys := make([]string, 0, len(cmds))
	for k := range cmds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		// Write to hash (error can be safely ignored as hash.Hash.Write never fails)
		_, _ = fmt.Fprintf(h, "%s=%s\n", k, cmds[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ApproveShellCommands saves shell command approval for a directory
func (a *Auth) ApproveShellCommands(dir string, shellCmds map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	auth, _ := a.lookup(dir)
	if auth == nil {
		return fmt.Errorf("directory not authorized")
	}
	auth.ShellCommandsHash = hashShellCommands(shellCmds)
	auth.ShellApprovedAt = time.Now()
	return a.persist()
}

// DirAuth stores the authorization state of a directory, including dynamic shell command approval
type DirAuth struct {
	Allowed           bool      `json:"allowed"`
	AllowedAt         time.Time `json:"allowed_at,omitempty"`
	ShellCommandsHash string    `json:"shell_commands_hash,omitempty"`
	ShellApprovedAt   time.Time `json:"shell_approved_at,omitempty"`
}

// File represents the v2 auth file structure with version metadata
type File struct {
	Version     int                 `json:"_version"`
	Directories map[string]*DirAuth `json:"directories"`
}

// Auth manages project directory authorization and shell command approval
type Auth struct {
	path       string
	mu         sync.RWMutex
	authorized map[string]*DirAuth
}

// New creates or loads an Auth instance from the given auth file path
func New(path string) (*Auth, error) {
	a := &Auth{
		path:       path,
		authorized: make(map[string]*DirAuth),
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), fsutil.StateDirPerm); err != nil {
		return nil, err
	}

	if err := a.load(); err != nil && !os.IsNotExist(err) {
		// Start with empty state on errors
		a.authorized = make(map[string]*DirAuth)
	}

	return a, nil
}

// Allow adds a directory to the authorized list
func (a *Auth) Allow(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	existing, key := a.lookup(path)

	// Check if already allowed - idempotent operation
	if existing != nil && existing.Allowed {
		return nil
	}

	now := time.Now()
	if existing == nil {
		a.authorized[key] = &DirAuth{
			Allowed:   true,
			AllowedAt: now,
		}
	} else {
		existing.Allowed = true
		existing.AllowedAt = now
	}
	return a.persist()
}

// IsAllowed checks if a directory is authorized
func (a *Auth) IsAllowed(path string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	auth, _ := a.lookup(path)
	return auth != nil && auth.Allowed, nil
}

// Revoke removes a directory from the authorized list
func (a *Auth) Revoke(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Both spellings go: an entry left under the other one would keep the
	// directory authorized after the user asked for that to stop
	delete(a.authorized, fsutil.NormalizePath(path))
	delete(a.authorized, fsutil.ResolvePath(path))
	return a.persist()
}

// List returns all authorized directories
func (a *Auth) List() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	paths := make([]string, 0, len(a.authorized))
	for path := range a.authorized {
		paths = append(paths, path)
	}
	return paths
}

// Clear removes all authorized directories
func (a *Auth) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.authorized = make(map[string]*DirAuth)
	return a.persist()
}

// load reads authorized directories from disk
func (a *Auth) load() error {
	data, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}

	var authFile File
	if err := json.Unmarshal(data, &authFile); err != nil {
		return fmt.Errorf("invalid auth file: %w", err)
	}

	if authFile.Version != currentAuthVersion {
		return fmt.Errorf("unsupported auth file version: %d", authFile.Version)
	}

	a.authorized = make(map[string]*DirAuth)
	for path, auth := range authFile.Directories {
		if auth != nil {
			a.authorized[fsutil.NormalizePath(path)] = auth
		}
	}

	return nil
}

// persist writes authorized directories to disk
func (a *Auth) persist() error {
	authFile := File{
		Version:     currentAuthVersion,
		Directories: a.authorized,
	}

	data, err := json.MarshalIndent(authFile, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.AtomicWrite(a.path, data, fsutil.StateFilePerm)
}
