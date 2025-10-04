package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigChainCalculation tests the calculation of active config chains
func TestConfigChainCalculation(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		configDirs  []string // Directories with .dirvana.yml files (from root to leaf)
		authorized  []string // Authorized directories
		localOnly   string   // Directory with local_only flag (empty if none)
		expected    []string // Expected active config chain
	}{
		{
			name:       "Simple hierarchy A/B/C with all authorized",
			dir:        "/A/B/C",
			configDirs: []string{"/A", "/A/B", "/A/B/C"},
			authorized: []string{"/A", "/A/B", "/A/B/C"},
			expected:   []string{"/A", "/A/B", "/A/B/C"},
		},
		{
			name:       "A/B/C with B not authorized",
			dir:        "/A/B/C",
			configDirs: []string{"/A", "/A/B", "/A/B/C"},
			authorized: []string{"/A", "/A/B/C"}, // B not authorized
			expected:   []string{"/A", "/A/B/C"}, // B should be skipped
		},
		{
			name:       "A/B with B not authorized - should only have A",
			dir:        "/A/B",
			configDirs: []string{"/A", "/A/B"},
			authorized: []string{"/A"}, // B not authorized
			expected:   []string{"/A"}, // Only A should be active
		},
		{
			name:       "Local only in C should ignore parents",
			dir:        "/A/B/C",
			configDirs: []string{"/A", "/A/B", "/A/B/C"},
			authorized: []string{"/A", "/A/B", "/A/B/C"},
			localOnly:  "/A/B/C",
			expected:   []string{"/A/B/C"}, // Only C due to local_only
		},
		{
			name:       "Local only in B should ignore A",
			dir:        "/A/B/C",
			configDirs: []string{"/A", "/A/B", "/A/B/C"},
			authorized: []string{"/A", "/A/B", "/A/B/C"},
			localOnly:  "/A/B",
			expected:   []string{"/A/B", "/A/B/C"}, // B and C, A ignored due to local_only in B
		},
		{
			name:       "No configs in current dir should return empty",
			dir:        "/X/Y/Z",
			configDirs: []string{"/A"},
			authorized: []string{"/A"},
			expected:   []string{}, // Z is not in config hierarchy
		},
		{
			name:       "Unauthorized root directory",
			dir:        "/A/B/C",
			configDirs: []string{"/A", "/A/B", "/A/B/C"},
			authorized: []string{"/A/B", "/A/B/C"}, // A not authorized
			expected:   []string{"/A/B", "/A/B/C"}, // Should start from B
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock auth checker
			authSet := make(map[string]bool)
			for _, dir := range tt.authorized {
				authSet[dir] = true
			}
			mockAuth := &mockAuthChecker{allowed: authSet}

			// Create mock config provider
			localOnlySet := make(map[string]bool)
			if tt.localOnly != "" {
				localOnlySet[tt.localOnly] = true
			}
			mockConfig := &mockConfigProvider{
				configs:   tt.configDirs,
				localOnly: localOnlySet,
			}

			result := GetActiveConfigChain(tt.dir, mockAuth, mockConfig)
			assert.Equal(t, tt.expected, result, "Active config chain mismatch")
		})
	}
}

// TestCleanupCalculation tests what needs to be cleaned up when moving between directories
func TestCleanupCalculation(t *testing.T) {
	tests := []struct {
		name            string
		prevChain       []string
		currentChain    []string
		expectedCleanup []string // Directories whose configs should be cleaned up
	}{
		{
			name:            "Moving from A to A/B - no cleanup needed",
			prevChain:       []string{"/A"},
			currentChain:    []string{"/A", "/A/B"},
			expectedCleanup: []string{},
		},
		{
			name:            "Moving from A/B/C to A/B - cleanup C only",
			prevChain:       []string{"/A", "/A/B/C"},
			currentChain:    []string{"/A"},
			expectedCleanup: []string{"/A/B/C"},
		},
		{
			name:            "Moving from A/B/C to A - cleanup B and C",
			prevChain:       []string{"/A", "/A/B", "/A/B/C"},
			currentChain:    []string{"/A"},
			expectedCleanup: []string{"/A/B", "/A/B/C"},
		},
		{
			name:            "Moving to different hierarchy - cleanup everything",
			prevChain:       []string{"/A", "/A/B"},
			currentChain:    []string{"/E", "/E/F"},
			expectedCleanup: []string{"/A", "/A/B"},
		},
		{
			name:            "Moving from A/B/C to A (C has local_only) - cleanup C only",
			prevChain:       []string{"/A/B/C"}, // C had local_only, so only C was active
			currentChain:    []string{"/A"},
			expectedCleanup: []string{"/A/B/C"},
		},
		{
			name:            "Same directory - no cleanup",
			prevChain:       []string{"/A", "/A/B"},
			currentChain:    []string{"/A", "/A/B"},
			expectedCleanup: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCleanup(tt.prevChain, tt.currentChain)
			assert.Equal(t, tt.expectedCleanup, result, "Cleanup calculation mismatch")
		})
	}
}

// Mock implementations for testing

type mockAuthChecker struct {
	allowed map[string]bool
}

func (m *mockAuthChecker) IsAllowed(path string) (bool, error) {
	return m.allowed[path], nil
}

type mockConfigProvider struct {
	configs   []string          // List of directories with configs
	localOnly map[string]bool   // Directories with local_only flag
}

func (m *mockConfigProvider) FindConfigs(dir string) []string {
	var result []string
	for _, configDir := range m.configs {
		// Check if configDir is in the hierarchy of dir
		if isInHierarchy(configDir, dir) {
			result = append(result, configDir)
		}
	}
	return result
}

func (m *mockConfigProvider) IsLocalOnly(dir string) bool {
	return m.localOnly[dir]
}

// isInHierarchy checks if configDir is in the directory hierarchy of targetDir
// e.g., /A is in hierarchy of /A/B/C
func isInHierarchy(configDir, targetDir string) bool {
	// Simple string prefix check (works for test purposes)
	return targetDir == configDir || len(targetDir) > len(configDir) && targetDir[:len(configDir)+1] == configDir+"/"
}
