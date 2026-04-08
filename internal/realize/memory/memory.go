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

// ServiceMethodSig records one exported method signature on a service/repository
// struct, extracted from committed files at their original, untruncated content.
// Unlike ConstructorSig (which covers New*/Make*/etc.), this covers all exported
// methods with receivers — the exact signatures handler tasks need to generate
// compatible calls.
type ServiceMethodSig struct {
	File      string
	Package   string
	Signature string
}

// TypeConflict records a type name declared in two different packages.
// Diagnostic-only: used by the orchestrator to log warnings after all tasks complete.
type TypeConflict struct {
	TypeName string
	First    TypeEntry
	Second   TypeEntry
}

// SharedMemory is a thread-safe store of completed task outputs.
// It is written to by TaskRunner after a successful commit and read by
// downstream agents before they are invoked.
type SharedMemory struct {
	mu             sync.RWMutex
	outputs        map[string]*TaskOutput
	rawPaths       map[string][]string  // task ID → committed file paths (untruncated)
	typeRegistry   map[string]TypeEntry // exported type name → first-seen location
	typeConflicts  []TypeConflict       // types declared in multiple packages
	constructors    []ConstructorSig     // all constructor sigs extracted at commit time
	serviceMethods  []ServiceMethodSig   // all exported method sigs extracted at commit time
	errorSentinels      []ErrorSentinel      // all Err* var declarations extracted at commit time
	interfaceContracts  []InterfaceContract  // all interface definitions extracted at commit time
	crossTaskIssues     string               // build errors from incremental compilation
	goModByModule       map[string]string    // Go module path → locked go.mod content
	goSumByModule       map[string]string    // Go module path → locked go.sum content
}

// New returns an empty SharedMemory.
func New() *SharedMemory {
	return &SharedMemory{
		outputs:       make(map[string]*TaskOutput),
		rawPaths:      make(map[string][]string),
		typeRegistry:  make(map[string]TypeEntry),
		goModByModule: make(map[string]string),
		goSumByModule: make(map[string]string),
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
		existing, exists := m.typeRegistry[name]
		if !exists {
			m.typeRegistry[name] = entry
		} else if existing.Package != entry.Package {
			// Same type name declared in different packages — potential conflict.
			m.typeConflicts = append(m.typeConflicts, TypeConflict{
				TypeName: name,
				First:    existing,
				Second:   entry,
			})
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

// RegisterServiceMethods appends exported method signatures (non-constructor) from
// a committed file to the shared registry. Called at commit time on untruncated
// content so downstream handler/bootstrap tasks see accurate method signatures even
// when file excerpts are truncated by the memory budget.
// Safe for concurrent use.
func (m *SharedMemory) RegisterServiceMethods(file string, sigs []string) {
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
		m.serviceMethods = append(m.serviceMethods, ServiceMethodSig{
			File:      file,
			Package:   pkg,
			Signature: sig,
		})
	}
}

// RegisterErrorSentinels appends exported Err* variable declarations from a
// committed file to the shared registry. Called at commit time so downstream
// prompts receive an explicit list of available sentinel names.
// Safe for concurrent use.
func (m *SharedMemory) RegisterErrorSentinels(sentinels []ErrorSentinel) {
	if len(sentinels) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorSentinels = append(m.errorSentinels, sentinels...)
}

// RegisterInterfaceContracts appends interface contracts extracted from committed Go
// files to the shared registry. Called at commit time so downstream tasks receive
// exact interface method signatures as a hard implementation checklist.
// Safe for concurrent use.
func (m *SharedMemory) RegisterInterfaceContracts(contracts []InterfaceContract) {
	if len(contracts) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceContracts = append(m.interfaceContracts, contracts...)
}

// AllInterfaceContracts returns a snapshot of every interface contract registered so far.
// Safe for concurrent use.
func (m *SharedMemory) AllInterfaceContracts() []InterfaceContract {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]InterfaceContract, len(m.interfaceContracts))
	copy(result, m.interfaceContracts)
	return result
}

// SetCrossTaskIssues records build errors detected by incremental compilation after
// a task commits. Downstream tasks see these as advisory context.
// Safe for concurrent use.
func (m *SharedMemory) SetCrossTaskIssues(issues string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.crossTaskIssues = issues
}

// CrossTaskIssues returns any known cross-task build errors from incremental compilation.
// Safe for concurrent use.
func (m *SharedMemory) CrossTaskIssues() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.crossTaskIssues
}

// AllErrorSentinels returns a snapshot of every error sentinel registered so far.
// Safe for concurrent use.
func (m *SharedMemory) AllErrorSentinels() []ErrorSentinel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ErrorSentinel, len(m.errorSentinels))
	copy(result, m.errorSentinels)
	return result
}

