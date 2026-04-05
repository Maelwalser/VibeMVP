package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/verify"
)

// repairIntegrationErrors attempts to fix cross-task compilation errors that
// survived deterministic fixes by invoking an LLM on each failing module.
// Up to 2 rounds of LLM repair + deterministic cleanup + recheck are run.
// The final IntegrationResult (passing or failing) is returned.
func repairIntegrationErrors(
	ctx context.Context,
	outputDir string,
	intResult verify.IntegrationResult,
	provider manifest.ProviderAssignment,
	tierOverrides map[ModelTier]string,
	verbose bool,
) verify.IntegrationResult {
	a := buildRepairAgent(provider, tierOverrides, verbose)

	const maxRepairAttempts = 2
	for attempt := 0; attempt < maxRepairAttempts; attempt++ {
		for _, ierr := range intResult.Errors {
			dir := filepath.Join(outputDir, ierr.Dir)

			sourceFiles, err := collectModuleFiles(dir, ierr.Language)
			if err != nil || len(sourceFiles) == 0 {
				continue
			}

			ac := buildRepairContext(attempt, sourceFiles, ierr.Dir, ierr.Output)
			result, err := a.Run(ctx, ac)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "realize: integration repair agent error: %v\n", err)
				}
				continue
			}

			// Write patched files back to disk, relative to the module directory.
			for _, f := range result.Files {
				fullPath := filepath.Join(dir, filepath.FromSlash(f.Path))
				if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
					continue
				}
				_ = os.WriteFile(fullPath, []byte(f.Content), 0o644)
			}
		}

		// Apply deterministic cleanup on LLM-patched output before rechecking.
		applyIntegrationFixes(outputDir)

		intResult = verify.RunIntegrationBuild(ctx, outputDir)
		if intResult.Passed {
			return intResult
		}
	}
	return intResult
}

// buildRepairAgent returns a TierSlow agent for integration repair, respecting
// any explicit tier override the user configured in the manifest.
func buildRepairAgent(pa manifest.ProviderAssignment, tierOverrides map[ModelTier]string, verbose bool) agent.Agent {
	if tierOverrides != nil {
		if modelID, ok := tierOverrides[TierSlow]; ok {
			return buildAgentWithModel(pa, modelID, config.DefaultMaxTokens, verbose)
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
			if name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") {
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
