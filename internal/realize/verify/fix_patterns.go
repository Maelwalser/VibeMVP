package verify

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

//go:embed fix_patterns.json
var fixPatternsJSON []byte

// FixPattern describes a simple string replacement fix for a common LLM hallucination.
// Patterns are loaded from the embedded fix_patterns.json file at startup.
type FixPattern struct {
	ID         string `json:"id"`
	Match      string `json:"match"`
	Replace    string `json:"replace"`
	FileFilter string `json:"file_filter"` // glob pattern, e.g. "*_test.go"
	Language   string `json:"language"`     // "go", "typescript", "python"
}

var loadedPatterns []FixPattern

func init() {
	_ = json.Unmarshal(fixPatternsJSON, &loadedPatterns)
}

// ApplyPatternFixes applies the embedded fix patterns to files in dir.
// Returns a summary of fixes applied, or "" if nothing changed.
func ApplyPatternFixes(dir string, files []string, language string) string {
	if len(loadedPatterns) == 0 {
		return ""
	}

	// Filter patterns by language.
	var patterns []FixPattern
	for _, p := range loadedPatterns {
		if p.Language == language {
			patterns = append(patterns, p)
		}
	}
	if len(patterns) == 0 {
		return ""
	}

	var applied []string
	for _, relPath := range files {
		absPath := filepath.Join(dir, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		content := string(data)
		modified := false

		for _, p := range patterns {
			// Check file filter.
			if p.FileFilter != "" {
				matched, _ := filepath.Match(p.FileFilter, filepath.Base(relPath))
				if !matched {
					continue
				}
			}
			if strings.Contains(content, p.Match) {
				content = strings.ReplaceAll(content, p.Match, p.Replace)
				modified = true
				applied = append(applied, p.ID)
			}
		}

		if modified {
			_ = os.WriteFile(absPath, []byte(content), 0644)
		}
	}

	if len(applied) == 0 {
		return ""
	}
	// Deduplicate pattern IDs.
	seen := make(map[string]bool)
	var unique []string
	for _, id := range applied {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return "pattern fixes: " + strings.Join(unique, ", ")
}