// AllServiceMethods returns a snapshot of every exported method signature registered so far.
// Safe for concurrent use.
func (m *SharedMemory) AllServiceMethods() []ServiceMethodSig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ServiceMethodSig, len(m.serviceMethods))
	copy(result, m.serviceMethods)
	return result
}

// TypeConflicts returns all type names that were declared in multiple packages.
// Diagnostic-only: call after all tasks complete to log warnings.
// Safe for concurrent use.
func (m *SharedMemory) TypeConflicts() []TypeConflict {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]TypeConflict, len(m.typeConflicts))
	copy(result, m.typeConflicts)
	return result
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

// StoreLockedGoMod records the locked go.mod content for a given module path.
// Called when a dependency resolution task commits its go.mod.
// Safe for concurrent use.
func (m *SharedMemory) StoreLockedGoMod(modulePath, content string) {
	if modulePath == "" || content == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.goModByModule[modulePath] = content
}

// LockedGoMod returns the stored go.mod content for a module path, or "" if none.
// Safe for concurrent use.
func (m *SharedMemory) LockedGoMod(modulePath string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.goModByModule[modulePath]
}

// AnyLockedGoMod returns the first stored go.mod content, or "" if none.
// Useful as a fallback when the exact module path is unknown.
// Safe for concurrent use.
func (m *SharedMemory) AnyLockedGoMod() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, content := range m.goModByModule {
		return content
	}
	return ""
}

// StoreLockedGoSum records the locked go.sum content for a given module path.
// Called alongside StoreLockedGoMod when a dependency resolution task commits.
// Safe for concurrent use.
func (m *SharedMemory) StoreLockedGoSum(modulePath, content string) {
	if modulePath == "" || content == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.goSumByModule[modulePath] = content
}

// LockedGoSum returns the stored go.sum content for a module path, or "" if none.
// Safe for concurrent use.
func (m *SharedMemory) LockedGoSum(modulePath string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.goSumByModule[modulePath]
}

// AnyLockedGoSum returns the first stored go.sum content, or "" if none.
// Safe for concurrent use.
func (m *SharedMemory) AnyLockedGoSum() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, content := range m.goSumByModule {
		return content
	}
	return ""
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

// depPriority returns a weight (0.0–1.0) indicating how important a dependency's
// context is for a specific downstream consumer. Higher-weight dependencies get a
// larger share of the character budget. The consumer task kind influences priority
// because different layers need different upstream information — e.g. a handler
// task needs service method signatures more than full domain struct bodies.
func depPriority(depKind, consumerKind dag.TaskKind) float64 {
	// Consumer-specific overrides for the most impactful task pairs.
	switch consumerKind {
	case dag.TaskKindServiceHandler:
		switch depKind {
		case dag.TaskKindServiceLogic:
			return 0.95 // handler needs exact method signatures
		case dag.TaskKindAuth:
			return 0.9 // middleware signatures
		case dag.TaskKindDataSchemas:
			return 0.5 // only struct names, not full bodies
		case dag.TaskKindServicePlan:
			return 0.7 // interfaces for route wiring
		}
	case dag.TaskKindServiceBootstrap:
		switch depKind {
		case dag.TaskKindServiceRepository, dag.TaskKindServiceLogic,
			dag.TaskKindServiceHandler, dag.TaskKindAuth:
			return 0.9 // bootstrap needs constructor signatures from all layers
		}
	case dag.TaskKindServicePlan:
		if depKind == dag.TaskKindDataSchemas {
			return 0.95 // plan needs full domain attribute lists
		}
	case dag.TaskKindServiceRepository:
		if depKind == dag.TaskKindDataSchemas {
			return 0.9 // repo needs full domain structs for SQL mapping
		}
	}

	// Default priorities by dependency kind.
	switch depKind {
	case dag.TaskKindAuth:
		return 0.9
	case dag.TaskKindDataSchemas:
		return 0.85
	case dag.TaskKindServiceLogic:
		return 0.8
	case dag.TaskKindServiceHandler:
		return 0.7
	case dag.TaskKindServicePlan:
		return 0.6
	case dag.TaskKindServiceRepository:
		return 0.5
	case dag.TaskKindDataMigrations:
		return 0.15
	case dag.TaskKindDependencyResolution:
		return 0.1
	default:
		return 0.5
	}
}

