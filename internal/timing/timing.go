// Package timing provides performance measurement utilities for Dirvana.
package timing

import (
	"fmt"
	"time"
)

// Timer tracks execution time of operations
type Timer struct {
	start time.Time
	marks map[string]time.Duration
	order []string // Track order of marks for consistent output
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{
		start: time.Now(),
		marks: make(map[string]time.Duration),
		order: make([]string, 0),
	}
}

// Mark records a checkpoint with a label
func (t *Timer) Mark(label string) time.Duration {
	elapsed := time.Since(t.start)
	t.marks[label] = elapsed
	t.order = append(t.order, label)
	return elapsed
}

// Elapsed returns total elapsed time since timer creation
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// Get returns the duration for a specific mark
func (t *Timer) Get(label string) (time.Duration, bool) {
	d, ok := t.marks[label]
	return d, ok
}

// Summary returns a formatted summary of all timings
func (t *Timer) Summary() string {
	total := t.Elapsed()
	summary := fmt.Sprintf("Total: %.3fms", float64(total.Microseconds())/1000.0)

	if len(t.marks) > 0 {
		summary += " ("
		for i, label := range t.order {
			dur := t.marks[label]
			if i > 0 {
				summary += ", "
			}
			summary += fmt.Sprintf("%s: %.3fms", label, float64(dur.Microseconds())/1000.0)
		}
		summary += ")"
	}

	return summary
}

// Reset resets the timer
func (t *Timer) Reset() {
	t.start = time.Now()
	t.marks = make(map[string]time.Duration)
	t.order = make([]string, 0)
}
