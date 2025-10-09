package completion

import (
	"fmt"
	"path/filepath"
	"strings"
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

// Complete tries each completer in order until one succeeds
func (e *Engine) Complete(tool string, args []string) (*Result, error) {
	// Check if we already know which completer works for this tool
	if cachedType := e.detectionCache.Get(tool); cachedType != "" {
		if completer, ok := e.completerByName[cachedType]; ok {
			suggestions, err := completer.Complete(tool, args)
			if err == nil && len(suggestions) > 0 {
				return &Result{
					Suggestions: suggestions,
					Source:      cachedType + " (cached)",
				}, nil
			}
		}
	}

	// Try each completer in order
	for _, completer := range e.completers {
		if !completer.Supports(tool, args) {
			continue
		}

		suggestions, err := completer.Complete(tool, args)
		if err != nil {
			continue
		}

		if len(suggestions) > 0 {
			source := getCompleterType(completer)
			e.detectionCache.Set(tool, source)
			_ = e.detectionCache.Save()

			return &Result{
				Suggestions: suggestions,
				Source:      source,
			}, nil
		}
	}

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
