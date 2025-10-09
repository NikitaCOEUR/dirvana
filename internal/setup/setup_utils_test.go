package setup

import (
	"strings"
	"testing"
)

func TestContainsMarkers(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		startMarker string
		endMarker   string
		want        bool
	}{
		{
			name:        "Both markers present",
			content:     "before\n" + HookMarkerStart + "\nmiddle\n" + HookMarkerEnd + "\nafter",
			startMarker: HookMarkerStart,
			endMarker:   HookMarkerEnd,
			want:        true,
		},
		{
			name:        "Only start marker",
			content:     "before\n" + HookMarkerStart + "\nmiddle",
			startMarker: HookMarkerStart,
			endMarker:   HookMarkerEnd,
			want:        false,
		},
		{
			name:        "Only end marker",
			content:     "middle\n" + HookMarkerEnd + "\nafter",
			startMarker: HookMarkerStart,
			endMarker:   HookMarkerEnd,
			want:        false,
		},
		{
			name:        "No markers",
			content:     "just some content",
			startMarker: HookMarkerStart,
			endMarker:   HookMarkerEnd,
			want:        false,
		},
		{
			name:        "Empty content",
			content:     "",
			startMarker: HookMarkerStart,
			endMarker:   HookMarkerEnd,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMarkers(tt.content, tt.startMarker, tt.endMarker)
			if result != tt.want {
				t.Errorf("containsMarkers() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestRemoveMarkedSection(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		startMarker     string
		endMarker       string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "Remove section from middle",
			content: "# User content before\n" +
				"# More before\n" +
				HookMarkerStart + "\n" +
				"# Hook code line 1\n" +
				"# Hook code line 2\n" +
				HookMarkerEnd + "\n" +
				"# User content after\n" +
				"# More after",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# User content before", "# More before", "# User content after", "# More after"},
			wantNotContains: []string{HookMarkerStart, HookMarkerEnd, "# Hook code line 1", "# Hook code line 2"},
		},
		{
			name: "Remove section at end",
			content: "# User content\n" +
				HookMarkerStart + "\n" +
				"# Hook code\n" +
				HookMarkerEnd,
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# User content"},
			wantNotContains: []string{HookMarkerStart, HookMarkerEnd, "# Hook code"},
		},
		{
			name: "Remove section at beginning",
			content: HookMarkerStart + "\n" +
				"# Hook code\n" +
				HookMarkerEnd + "\n" +
				"# User content",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# User content"},
			wantNotContains: []string{HookMarkerStart, HookMarkerEnd, "# Hook code"},
		},
		{
			name:            "No markers present - return unchanged",
			content:         "# Just user content\n# More content",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# Just user content", "# More content"},
			wantNotContains: []string{HookMarkerStart, HookMarkerEnd},
		},
		{
			name: "Only start marker - return unchanged",
			content: "# Content\n" +
				HookMarkerStart + "\n" +
				"# More content",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# Content", HookMarkerStart, "# More content"},
			wantNotContains: []string{},
		},
		{
			name: "Only end marker - return unchanged",
			content: "# Content\n" +
				HookMarkerEnd + "\n" +
				"# More content",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# Content", HookMarkerEnd, "# More content"},
			wantNotContains: []string{},
		},
		{
			name: "Markers in wrong order - return unchanged",
			content: "# Content\n" +
				HookMarkerEnd + "\n" +
				"# Middle\n" +
				HookMarkerStart + "\n" +
				"# More",
			startMarker:     HookMarkerStart,
			endMarker:       HookMarkerEnd,
			wantContains:    []string{"# Content", HookMarkerEnd, "# Middle", HookMarkerStart, "# More"},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeMarkedSection(tt.content, tt.startMarker, tt.endMarker)

			for _, needle := range tt.wantContains {
				if !strings.Contains(result, needle) {
					t.Errorf("Result should contain %q\nGot: %s", needle, result)
				}
			}

			for _, needle := range tt.wantNotContains {
				if strings.Contains(result, needle) {
					t.Errorf("Result should NOT contain %q\nGot: %s", needle, result)
				}
			}
		})
	}
}

func TestRemoveMarkedSection_PreservesContent(t *testing.T) {
	// Test that user content is preserved exactly (no extra newlines added/removed incorrectly)
	content := "line1\nline2\n" +
		HookMarkerStart + "\n" +
		"hook\n" +
		HookMarkerEnd + "\n" +
		"line3\nline4\n"

	result := removeMarkedSection(content, HookMarkerStart, HookMarkerEnd)

	// Should have preserved the user content
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line2") ||
		!strings.Contains(result, "line3") || !strings.Contains(result, "line4") {
		t.Error("User content was not fully preserved")
	}

	// Should not have the hook markers or content
	if strings.Contains(result, HookMarkerStart) || strings.Contains(result, HookMarkerEnd) ||
		strings.Contains(result, "hook") {
		t.Error("Hook markers or content still present")
	}
}

func TestRemoveMarkedSection_TrimsExtraNewlines(t *testing.T) {
	content := "line1\n" +
		HookMarkerStart + "\n" +
		"hook\n" +
		HookMarkerEnd + "\n" +
		"line2\n"

	result := removeMarkedSection(content, HookMarkerStart, HookMarkerEnd)

	// Should not have excessive newlines between line1 and line2
	// The function trims and adds back a single newline
	lines := strings.Split(result, "\n")

	// Filter out empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// Should have exactly 2 non-empty lines: line1 and line2
	if len(nonEmptyLines) != 2 {
		t.Errorf("Expected 2 non-empty lines, got %d: %v", len(nonEmptyLines), nonEmptyLines)
	}
}
