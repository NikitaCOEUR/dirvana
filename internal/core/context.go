package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/trace"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
)

// cacheValidationTTL is the time during which we trust the cache without
// revalidating. This avoids expensive hash recalculation on every
// completion keystroke and every aliased command execution.
const cacheValidationTTL = 2 * time.Second

// FunctionPrefix marks function entries in the command map so exec knows
// to run them as shell functions instead of commands.
const FunctionPrefix = "__dirvana_function__"

// Context is the merged dirvana configuration applicable to a directory:
// the full alias specs (conditions and completion overrides included) and
// the function bodies.
type Context struct {
	Aliases   map[string]config.AliasConfig
	Functions map[string]string

	commandMap    map[string]string
	completionMap map[string]string
}

// NewContext builds a Context, normalizing nil maps to empty ones
func NewContext(aliases map[string]config.AliasConfig, functions map[string]string) *Context {
	if aliases == nil {
		aliases = make(map[string]config.AliasConfig)
	}
	if functions == nil {
		functions = make(map[string]string)
	}
	return &Context{Aliases: aliases, Functions: functions}
}

// Empty reports whether no alias or function applies to the directory
func (c *Context) Empty() bool {
	return len(c.Aliases) == 0 && len(c.Functions) == 0
}

// CommandMap maps alias/function names to the command exec should run.
// Functions are marked with FunctionPrefix. Derived lazily and memoized.
func (c *Context) CommandMap() map[string]string {
	if c.commandMap == nil {
		m := make(map[string]string, len(c.Aliases)+len(c.Functions))
		for name, aliasConf := range c.Aliases {
			m[name] = aliasConf.Command
		}
		for name := range c.Functions {
			m[name] = FunctionPrefix + name
		}
		c.commandMap = m
	}
	return c.commandMap
}

// CompletionMap maps alias names to the command used for completion:
// the explicit completion override when set, the alias command otherwise,
// nothing when completion is disabled. Derived lazily and memoized.
func (c *Context) CompletionMap() map[string]string {
	if c.completionMap == nil {
		m := make(map[string]string, len(c.Aliases))
		for name, aliasConf := range c.Aliases {
			switch v := aliasConf.Completion.(type) {
			case string:
				if v != "" {
					m[name] = v
				} else {
					m[name] = aliasConf.Command
				}
			case bool:
				if v {
					m[name] = aliasConf.Command
				}
				// completion: false -> no entry
			default:
				// No completion specified: complete with the command itself
				m[name] = aliasConf.Command
			}
		}
		c.completionMap = m
	}
	return c.completionMap
}

// Engine resolves the merged context for directories, backed by the
// persistent cache so the exec and completion hot paths avoid re-reading
// the whole config hierarchy on every invocation.
type Engine struct {
	CachePath string
	AuthPath  string
}

// NewEngine creates an Engine using the given state file paths
func NewEngine(cachePath, authPath string) *Engine {
	return &Engine{CachePath: cachePath, AuthPath: authPath}
}

