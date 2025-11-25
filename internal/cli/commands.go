package cli

// This file has been refactored. The main command implementations have been moved to:
// - export_cmd.go: Export command and related helpers
// - allow_cmd.go: Allow, Revoke, and List commands
// - init_cmd.go: Init command
// - exec.go: Exec command (already existed)
// - edit.go: Edit command (already existed)
// - completion.go: Completion command (already existed)
// - validate.go: Validate command (already existed)
// - schema.go: Schema command (already existed)
// - hooks.go: Hook and Setup commands (already existed)
// - clean.go: Clean command (already existed)
// - status.go: Status command (already existed)
//
// Common helpers and utilities are in:
// - helpers.go: Shared helper functions for all commands
