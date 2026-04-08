package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/verify"
)

// RepairSummary reports what the integration repair phase did, making the
// outcome visible to operators instead of silently swallowing errors.
type RepairSummary struct {
	AttemptCount  int
	PatchedFiles  int
	AgentErrors   int
	WriteErrors   int
	SkippedErrors []string // human-readable descriptions of skipped errors
}

// repairIntegrationErrors attempts to fix cross-task compilation errors that
// survived deterministic fixes by invoking an LLM on each failing module.
// Up to 2 rounds of LLM repair + deterministic cleanup + recheck are run.
// The final IntegrationResult (passing or failing) is returned along with a
// RepairSummary for operator visibility.
func repairIntegrationErrors(
	ctx context.Context,
	outputDir string,
	intResult verify.IntegrationResult,
	provider manifest.ProviderAssignment,
	tierOverrides map[ModelTier]string,
	verbose bool,
	logFn func(string),
) (verify.IntegrationResult, RepairSummary) {
	logf := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if logFn != nil {
			logFn(msg)
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
	}

	a := buildRepairAgent(provider, tierOverrides, verbose)
	var summary RepairSummary

	const maxRepairAttempts = 2
	const maxParallelRepairs = 4

	for attempt := 0; attempt < maxRepairAttempts; attempt++ {
		summary.AttemptCount++

		var mu sync.Mutex
		sem := make(chan struct{}, maxParallelRepairs)
		g, gctx := errgroup.WithContext(ctx)

		for _, ierr := range intResult.Errors {
			ierr := ierr // capture for goroutine
			g.Go(func() error {
				sem <- struct{}{}
				defer func() { <-sem }()

				dir := filepath.Join(outputDir, ierr.Dir)

				allFiles, err := collectModuleFiles(dir, ierr.Language)
				if err != nil {
					logf("realize: repair: could not read files from %s: %v", ierr.Dir, err)
					mu.Lock()
					summary.SkippedErrors = append(summary.SkippedErrors,
						fmt.Sprintf("module %s: read error: %v", ierr.Dir, err))
					mu.Unlock()
					return nil
				}
				if len(allFiles) == 0 {
					mu.Lock()
					summary.SkippedErrors = append(summary.SkippedErrors,
						fmt.Sprintf("module %s: no %s files found", ierr.Dir, ierr.Language))
					mu.Unlock()
					return nil
				}

				// Filter to error cluster: only files mentioned in compiler errors
				// plus files they import. This prevents context window overflow for
				// large modules while keeping enough context for the LLM to fix.
				sourceFiles := filterErrorCluster(allFiles, ierr.Output, ierr.Language)
				if len(sourceFiles) > config.MaxRepairFilesPerCall {
					logf("realize: repair: %s cluster has %d files (cap %d), truncating",
						ierr.Dir, len(sourceFiles), config.MaxRepairFilesPerCall)
					sourceFiles = sourceFiles[:config.MaxRepairFilesPerCall]
				}

				ac := buildRepairContext(attempt, sourceFiles, ierr.Dir, ierr.Output)
				result, agentErr := a.Run(gctx, ac)
				if agentErr != nil {
					mu.Lock()
					summary.AgentErrors++
					mu.Unlock()
					logf("realize: repair: agent error for %s (attempt %d): %v", ierr.Dir, attempt, agentErr)
					mu.Lock()
					summary.SkippedErrors = append(summary.SkippedErrors,
						fmt.Sprintf("module %s: agent error: %v", ierr.Dir, agentErr))
					mu.Unlock()
					return nil
				}

				// Write patched files back to disk, relative to the module directory.
				for _, f := range result.Files {
					fullPath := filepath.Join(dir, filepath.FromSlash(f.Path))
					if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
						mu.Lock()
						summary.WriteErrors++
						mu.Unlock()
						logf("realize: repair: mkdir %s: %v", filepath.Dir(fullPath), mkErr)
						continue
					}
					if writeErr := os.WriteFile(fullPath, []byte(f.Content), 0o644); writeErr != nil {
						mu.Lock()
						summary.WriteErrors++
						mu.Unlock()
						logf("realize: repair: write %s: %v", f.Path, writeErr)
						continue
					}
					mu.Lock()
					summary.PatchedFiles++
					mu.Unlock()
				}
				return nil
			})
		}

		// Wait for all parallel repairs to finish.
		_ = g.Wait()

		// Apply deterministic cleanup on LLM-patched output before rechecking.
		applyIntegrationFixes(outputDir)

		intResult = verify.RunIntegrationBuild(ctx, outputDir)
		if intResult.Passed {
			return intResult, summary
		}
	}
	return intResult, summary
}

