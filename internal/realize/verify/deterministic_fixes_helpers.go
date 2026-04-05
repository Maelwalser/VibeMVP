package verify

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// filterByExt returns the absolute paths (under dir) of files whose extension
// matches any of the given extensions (e.g. ".ts", ".tsx", ".py").
func filterByExt(dir string, files []string, exts ...string) []string {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[strings.ToLower(e)] = true
	}
	var result []string
	for _, f := range files {
		if extSet[strings.ToLower(filepath.Ext(f))] {
			result = append(result, filepath.Join(dir, f))
		}
	}
	return result
}

// snapshotFiles records the byte content of each absolute file path so that
// countChanged can detect modifications after running an external tool.
func snapshotFiles(paths []string) map[string][]byte {
	snap := make(map[string][]byte, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			snap[p] = data
		}
	}
	return snap
}

// countChanged returns how many of the given paths differ from their snapshot.
func countChanged(paths []string, before map[string][]byte) int {
	n := 0
	for _, p := range paths {
		after, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if !bytes.Equal(before[p], after) {
			n++
		}
	}
	return n
}
