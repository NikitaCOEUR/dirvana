package setup

import (
	"os"
	"strings"
)

// Markers that pre-strategy releases wrote around the inline hook block
// in RC files. Current releases never write them; they are only searched
// for so upgrades don't leave a second, stale hook behind.
const (
	legacyMarkerStart = "# Dirvana shell hook - START"
	legacyMarkerEnd   = "# Dirvana shell hook - END"
)

// cleanupLegacyHook removes the inline hook block that old releases wrote
// directly into the RC file. Returns true when a block was removed; a file
// without markers (or no file at all) is left untouched.
func cleanupLegacyHook(rcFile string) (bool, error) {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	content := string(data)
	startIdx := strings.Index(content, legacyMarkerStart)
	endIdx := strings.Index(content, legacyMarkerEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return false, nil
	}

	before := strings.TrimRight(content[:startIdx], "\n")
	after := strings.TrimLeft(content[endIdx+len(legacyMarkerEnd):], "\n")

	var newContent string
	switch {
	case before != "" && after != "":
		newContent = before + "\n\n" + after
	case before != "":
		newContent = before + "\n"
	default:
		newContent = after
	}

	if err := atomicWrite(rcFile, []byte(newContent)); err != nil {
		return false, err
	}
	return true, nil
}
