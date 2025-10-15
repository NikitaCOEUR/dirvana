package completion

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Engine orchestrates multiple completion strategies
type Engine struct {
	completers      []Completer
	detectionCache  *DetectionCache
	completerByName map[string]Completer
}

// NewEngine creates a new completion engine with all strategies
func NewEngine(cacheDir string) *Engine {
	flag := NewFlagCompleter()
	cobra := NewCobraCompleter()
	env := NewEnvCompleter()
	script := NewScriptCompleter(cacheDir)

	// Load detection cache
	cachePath := filepath.Join(cacheDir, "completion-detection.json")
	detectionCache, _ := NewDetectionCache(cachePath)

	return &Engine{
		completers: []Completer{
			cobra,  // Try Cobra first (kubectl, helm, etc.) - most specific
			flag,   // Then flag-based (dirvana, and other Go CLI tools)
			env,    // Then env-based (terraform, consul, vault, nomad, etc.)
			script, // Finally script-based (git, docker, systemctl, etc.)
		},
		detectionCache: detectionCache,
		completerByName: map[string]Completer{
			"Flag":   flag,
			"Cobra":  cobra,
			"Env":    env,
			"Script": script,
		},
	}
}

// completerResult holds the result of a parallel completer attempt
type completerResult struct {
	completer   Completer
	suggestions []Suggestion
	err         error
}

// Complete tries all completers in parallel and returns the first successful result
func (e *Engine) Complete(tool string, args []string) (*Result, error) {
	// Check if we already know which completer works for this tool
	if cachedType := e.detectionCache.Get(tool); cachedType != "" {
		if completer, ok := e.completerByName[cachedType]; ok {
			suggestions, err := completer.Complete(tool, args)
			if err == nil {
				// Return immediately, even with empty suggestions
				return &Result{
					Suggestions: suggestions,
					Source:      cachedType + " (cached)",
				}, nil
			}
		}
	}

	// Launch all completers in parallel
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()

	resultChan := make(chan completerResult, len(e.completers))
	var wg sync.WaitGroup

	// Start a goroutine for each completer
	for _, completer := range e.completers {
		wg.Add(1)
		go func(c Completer) {
			defer wg.Done()

			// Check if this completer supports the tool
			if !c.Supports(tool, args) {
				return
			}

			// Try to complete
			suggestions, err := c.Complete(tool, args)
			if err != nil {
				return
			}

			// Send result to channel
			select {
			case resultChan <- completerResult{completer: c, suggestions: suggestions, err: nil}:
			case <-ctx.Done():
				return
			}
		}(completer)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Wait for first result (even if empty) or all completions
	for result := range resultChan {
		// Cache and return first result immediately, even if no suggestions
		// This prevents waiting for slower completers
		source := getCompleterType(result.completer)
		e.detectionCache.Set(tool, source)
		_ = e.detectionCache.Save()

		cancel() // Stop other goroutines

		// Wait for all goroutines to finish cleanup
		// This ensures subprocesses are terminated and files are closed
		// before we return, preventing test cleanup issues
		wg.Wait()

		return &Result{
			Suggestions: result.suggestions,
			Source:      source,
		}, nil
	}

	// No completer supported this tool
	return &Result{
		Suggestions: []Suggestion{},
		Source:      "none",
	}, nil
}

// Filter applies prefix filtering to suggestions
func (e *Engine) Filter(suggestions []Suggestion, prefix string) []Suggestion {
	if prefix == "" {
		return suggestions
	}

	var filtered []Suggestion
	for _, s := range suggestions {
		if strings.HasPrefix(s.Value, prefix) {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

func getCompleterType(completer Completer) string {
	source := fmt.Sprintf("%T", completer)
	source = strings.TrimPrefix(source, "*completion.")
	source = strings.TrimSuffix(source, "Completer")
	return source
}
