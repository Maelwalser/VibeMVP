package memory

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
)

// FileExcerpt is a filtered, possibly-truncated snapshot of one generated file.
type FileExcerpt struct {
	Path    string
	Content string
	// Truncated is true when the original file was larger than config.MaxFileChars.
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

// ConstructorSig records one exported constructor or factory function signature
// extracted from a committed file at its original, untruncated content.
// Stored in SharedMemory so downstream prompts receive accurate signatures even
// when file excerpts are truncated by the shared memory budget.
type ConstructorSig struct {
	// File is the module-relative source path, e.g. "internal/repository/postgres/user.go".
	File string
	// Package is the directory portion of File, e.g. "internal/repository/postgres".
	Package string
	// Signature is the full function declaration line with the body stripped,
	// e.g. "func NewUserRepository(db *pgxpool.Pool) (*UserRepository, error)".
	Signature string
}

// TypeEntry records where an exported type is first defined across the project.
// Used to build the cross-task type registry that prevents duplicate declarations.
type TypeEntry struct {
	// Package is the relative package directory, e.g. "internal/domain".
	Package string
	// File is the relative source file path, e.g. "internal/domain/user.go".
	File string
	// Definition is the full type declaration body (struct/interface fields included).
	// Injected into downstream agent prompts so they know method signatures without
	// having to re-read the full file excerpt.
	Definition string
}

// SharedMemory is a thread-safe store of completed task outputs.
// It is written to by TaskRunner after a successful commit and read by
// downstream agents before they are invoked.
type SharedMemory struct {
	mu           sync.RWMutex
	outputs      map[string]*TaskOutput
	rawPaths     map[string][]string  // task ID → committed file paths (untruncated)
	typeRegistry map[string]TypeEntry // exported type name → first-seen location
	constructors []ConstructorSig     // all constructor sigs extracted at commit time
}

// New returns an empty SharedMemory.
func New() *SharedMemory {
	return &SharedMemory{
		outputs:      make(map[string]*TaskOutput),
		rawPaths:     make(map[string][]string),
		typeRegistry: make(map[string]TypeEntry),
	}
}

// Record stores the output of a completed task. Only contextually useful files
// are retained (interface/type/schema/contract files); large files are truncated.
//
// files should contain the disk-prefixed paths (e.g. "backend/internal/domain/user.go").
// outputDir is stripped from file paths before building agent context excerpts, so
// agents see module-relative paths like "internal/domain/user.go" — not the filesystem
// output directory prefix. This prevents agents from constructing wrong import paths
// such as "backend/internal/domain" instead of the module-relative "internal/domain".
//
// rawPaths keeps the full prefixed paths so CommittedPaths can stage files correctly.
// Safe for concurrent use.
func (m *SharedMemory) Record(task *dag.Task, files []dag.GeneratedFile, outputDir string) {
	// Strip the output dir prefix for agent context: agents work with module-relative
	// paths, not filesystem paths. The OutputDir is a deployment artifact only.
	contextFiles := stripOutputDirFromFiles(files, outputDir)
	excerpts := buildExcerpts(contextFiles)
	out := &TaskOutput{
		TaskID: task.ID,
		Label:  task.Label,
		Kind:   task.Kind,
		Files:  excerpts,
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path // keep prefixed for disk staging
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[task.ID] = out
	m.rawPaths[task.ID] = paths
}

// stripOutputDirFromFiles returns a copy of files with the outputDir prefix stripped
// from each path. If outputDir is empty or ".", files are returned unchanged.
func stripOutputDirFromFiles(files []dag.GeneratedFile, outputDir string) []dag.GeneratedFile {
	if outputDir == "" || outputDir == "." {
		return files
	}
	prefix := filepath.ToSlash(outputDir) + "/"
	result := make([]dag.GeneratedFile, len(files))
	for i, f := range files {
		normalized := filepath.ToSlash(f.Path)
		stripped := strings.TrimPrefix(normalized, prefix)
		result[i] = dag.GeneratedFile{Path: stripped, Content: f.Content}
	}
	return result
}

// RegisterTypes records the exported types produced by a task into the shared type
// registry. If a type name is already registered (by an earlier task), the existing
// entry is kept — first-writer wins, which matches Go's compilation semantics where
// the first definition in the dependency order is authoritative.
// Safe for concurrent use.
func (m *SharedMemory) RegisterTypes(types map[string]TypeEntry) {
	if len(types) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, entry := range types {
		if _, exists := m.typeRegistry[name]; !exists {
			m.typeRegistry[name] = entry
		}
	}
}

// RegisterConstructors appends constructor signatures extracted from file to the
// shared registry. Called at commit time on untruncated file content so that
// downstream prompts always receive accurate signatures regardless of excerpt
// truncation. Safe for concurrent use.
func (m *SharedMemory) RegisterConstructors(file string, sigs []string) {
	if len(sigs) == 0 {
		return
	}
	pkg := filepath.Dir(file)
	if pkg == "." {
		pkg = ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sig := range sigs {
		m.constructors = append(m.constructors, ConstructorSig{
			File:      file,
			Package:   pkg,
			Signature: sig,
		})
	}
}

// AllConstructors returns a snapshot of every constructor signature registered so far.
// Safe for concurrent use.
func (m *SharedMemory) AllConstructors() []ConstructorSig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ConstructorSig, len(m.constructors))
	copy(result, m.constructors)
	return result
}

// TypeRegistry returns a snapshot of all exported types seen so far.
// Safe for concurrent use.
func (m *SharedMemory) TypeRegistry() map[string]TypeEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]TypeEntry, len(m.typeRegistry))
	for k, v := range m.typeRegistry {
		result[k] = v
	}
	return result
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

	budget := config.MaxTotalCharsFor(string(task.Kind))

	var results []*TaskOutput
	total := 0

	for _, depID := range task.Dependencies {
		out, ok := m.outputs[depID]
		if !ok {
			continue
		}
		if total >= budget {
			break
		}
		// Shallow-copy the output, trimming files once the budget is reached.
		trimmed := &TaskOutput{
			TaskID: out.TaskID,
			Label:  out.Label,
			Kind:   out.Kind,
		}
		for _, f := range out.Files {
			if total >= budget {
				break
			}
			content := f.Content
			remaining := budget - total
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
		originalExceeded := len(f.Content) > config.MaxFileChars
		content := extractSignatures(f.Path, f.Content)
		truncated := originalExceeded
		if len(content) > config.MaxFileChars {
			content = content[:config.MaxFileChars] + "\n// ... [truncated]"
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

	// All files under a domain/ package contain entity structs and sentinel
	// errors that every downstream task needs to stay type-consistent.
	if strings.Contains(lower, "/domain/") {
		return true
	}
	// Repository interface and error files are the binding contract between layers.
	if strings.HasSuffix(lower, "interfaces.go") {
		return true
	}
	if strings.Contains(lower, "/repository/") && strings.HasSuffix(lower, "errors.go") {
		return true
	}

	suffixes := []string{
		"types.go", "models.go", "schema.go",
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