// DepsOf returns the recorded outputs for each direct dependency of task.
// Dependencies with no recorded output (e.g. skipped on resume) are omitted.
// Budget is allocated proportionally by dependency priority weight so that
// high-value dependencies (auth signatures, service methods) are not crowded
// out by verbose lower-priority outputs.
// Safe for concurrent use.
func (m *SharedMemory) DepsOf(task *dag.Task) []*TaskOutput {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalBudget := config.MaxTotalCharsFor(string(task.Kind))

	// Phase 1: collect present dependencies and compute proportional budgets.
	type depSlot struct {
		out    *TaskOutput
		weight float64
		budget int
		used   int
	}
	var slots []depSlot
	totalWeight := 0.0

	for _, depID := range task.Dependencies {
		out, ok := m.outputs[depID]
		if !ok {
			continue
		}
		w := depPriority(out.Kind, task.Kind)
		slots = append(slots, depSlot{out: out, weight: w})
		totalWeight += w
	}
	if len(slots) == 0 {
		return nil
	}

	// Allocate budget proportionally by weight.
	if totalWeight == 0 {
		totalWeight = 1.0 // avoid division by zero
	}
	allocated := 0
	for i := range slots {
		slots[i].budget = int(float64(totalBudget) * (slots[i].weight / totalWeight))
		allocated += slots[i].budget
	}
	// Distribute rounding remainder to the highest-weight slot.
	if remainder := totalBudget - allocated; remainder > 0 && len(slots) > 0 {
		bestIdx := 0
		for i := 1; i < len(slots); i++ {
			if slots[i].weight > slots[bestIdx].weight {
				bestIdx = i
			}
		}
		slots[bestIdx].budget += remainder
	}

	// Phase 2: fill each dependency up to its allocated budget.
	results := make([]*TaskOutput, 0, len(slots))
	totalUnused := 0

	for i := range slots {
		s := &slots[i]
		trimmed := &TaskOutput{
			TaskID: s.out.TaskID,
			Label:  s.out.Label,
			Kind:   s.out.Kind,
		}
		for _, f := range s.out.Files {
			if s.used >= s.budget {
				break
			}
			content := f.Content
			remaining := s.budget - s.used
			if len(content) > remaining {
				content = content[:remaining] + "\n// [truncated by shared memory budget]"
			}
			trimmed.Files = append(trimmed.Files, FileExcerpt{
				Path:      f.Path,
				Content:   content,
				Truncated: f.Truncated || len(content) < len(f.Content),
			})
			s.used += len(content)
		}
		totalUnused += s.budget - s.used
		if len(trimmed.Files) > 0 {
			results = append(results, trimmed)
		}
	}

	// Phase 3: redistribute unused budget to slots that were truncated.
	// This handles cases where small deps (e.g. go.mod) use only a fraction
	// of their allocation — the surplus flows to deps that need more room.
	if totalUnused > 0 {
		for i := range slots {
			s := &slots[i]
			if s.used < s.budget || totalUnused <= 0 {
				continue // not truncated or no surplus left
			}
			// Find the corresponding result entry.
			for ri := range results {
				if results[ri].TaskID != s.out.TaskID {
					continue
				}
				// Try to extend the last truncated file or add more files.
				extraBudget := totalUnused
				for fi := range s.out.Files {
					if extraBudget <= 0 {
						break
					}
					f := s.out.Files[fi]
					// Check if this file was already fully included.
					if fi < len(results[ri].Files) && !results[ri].Files[fi].Truncated {
						continue
					}
					if fi < len(results[ri].Files) {
						// Extend truncated file.
						content := f.Content
						if len(content) > s.used+extraBudget {
							content = content[:s.used+extraBudget] + "\n// [truncated by shared memory budget]"
						}
						added := len(content) - len(results[ri].Files[fi].Content)
						if added > 0 {
							results[ri].Files[fi].Content = content
							results[ri].Files[fi].Truncated = len(content) < len(f.Content)
							extraBudget -= added
							totalUnused -= added
						}
					} else {
						// Add new file from surplus.
						content := f.Content
						if len(content) > extraBudget {
							content = content[:extraBudget] + "\n// [truncated by shared memory budget]"
						}
						results[ri].Files = append(results[ri].Files, FileExcerpt{
							Path:      f.Path,
							Content:   content,
							Truncated: len(content) < len(f.Content),
						})
						extraBudget -= len(content)
						totalUnused -= len(content)
					}
				}
				break
			}
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
		// Python
		"models.py", "schemas.py", "types.py", "interfaces.py", "entities.py",
		// Java
		"entity.java", "repository.java", "model.java",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	keywords := []string{
		".proto", "openapi", "swagger", "_types", "_models",
		"_schema", "_interfaces", "_entities",
		// Config/dependency files that downstream agents need for correct versions
		"go.mod", "package.json", "pyproject.toml", "requirements.txt",
		// Interface directories
		"/interfaces/",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
