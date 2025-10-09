package setup

import "strings"

// containsMarkers checks if content contains both start and end markers
func containsMarkers(content, startMarker, endMarker string) bool {
	return strings.Contains(content, startMarker) && strings.Contains(content, endMarker)
}

// removeMarkedSection removes a section marked by start and end markers
func removeMarkedSection(content, startMarker, endMarker string) string {
	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return content
	}

	// Remove the marked section including markers
	before := content[:startIdx]
	after := content[endIdx+len(endMarker):]

	// Trim extra newlines
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	if len(before) > 0 && len(after) > 0 {
		return before + "\n" + after
	}
	if len(before) > 0 {
		return before + "\n"
	}
	return after
}
