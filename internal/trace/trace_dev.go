//go:build dev

// Package trace provides runtime tracing for development builds.
// This is the dev version with actual tracing support via runtime/trace.
//
// Usage:
//
//	DIRVANA_TRACE=trace.out dirvana completion k get ''
//	go tool trace trace.out
package trace

import (
	"context"
	"fmt"
	"os"
	"runtime/trace"
	"sync"
)

var (
	traceFile   *os.File
	traceMu     sync.Mutex
	traceActive bool
)

// Init initializes tracing if DIRVANA_TRACE is set to a file path.
// Returns a cleanup function that should be deferred.
func Init() func() {
	tracePath := os.Getenv("DIRVANA_TRACE")
	if tracePath == "" {
		return func() {}
	}

	traceMu.Lock()
	defer traceMu.Unlock()

	var err error
	traceFile, err = os.Create(tracePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dirvana: failed to create trace file %s: %v\n", tracePath, err)
		return func() {}
	}

	if err := trace.Start(traceFile); err != nil {
		fmt.Fprintf(os.Stderr, "dirvana: failed to start trace: %v\n", err)
		traceFile.Close()
		traceFile = nil
		return func() {}
	}

	traceActive = true
	fmt.Fprintf(os.Stderr, "dirvana: tracing to %s\n", tracePath)

	return func() {
		traceMu.Lock()
		defer traceMu.Unlock()

		if traceActive {
			trace.Stop()
			traceActive = false
		}
		if traceFile != nil {
			traceFile.Close()
			traceFile = nil
		}
	}
}

// Region creates a trace region. Returns a function to end the region.
func Region(ctx context.Context, regionType string) func() {
	if !traceActive {
		return func() {}
	}
	region := trace.StartRegion(ctx, regionType)
	return region.End
}

// StartRegion starts a named region (for manual end control).
func StartRegion(ctx context.Context, regionType string) {
	if traceActive {
		trace.StartRegion(ctx, regionType)
	}
}

// EndRegion ends the current region.
func EndRegion() {
	// Note: trace.StartRegion returns a *Region that must be ended via End()
	// This function is kept for API compatibility but Region() is preferred
}

// Log logs a message to the trace.
func Log(ctx context.Context, category, message string) {
	if traceActive {
		trace.Log(ctx, category, message)
	}
}

// WithRegion executes a function within a trace region.
func WithRegion(ctx context.Context, regionType string, f func()) {
	if traceActive {
		trace.WithRegion(ctx, regionType, f)
	} else {
		f()
	}
}

// IsEnabled returns true if tracing is enabled.
func IsEnabled() bool {
	return traceActive
}
