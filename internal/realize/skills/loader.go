package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
)

// FileRegistry implements Registry by reading skill markdown files from a directory.
type FileRegistry struct {
	skillsDir string
	index     map[string]string // normalized key → content
}

// Load reads all *.md files from skillsDir and returns a FileRegistry.
// If skillsDir does not exist, an empty registry is returned without error.
func Load(skillsDir string) (*FileRegistry, error) {
	r := &FileRegistry{
		skillsDir: skillsDir,
		index:     make(map[string]string),
	}

	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read skills dir %s: %w", skillsDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		key := strings.TrimSuffix(e.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(skillsDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read skill file %s: %w", e.Name(), err)
		}
		r.index[key] = string(data)
	}
	return r, nil
}

// Lookup returns the content for the given technology string, or ("", false).
func (r *FileRegistry) Lookup(technology string) (string, bool) {
	key := normalize(technology)
	if content, ok := r.index[key]; ok {
		return content, true
	}
	return "", false
}

// LookupAll returns all skill docs relevant to a task kind and technology list.
// Technology-specific skills are looked up first; universal quality skills for the
// task kind are appended afterwards (deduplication ensures no double-injection).
func (r *FileRegistry) LookupAll(kind dag.TaskKind, technologies []string) []Doc {
	seen := make(map[string]bool)
	docs := make([]Doc, 0)

	for _, tech := range technologies {
		if tech == "" {
			continue
		}
		key := normalize(tech)
		if seen[key] {
			continue
		}
		seen[key] = true
		content, ok := r.index[key]
		if !ok {
			continue
		}
		docs = append(docs, Doc{Technology: tech, Content: content})
	}

	// Inject universal quality skills for this task kind.
	for _, key := range universalSkillsForKind[kind] {
		if seen[key] {
			continue
		}
		seen[key] = true
		content, ok := r.index[key]
		if !ok {
			continue
		}
		docs = append(docs, Doc{Technology: key, Content: content})
	}

	return docs
}

// normalize maps a technology name to its skill file base name.
func normalize(tech string) string {
	if alias, ok := aliasMap[tech]; ok {
		return alias
	}
	// Fallback: lowercase with spaces replaced by hyphens.
	return strings.ToLower(strings.ReplaceAll(tech, " ", "-"))
}
