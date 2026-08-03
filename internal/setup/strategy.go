package setup

import (
	"github.com/NikitaCOEUR/dirvana/internal/fsutil"
	"github.com/NikitaCOEUR/dirvana/internal/shell"
)

// InstallStrategy defines the interface for different hook installation strategies
type InstallStrategy interface {
	// Install installs the hook using the strategy
	Install() error
	// Uninstall removes the hook
	Uninstall() error
	// IsInstalled checks if the hook is currently installed
	IsInstalled() bool
	// NeedsUpdate checks if the hook needs to be updated
	NeedsUpdate() bool
	// GetMessage returns a user-friendly message about the installation
	GetMessage() string
	// GetRCFile returns the RC file path (if applicable)
	GetRCFile() string
	// HookFile returns the file holding the hook code itself, which is not
	// the RC file whenever the strategy keeps it separate.
	HookFile() string
}

// Common messages used across strategies
const (
	MsgHookUpToDate = "✓ Dirvana hook is up to date"
)

// SelectInstallStrategy selects the best installation strategy for the given shell
func SelectInstallStrategy(shellName string) (InstallStrategy, error) {
	// Fish requires special handling due to is-interactive block
	if shellName == shell.Fish {
		return NewFishHookStrategy()
	}

	// Try drop-in strategy first (cleanest approach)
	dropIn, err := NewDropInStrategy(shellName)
	if err == nil && dropIn.IsSupported() {
		return dropIn, nil
	}

	// Fallback to external hook strategy
	return NewExternalHookStrategy(shellName)
}

// atomicWrite writes an RC or hook file atomically (0644: shell config
// files are conventionally world-readable)
func atomicWrite(filename string, data []byte) error {
	return fsutil.AtomicWrite(filename, data, 0o644)
}
