package skills

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
)

// FileRegistry implements Registry by reading skill markdown files from a directory.
type FileRegistry struct {
	skillsDir  string
	index      map[string]string // normalized key → content
	checksums  map[string]string // normalized key → SHA256 hex digest
}

// Load reads all *.md files from skillsDir and returns a FileRegistry.
// If skillsDir does not exist, an empty registry is returned without error.
func Load(skillsDir string) (*FileRegistry, error) {
	r := &FileRegistry{
		skillsDir: skillsDir,
		index:     make(map[string]string),
		checksums: make(map[string]string),
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
		content := string(data)
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if len(content) > config.MaxSkillBytes {
			content = content[:config.MaxSkillBytes] + "\n\n[skill document truncated — core rules above are sufficient]"
		}
		r.index[key] = content
		r.checksums[key] = hash
	}
	return r, nil
}

// WriteSkillsLock writes a JSON lock file mapping skill names to SHA256 checksums.
// Called once per pipeline run so the exact skill versions used are reproducible.
func (r *FileRegistry) WriteSkillsLock(outputDir string) error {
	if len(r.checksums) == 0 {
		return nil
	}
	lockDir := filepath.Join(outputDir, ".realize")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.checksums, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(lockDir, "skills.lock"), data, 0o644)
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
