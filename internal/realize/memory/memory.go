package memory

import (
	"strings"
	"sync"

	"github.com/vibe-menu/internal/realize/dag"
)

const (
	// maxFileChars is the maximum characters included from a single file after
	// signature extraction. Type-signature excerpts are much more compact than
	// full implementations, so this budget covers most files completely.
	maxFileChars = 1500

	// maxTotalChars is the total character budget across all dependency outputs
	// injected into one agent's context. Signature-only context is denser than
	// raw file content, so a lower budget covers all necessary type information.
	maxTotalChars = 8000
)

// FileExcerpt is a filtered, possibly-truncated snapshot of one generated file.
type FileExcerpt struct {
	Path    string
	Content string
	// Truncated is true when the original file was larger than maxFileChars.
	Truncated bool
}

// TaskOutput captures the files a completed task produced, filtered to
// excerpts most useful as shared context for downstream agents.
type TaskOutput struct {
	TaskID string
	Label  string
	Kind   dag.TaskKind
	Files  []FileExcerpt
}

// SharedMemory is a thread-safe store of completed task outputs.
// It is written to by TaskRunner after a successful commit and read by
// downstream agents before they are invoked.
type SharedMemory struct {
	mu       sync.RWMutex
	outputs  map[string]*TaskOutput
	rawPaths map[string][]string // task ID → committed file paths (untruncated)
}

// New returns an empty SharedMemory.
func New() *SharedMemory {
	return &SharedMemory{
		outputs:  make(map[string]*TaskOutput),
		rawPaths: make(map[string][]string),
	}
}

// Record stores the output of a completed task. Only contextually useful files
// are retained (interface/type/schema/contract files); large files are truncated.
// Safe for concurrent use.
func (m *SharedMemory) Record(task *dag.Task, files []dag.GeneratedFile) {
	excerpts := buildExcerpts(files)
	out := &TaskOutput{
		TaskID: task.ID,
		Label:  task.Label,
		Kind:   task.Kind,
		Files:  excerpts,
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[task.ID] = out
	m.rawPaths[task.ID] = paths
}

// CommittedPaths returns all file paths committed by the given dependency task IDs.
// Used by downstream task runners to stage dependency files in the verifier sandbox.
// Safe for concurrent use.
func (m *SharedMemory) CommittedPaths(depIDs []string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]struct{})
	var result []string
	for _, id := range depIDs {
		for _, p := range m.rawPaths[id] {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				result = append(result, p)
			}
		}
	}
	return result
}

// DepsOf returns the recorded outputs for each direct dependency of task.
// Dependencies with no recorded output (e.g. skipped on resume) are omitted.
// The returned slice is ordered by dependency ID for determinism.
// Safe for concurrent use.
func (m *SharedMemory) DepsOf(task *dag.Task) []*TaskOutput {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*TaskOutput
	total := 0

	for _, depID := range task.Dependencies {
		out, ok := m.outputs[depID]
		if !ok {
			continue
		}
		if total >= maxTotalChars {
			break
		}
		// Shallow-copy the output, trimming files once the budget is reached.
		trimmed := &TaskOutput{
			TaskID: out.TaskID,
			Label:  out.Label,
			Kind:   out.Kind,
		}
		for _, f := range out.Files {
			if total >= maxTotalChars {
				break
			}
			content := f.Content
			remaining := maxTotalChars - total
			if len(content) > remaining {
				content = content[:remaining] + "\n// [truncated by shared memory budget]"
			}
			trimmed.Files = append(trimmed.Files, FileExcerpt{
				Path:      f.Path,
				Content:   content,
				Truncated: f.Truncated || len(content) < len(f.Content),
			})
			total += len(content)
		}
		if len(trimmed.Files) > 0 {
			results = append(results, trimmed)
		}
	}

	return results
}

// buildExcerpts filters and truncates a file list to retain only the entries
// most relevant as shared context (type/interface/schema files), then applies
// signature extraction and the per-file character cap.
func buildExcerpts(files []dag.GeneratedFile) []FileExcerpt {
	// Separate high-value files from the rest.
	var priority, rest []dag.GeneratedFile
	for _, f := range files {
		if isHighValue(f.Path) {
			priority = append(priority, f)
		} else {
			rest = append(rest, f)
		}
	}

	// Include all high-value files first, then fill remaining budget with rest.
	ordered := append(priority, rest...)
	excerpts := make([]FileExcerpt, 0, len(ordered))
	for _, f := range ordered {
		// Extract only type signatures and declarations — not implementation bodies.
		// This reduces per-file context by ~80–90% while preserving all structural
		// information downstream agents need to stay type-consistent.
		// Mark Truncated=true when the original exceeded the budget (signature
		// extraction may have already shrunk the content below the cap, but the
		// caller still needs to know the excerpt is not the full file).
		originalExceeded := len(f.Content) > maxFileChars
		content := extractSignatures(f.Path, f.Content)
		truncated := originalExceeded
		if len(content) > maxFileChars {
			content = content[:maxFileChars] + "\n// ... [truncated]"
			truncated = true
		}
		excerpts = append(excerpts, FileExcerpt{
			Path:      f.Path,
			Content:   content,
			Truncated: truncated,
		})
	}
	return excerpts
}

// isHighValue reports whether a file path suggests it contains type, interface,
// schema, or contract definitions — the most useful shared context.
func isHighValue(path string) bool {
	lower := strings.ToLower(path)
	suffixes := []string{
		"types.go", "models.go", "schema.go", "interfaces.go",
		"entities.go", "domain.go", "dto.go",
		"types.ts", "models.ts", "schema.ts", "types.tsx",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	keywords := []string{
		".proto", "openapi", "swagger", "_types", "_models",
		"_schema", "_interfaces", "_entities",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