// buildRepairAgent returns a TierSlow agent for integration repair, respecting
// any explicit tier override the user configured in the manifest.
func buildRepairAgent(pa manifest.ProviderAssignment, tierOverrides map[ModelTier]string, verbose bool) agent.Agent {
	if tierOverrides != nil {
		if modelID, ok := tierOverrides[TierSlow]; ok {
			return buildAgentWithModel(pa, modelID, config.DefaultMaxTokens, thinkingBudgetForTier(TierSlow), reasoningEffortForTier(TierSlow), verbose)
		}
	}
	return buildAgentForTier(pa, TierSlow, config.DefaultMaxTokens, verbose)
}

// buildRepairContext assembles an agent.Context for a single integration repair
// invocation. The failing source files are presented as dependency outputs so
// the agent sees their full content.
func buildRepairContext(attempt int, sourceFiles []dag.GeneratedFile, moduleDir, errOutput string) *agent.Context {
	excerpts := make([]memory.FileExcerpt, 0, len(sourceFiles))
	for _, f := range sourceFiles {
		excerpts = append(excerpts, memory.FileExcerpt{
			Path:    f.Path,
			Content: f.Content,
		})
	}

	syntheticOutput := &memory.TaskOutput{
		TaskID: "integration.repair.source",
		Label:  fmt.Sprintf("Failing source files in %s", moduleDir),
		Kind:   dag.TaskKindIntegrationRepair,
		Files:  excerpts,
	}

	return &agent.Context{
		Task: &dag.Task{
			ID:    "integration.repair",
			Kind:  dag.TaskKindIntegrationRepair,
			Label: "Integration Build Repair",
		},
		PreviousErrors:    errOutput,
		DependencyOutputs: []*memory.TaskOutput{syntheticOutput},
		AttemptNumber:     attempt,
	}
}

// filterErrorCluster selects only the files mentioned in compiler errors and their
// direct imports from the full module file list. If no files are extracted from
// the error output (or the result is small enough), returns all files as-is.
func filterErrorCluster(allFiles []dag.GeneratedFile, errOutput, language string) []dag.GeneratedFile {
	// If the module is small enough, send everything.
	if len(allFiles) <= config.MaxRepairFilesPerCall {
		return allFiles
	}

	// Extract file names from compiler error output.
	errorFiles := make(map[string]bool)
	for _, line := range strings.Split(errOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Go errors: "file.go:line:col: error message"
		// TS errors: "file.ts(line,col): error TS..."
		colon := strings.Index(line, ":")
		if colon > 0 {
			candidate := line[:colon]
			if isSourceExt(candidate, language) {
				errorFiles[filepath.ToSlash(candidate)] = true
			}
		}
	}

	if len(errorFiles) == 0 {
		return allFiles
	}

	// Build a lookup by path.
	byPath := make(map[string]dag.GeneratedFile, len(allFiles))
	for _, f := range allFiles {
		byPath[f.Path] = f
	}

	// Collect error cluster: files with errors + files they import.
	cluster := make(map[string]bool)
	for path := range errorFiles {
		cluster[path] = true
		// Find imports in the error file and add them to the cluster.
		if f, ok := byPath[path]; ok {
			for _, imp := range extractLocalImports(f.Content, language) {
				// Match imported package to a file in the module.
				for _, af := range allFiles {
					if strings.Contains(af.Path, imp) {
						cluster[af.Path] = true
					}
				}
			}
		}
	}

	// Build result in original order, error files first.
	var result []dag.GeneratedFile
	for _, f := range allFiles {
		if cluster[f.Path] {
			result = append(result, f)
		}
	}
	return result
}

// isSourceExt checks if a filename has a source extension for the given language.
func isSourceExt(name, language string) bool {
	switch language {
	case "go":
		return strings.HasSuffix(name, ".go")
	case "typescript":
		return strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")
	case "python":
		return strings.HasSuffix(name, ".py")
	}
	return false
}

// extractLocalImports extracts local (non-stdlib) import paths from source content.
func extractLocalImports(content, language string) []string {
	var imports []string
	switch language {
	case "go":
		// Match import lines like: "module/internal/domain"
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, `"`) && strings.Contains(line, "/internal/") {
				// Extract the last path component as a package match hint.
				imp := strings.Trim(line, `"`)
				parts := strings.Split(imp, "/")
				if len(parts) > 0 {
					imports = append(imports, parts[len(parts)-1])
				}
			}
		}
	}
	return imports
}

// collectModuleFiles reads all source files matching the given language from dir,
// skipping vendor, node_modules, and hidden directories. Returns GeneratedFile
// values with slash-normalised paths relative to dir.
func collectModuleFiles(dir, language string) ([]dag.GeneratedFile, error) {
	var ext string
	switch language {
	case "go":
		ext = ".go"
	case "typescript":
		ext = ".ts"
	case "python":
		ext = ".py"
	default:
		return nil, nil
	}

	var files []dag.GeneratedFile
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") ||
				name == "__pycache__" || name == "venv" || name == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ext) {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files; non-fatal
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return nil
		}
		files = append(files, dag.GeneratedFile{
			Path:    filepath.ToSlash(rel),
			Content: string(content),
		})
		return nil
	})
	return files, err
}
