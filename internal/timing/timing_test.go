package timing

import (
	"strings"
	"testing"
	"time"
)

func TestTimer_Basic(t *testing.T) {
	timer := NewTimer()

	time.Sleep(10 * time.Millisecond)
	timer.Mark("checkpoint1")

	time.Sleep(10 * time.Millisecond)
	timer.Mark("checkpoint2")

	elapsed := timer.Elapsed()
	if elapsed < 20*time.Millisecond {
		t.Errorf("Expected at least 20ms, got %v", elapsed)
	}

	// Check marks
	if d, ok := timer.Get("checkpoint1"); !ok {
		t.Error("checkpoint1 not found")
	} else if d < 10*time.Millisecond {
		t.Errorf("checkpoint1 should be >= 10ms, got %v", d)
	}

	if d, ok := timer.Get("checkpoint2"); !ok {
		t.Error("checkpoint2 not found")
	} else if d < 20*time.Millisecond {
		t.Errorf("checkpoint2 should be >= 20ms, got %v", d)
	}
}

func TestTimer_Summary(t *testing.T) {
	timer := NewTimer()

	time.Sleep(5 * time.Millisecond)
	timer.Mark("step1")

	time.Sleep(5 * time.Millisecond)
	timer.Mark("step2")

	summary := timer.Summary()

	// Check that summary contains expected parts
	if !strings.Contains(summary, "Total:") {
		t.Errorf("Summary should contain 'Total:', got: %s", summary)
	}

	if !strings.Contains(summary, "step1:") {
		t.Errorf("Summary should contain 'step1:', got: %s", summary)
	}

	if !strings.Contains(summary, "step2:") {
		t.Errorf("Summary should contain 'step2:', got: %s", summary)
	}

	if !strings.Contains(summary, "ms") {
		t.Errorf("Summary should contain 'ms', got: %s", summary)
	}
}

func TestTimer_Reset(t *testing.T) {
	timer := NewTimer()

	time.Sleep(10 * time.Millisecond)
	timer.Mark("before_reset")

	timer.Reset()

	// After reset, elapsed should be very small
	elapsed := timer.Elapsed()
	if elapsed > 5*time.Millisecond {
		t.Errorf("After reset, elapsed should be small, got %v", elapsed)
	}

	// Old mark should not exist
	if _, ok := timer.Get("before_reset"); ok {
		t.Error("Mark should not exist after reset")
	}
}