// Load returns the merged context for dir.
//
// Resolution is three-tier:
//  1. fast path: a cache entry newer than cacheValidationTTL is trusted
//     without touching auth or config files (zero file I/O beyond the
//     cache itself);
//  2. validated path: an older entry is kept if the hierarchy hash still
//     matches the config files on disk;
//  3. slow path: the hierarchy is fully re-loaded and the refreshed
//     snapshot is written back so the next call hits the fast path.
func (e *Engine) Load(dir string) (*Context, error) {
	ctx := context.Background()
	defer trace.Region(ctx, "core.Load")()

	var cacheStore *cache.Cache
	var err error
	trace.WithRegion(ctx, "cache.New", func() {
		cacheStore, err = cache.New(e.CachePath)
	})
	if err != nil {
		return nil, err
	}

	// FAST PATH: check cache TTL before loading heavy components (auth, config)
	if entry, found := cacheStore.Get(dir); found && entryValidFast(entry) {
		trace.Log(ctx, "cache", "hit-fast")
		return NewContext(entry.MergedAliases, entry.MergedFunctions), nil
	}

	// SLOW PATH: cache miss or TTL expired - load auth and config
	authMgr, err := auth.New(e.AuthPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}
	loader := config.New()

	if entry, found := cacheStore.Get(dir); found {
		var valid bool
		trace.WithRegion(ctx, "validateEntry", func() {
			valid = validateEntry(entry, dir, loader, authMgr)
		})
		if valid {
			trace.Log(ctx, "cache", "hit-validated")
			return NewContext(entry.MergedAliases, entry.MergedFunctions), nil
		}
		trace.Log(ctx, "cache", "invalid")
	} else {
		trace.Log(ctx, "cache", "miss")
	}

	var merged *config.Config
	trace.WithRegion(ctx, "LoadHierarchyWithAuth", func() {
		merged, _, err = loader.LoadHierarchyWithAuth(dir, authMgr)
	})
	if err != nil {
		return nil, err
	}
	if merged == nil {
		return NewContext(nil, nil), nil
	}

	c := NewContext(merged.GetAliases(), merged.Functions)

	// Write the refreshed snapshot back so the next exec/completion hits
	// the fast path, preserving any cleanup bookkeeping written by export.
	e.writeBack(cacheStore, loader, authMgr, dir, c)

	return c, nil
}

// writeBack refreshes the cached merged snapshot for dir
func (e *Engine) writeBack(cacheStore *cache.Cache, loader *config.Loader, authMgr *auth.Auth, dir string, c *Context) {
	chain := ActiveConfigChain(dir, authMgr, loader)
	hierarchyHash, hierarchyPaths, err := HierarchyHash(chain, loader)
	if err != nil || hierarchyHash == "" {
		return
	}

	entry := &cache.Entry{
		Path:            dir,
		Hash:            hierarchyHash,
		Timestamp:       time.Now(),
		Version:         version.Version,
		MergedAliases:   c.Aliases,
		MergedFunctions: c.Functions,
		HierarchyHash:   hierarchyHash,
		HierarchyPaths:  hierarchyPaths,
	}

	// Preserve the cleanup bookkeeping maintained by export
	if old, found := cacheStore.Get(dir); found {
		entry.Aliases = old.Aliases
		entry.Functions = old.Functions
		entry.EnvVars = old.EnvVars
		entry.LocalOnly = old.LocalOnly
	}

	// Best effort: a failed write only costs the next call a slow path
	_ = cacheStore.Set(entry)
}

// entryValidFast performs quick validation without file I/O:
// version match, merged snapshot present, and entry newer than the TTL
func entryValidFast(entry *cache.Entry) bool {
	return entry.Version == version.Version &&
		entry.MergedAliases != nil &&
		entry.HierarchyHash != "" &&
		time.Since(entry.Timestamp) < cacheValidationTTL
}

// validateEntry checks an expired entry against the config files on disk:
// the entry stays valid while the hierarchy hash matches
func validateEntry(entry *cache.Entry, dir string, loader *config.Loader, authMgr *auth.Auth) bool {
	if entry.Version != version.Version || entry.MergedAliases == nil || entry.HierarchyHash == "" {
		return false
	}

	activeChain := ActiveConfigChain(dir, authMgr, loader)
	currentHash, _, err := HierarchyHash(activeChain, loader)
	if err != nil {
		return false
	}

	return currentHash == entry.HierarchyHash
}

// HierarchyHash computes a composite hash from all configs in the chain.
// Returns the colon-joined per-file hashes and the config file paths,
// making the result sensitive to changes in any file of the hierarchy.
func HierarchyHash(configDirs []string, loader *config.Loader) (string, []string, error) {
	var hashes []string
	var paths []string

	for _, configDir := range configDirs {
		configPath := config.FindConfigInDir(configDir)
		if configPath == "" {
			continue
		}

		hash, err := loader.Hash(configPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to hash %s: %w", configPath, err)
		}

		hashes = append(hashes, hash)
		paths = append(paths, configPath)
	}

	return strings.Join(hashes, ":"), paths, nil
}
