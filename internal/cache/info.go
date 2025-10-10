package cache

import (
	"encoding/json"
	"os"
)

// Info contains information about the cache file
type Info struct {
	Path         string
	Size         int64
	TotalEntries int
}

// GetCacheInfo returns information about the cache file
func GetCacheInfo(cachePath string) (*Info, error) {
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Info{Path: cachePath}, nil
		}
		return nil, err
	}

	result := &Info{
		Path: cachePath,
		Size: info.Size(),
	}

	// Count total entries
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return result, nil // Return partial info
	}

	var entries map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return result, nil // Return partial info
	}

	result.TotalEntries = len(entries)

	return result, nil
}
