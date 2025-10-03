// Package completion provides a pluggable completion system with multiple strategies.
package completion

// Suggestion represents a single completion suggestion
type Suggestion struct {
	Value       string // The actual value to complete
	Description string // Optional description/help text
}

// Completer defines the interface for completion strategies
type Completer interface {
	// Supports returns true if this completer can handle the given tool
	Supports(tool string, args []string) bool

	// Complete returns completion suggestions for the given tool and arguments
	// Returns suggestions and nil if successful, or nil and error if failed
	Complete(tool string, args []string) ([]Suggestion, error)
}

// Result represents the result of a completion attempt
type Result struct {
	Suggestions []Suggestion
	Source      string // Which completer provided these suggestions
}
