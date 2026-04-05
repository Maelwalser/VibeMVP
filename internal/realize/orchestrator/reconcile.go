package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/output"
	"github.com/vibe-menu/internal/realize/state"
	"github.com/vibe-menu/internal/realize/verify"
)

// runReconciliationTask is a specialized runner for TaskKindReconciliation.
// Unlike the standard TaskRunner it does not generate new files — it reads the
// codebase already written to disk by upstream service tasks and patches only the
// files that fail to compile.
//
// Algorithm:
//  1. Apply deterministic import-path fixes (zero LLM cost, catches the majority of errors).
//  2. Run go build ./... across all Go modules under the backend scan root.
//  3. If no errors: mark complete without any LLM call.
//  4. If errors: for each failing module, collect ALL source files (untruncated),
//     invoke a TierSlow repair agent, and write only the returned files back to disk.
func runReconciliationTask(
	ctx context.Context,
	task *dag.Task,
	writer *output.Writer,
	st *state.Store,
	provider manifest.ProviderAssignment,
	tierOverrides map[ModelTier]string,
	verbose bool,
	logFn func(string),
) error {
	logf := makeTaskLogger(task.ID, logFn)

	outputDir := writer.BaseDir()

	// Derive the root directory from which Go modules are scanned.
	// OutputDir in the payload is the backend base directory relative to outputDir:
	//   - "backend" for monolith with frontend
	//   - "."       for backend-only monolith or microservices (scan from project root)
	scanRoot := outputDir
	if rel := task.Payload.OutputDir; rel != "" && rel != "." {
		scanRoot = filepath.Join(outputDir, filepath.FromSlash(rel))
	}

	// Step 1: deterministic import-path fixes — zero cost, fixes the most common class
	// of cross-task errors (agents using placeholder module paths like "github.com/your-org").
	if fixes := verify.FixImportPaths(outputDir); fixes != "" {
		logf("import path pre-flight: %s", fixes)
	}

	// Step 2: project-wide Go build to discover cross-task compilation errors.
	logf("running go build ./... across all Go modules under %s", scanRoot)
	intResult := verify.RunIntegrationBuild(ctx, scanRoot)

	if intResult.Passed {
		logf("reconciliation: all modules compile — no cross-task errors found ✓")
		return st.MarkCompleted(task.ID)
	}

	// Collect only Go failures — the reconciliation task owns Go compilation only.
	var goErrors []verify.IntegrationError
	for _, e := range intResult.Errors {
		if e.Language == "go" {
			goErrors = append(goErrors, e)
		}
	}
	if len(goErrors) == 0 {
		logf("reconciliation: no Go compilation errors — skipping LLM repair")
		return st.MarkCompleted(task.ID)
	}

	logf("reconciliation: %d Go module(s) have compilation errors — invoking LLM repair", len(goErrors))

	// Step 3: per-module LLM repair.
	// Each failing module is repaired in its own agent call so prompts stay within
	// the context window even for large codebases.
	a := buildRepairAgent(provider, tierOverrides, verbose)

	for _, ierr := range goErrors {
		// ierr.Dir is relative to scanRoot (set by RunIntegrationBuild).
		moduleDir := filepath.Join(scanRoot, filepath.FromSlash(ierr.Dir))

		sourceFiles, err := collectModuleFiles(moduleDir, "go")
		if err != nil || len(sourceFiles) == 0 {
			logf("reconciliation: could not read files from %s (%v) — skipping", ierr.Dir, err)
			continue
		}

		logf("reconciliation: repairing module %s (%d files)", ierr.Dir, len(sourceFiles))
		if verbose {
			logf("reconciliation: build errors in %s:\n%s", ierr.Dir, ierr.Output)
		}

		ac := buildReconciliationAgentContext(task, sourceFiles, ierr.Dir, ierr.Output)
		result, agentErr := a.Run(ctx, ac)
		if agentErr != nil {
			// Non-fatal: log and continue; the post-pipeline integration repair step
			// will attempt a second pass on any remaining errors.
			logf("reconciliation: agent error for %s: %v", ierr.Dir, agentErr)
			continue
		}

		if len(result.Files) == 0 {
			logf("reconciliation: agent returned no changes for %s", ierr.Dir)
			continue
		}

		// Write only the patched files back to disk.
		for _, f := range result.Files {
			fullPath := filepath.Join(moduleDir, filepath.FromSlash(f.Path))
			if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
				logf("reconciliation: mkdir %s: %v", filepath.Dir(fullPath), mkErr)
				continue
			}
			if writeErr := os.WriteFile(fullPath, []byte(f.Content), 0o644); writeErr != nil {
				logf("reconciliation: write %s: %v", f.Path, writeErr)
			}
		}
		logf("reconciliation: patched %d file(s) in %s", len(result.Files), ierr.Dir)
	}

	// Apply deterministic cleanup on any LLM-modified files before marking complete.
	// This catches formatting drift introduced by the repair agent.
	if fixes := verify.FixImportPaths(outputDir); fixes != "" {
		logf("reconciliation post-fix: %s", fixes)
	}

	return st.MarkCompleted(task.ID)
}

// buildReconciliationAgentContext constructs the agent.Context for one reconciliation
// LLM call. All source files from the failing module are included at FULL content
// (never truncated) so the agent can reason across the entire module to identify the
// root cause of each compilation error.
func buildReconciliationAgentContext(task *dag.Task, sourceFiles []dag.GeneratedFile, moduleDir, buildErrors string) *agent.Context {
	excerpts := make([]memory.FileExcerpt, 0, len(sourceFiles))
	for _, f := range sourceFiles {
		excerpts = append(excerpts, memory.FileExcerpt{
			Path:      f.Path,
			Content:   f.Content,
			Truncated: false, // always full content for reconciliation
		})
	}

	syntheticOutput := &memory.TaskOutput{
		TaskID: "reconciliation.source",
		Label:  fmt.Sprintf("Complete Go module at %s", moduleDir),
		Kind:   dag.TaskKindReconciliation,
		Files:  excerpts,
	}

	return &agent.Context{
		Task:              task,
		PreviousErrors:    buildErrors,
		DependencyOutputs: []*memory.TaskOutput{syntheticOutput},
		AttemptNumber:     0,
	}
}

// makeTaskLogger returns a logging function that prefixes messages with the task ID.
func makeTaskLogger(taskID string, logFn func(string)) func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf("[%s] "+format, append([]interface{}{taskID}, args...)...)
		if logFn != nil {
			logFn(msg)
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}
