package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecentPaths = 5

// recentFile returns the path to the recent-manifests JSON file.
func recentFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibemenu", "recent.json"), nil
}

// LoadRecentPaths returns the list of recently accessed manifest paths,
// most recent first. Returns nil on any error.
func LoadRecentPaths() []string {
	p, err := recentFile()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil
	}
	return paths
}

// RecordRecentPath prepends path to the persisted recent list.
// Silently ignores any I/O errors.
func RecordRecentPath(path string) {
	existing := LoadRecentPaths()
	filtered := make([]string, 0, len(existing))
	for _, p := range existing {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	updated := append([]string{path}, filtered...)
	if len(updated) > maxRecentPaths {
		updated = updated[:maxRecentPaths]
	}
	p, err := recentFile()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return
	}
	data, err := json.Marshal(updated)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0644)
}
