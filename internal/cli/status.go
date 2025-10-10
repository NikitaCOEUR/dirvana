package cli

import (
	"fmt"

	"github.com/NikitaCOEUR/dirvana/internal/status"
)

// StatusParams contains parameters for the Status command
type StatusParams struct {
	CachePath string
	AuthPath  string
}

// Status displays the current Dirvana configuration status
func Status(params StatusParams) error {
	// Collect all status data
	data, err := status.CollectAll(params.CachePath, params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to collect status data: %w", err)
	}

	// Render and display
	output := status.Render(data)
	fmt.Println(output)

	return nil
}
