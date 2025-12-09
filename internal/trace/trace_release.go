//go:build !dev

// Package trace provides runtime tracing for development builds.
// This is the release version with no-op stubs.
package trace

import "context"

// Init initializes tracing. In release builds, this is a no-op.
// Returns a cleanup function that should be deferred.
func Init() func() {
	return func() {}
}

// Region creates a trace region. In release builds, this is a no-op.
func Region(ctx context.Context, regionType string) func() {
	return func() {}
}

// StartRegion starts a named region. In release builds, this is a no-op.
func StartRegion(ctx context.Context, regionType string) {
}

// EndRegion ends the current region. In release builds, this is a no-op.
func EndRegion() {
}

// Log logs a message to the trace. In release builds, this is a no-op.
func Log(ctx context.Context, category, message string) {
}

// WithRegion executes a function within a trace region. In release builds, just calls f.
func WithRegion(ctx context.Context, regionType string, f func()) {
	f()
}

// IsEnabled returns true if tracing is enabled.
func IsEnabled() bool {
	return false
}
