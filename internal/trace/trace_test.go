package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tracing is opt-in: the release build stubs everything out, and the dev
// build stays inert until DIRVANA_TRACE names a file. Both must therefore
// behave identically here — running the traced code, tracing nothing.
func TestTracingIsInertWhenNotRequested(t *testing.T) {
	t.Setenv("DIRVANA_TRACE", "")

	stop := Init()
	require.NotNil(t, stop)
	defer stop()

	assert.False(t, IsEnabled())

	ctx := context.Background()

	endRegion := Region(ctx, "test.region")
	require.NotNil(t, endRegion)
	endRegion()

	StartRegion(ctx, "test.manual")
	EndRegion()

	Log(ctx, "category", "message")

	// WithRegion must run the wrapped work whether or not tracing is on
	called := false
	WithRegion(ctx, "test.with", func() { called = true })
	assert.True(t, called)
}
