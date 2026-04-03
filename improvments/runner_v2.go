//go:build ignore

package orchestrator

// This file shows the KEY CHANGES to runner.go. Merge these into your existing runner.go.
//
// Changes:
// 1. Apply deterministic fixes BEFORE every verification (not just on retry)
// 2. Copy locked go.mod over agent-generated one when available
// 3. Classify errors and skip LLM retry for mechanically fixable issues
// 4. Inject previous attempt files on model escalation to prevent type redeclaration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/output"
	"github.com/vibe-menu/internal/realize/skills"
	"github.com/vibe-menu/internal/realize/state"
	"github.com/vibe-menu/internal/realize/verify"
)

// errorType classifies a verification failure for routing to the right fix strategy.
type errorType int

const (
	errTypeUnknown   errorType = iota
	errTypeGofmt               // fixable by gofmt -w
	errTypeEscape              // fixable by raw string conversion
	errTypeDeps                // fixable by go mod tidy with correct versions
	errTypeUndefined           // needs LLM retry with targeted guidance
	errTypeTestFail            // needs LLM retry with test output
	errTypeDuplicate           // fixable by removing duplicate decls
)

// classifyError inspects verification output and returns the dominant error type.
func classifyError(output string) errorType {
	switch {
	case strings.Contains(output, "unknown escape sequence"):
		return errTypeEscape
	case strings.Contains(output, "files not gofmt-clean") &&
		!strings.Contains(output, "undefined:") &&
		!strings.Contains(output, "FAIL"):
		return errTypeGofmt
	case strings.Contains(output, "missing go.sum entry") ||
		strings.Contains(output, "invalid version") ||
		strings.Contains(output, "cannot find module"):
		return errTypeDeps
	case strings.Contains(output, "redeclared in this block"):
		return errTypeDuplicate
	case strings.Contains(output, "--- FAIL:"):
		return errTypeTestFail
	case strings.Contains(output, "undefined:") ||
		strings.Contains(output, "does not implement"):
		return errTypeUndefined
	default:
		return errTypeUnknown
	}
}

// TaskRunner handles one task's agent invocation + verification retry loop.
// MODIFIED: added lockedGoMod field and deterministic fix integration.
type TaskRunnerV2 struct {
	task        *dag.Task
	agent       agent.Agent
	verifier    verify.Verifier
	writer      *output.Writer
	state       *state.Store
	memory      *memory.SharedMemory
	skillDocs   []skills.Doc
	maxRetries  int
	verbose     bool
	logFn       func(string)
	lockedGoMod string // if non-empty, overwrites any agent-generated go.mod
}

func (r *TaskRunnerV2) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if r.logFn != nil {
		r.logFn(msg)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// Run executes the task with improved retry logic.
func (r *TaskRunnerV2) Run(ctx context.Context) error {
	var lastVerifyOutput string
	var lastFiles []dag.GeneratedFile

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			r.log("[%s] retry %d/%d", r.task.ID, attempt, r.maxRetries)

			// ── NEW: classify the error and try deterministic fix first ──
			errType := classifyError(lastVerifyOutput)
			tmpDir := filepath.Join(r.writer.BaseDir(), ".tmp", r.task.ID)

			switch errType {
			case errTypeGofmt, errTypeEscape, errTypeDuplicate:
				// Try deterministic fix without burning an LLM call.
				r.log("[%s] applying deterministic fixes before retry", r.task.ID)
				fixes := verify.ApplyDeterministicFixes(tmpDir, verify.FilePaths(lastFiles))
				if fixes != "" {
					r.log("[%s] deterministic fix applied: %s", r.task.ID, fixes)
					// Re-verify without calling the LLM.
					vResult, err := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(lastFiles))
					if err != nil {
						return fmt.Errorf("task %s: verifier error: %w", r.task.ID, err)
					}
					if vResult.Passed {
						r.log("[%s] deterministic fix resolved verification — skipping LLM retry", r.task.ID)
						return r.commitAndRecord(tmpDir, lastFiles)
					}
					lastVerifyOutput = vResult.Output
					// If deterministic fix didn't fully resolve it, fall through to LLM retry.
				}

			case errTypeDeps:
				// Try fixing go.mod without LLM.
				if r.lockedGoMod != "" {
					modPath := findGoMod(tmpDir)
					if modPath != "" {
						os.WriteFile(modPath, []byte(r.lockedGoMod), 0644)
						// Re-run go mod tidy.
						runGoModTidy(ctx, filepath.Dir(modPath))
						vResult, err := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(lastFiles))
						if err == nil && vResult.Passed {
							r.log("[%s] deps fix resolved verification — skipping LLM retry", r.task.ID)
							return r.commitAndRecord(tmpDir, lastFiles)
						}
						if err == nil {
							lastVerifyOutput = vResult.Output
						}
					}
				}
			}
		}

		ac := &agent.Context{
			Task:              r.task,
			SkillDocs:         r.skillDocs,
			PreviousErrors:    lastVerifyOutput,
			DependencyOutputs: r.memory.DepsOf(r.task),
		}

		result, err := r.agent.Run(ctx, ac)
		if err != nil {
			if attempt == r.maxRetries {
				return fmt.Errorf("task %s: agent failed after %d attempts: %w", r.task.ID, attempt+1, err)
			}
			if isRateLimitError(err) {
				wait := time.Duration(attempt+1) * 60 * time.Second
				r.log("[%s] rate limited — waiting %s", r.task.ID, wait)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			lastVerifyOutput = fmt.Sprintf("Agent error: %v", err)
			continue
		}

		lastFiles = result.Files

		// Write to task-scoped temp directory.
		tmpDir := filepath.Join(r.writer.BaseDir(), ".tmp", r.task.ID)
		if err := r.writer.WriteAllTo(tmpDir, result.Files); err != nil {
			return fmt.Errorf("task %s: write to temp dir: %w", r.task.ID, err)
		}

		// ── NEW: overwrite go.mod with locked version if available ──
		if r.lockedGoMod != "" {
			if modPath := findGoMod(tmpDir); modPath != "" {
				os.WriteFile(modPath, []byte(r.lockedGoMod), 0644)
				runGoModTidy(ctx, filepath.Dir(modPath))
			}
		}

		// ── NEW: apply deterministic fixes BEFORE every verification ──
		if fixes := verify.ApplyDeterministicFixes(tmpDir, verify.FilePaths(result.Files)); fixes != "" {
			r.log("[%s] applied fixes: %s", r.task.ID, fixes)
		}

		// Run verification.
		vResult, err := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(result.Files))
		if err != nil {
			return fmt.Errorf("task %s: verifier error: %w", r.task.ID, err)
		}

		r.log("[%s] verify: passed=%v", r.task.ID, vResult.Passed)
		if r.verbose {
			r.log("%s", vResult.Output)
		}

		if vResult.Passed {
			return r.commitAndRecord(tmpDir, result.Files)
		}

		lastVerifyOutput = vResult.Output
	}

	return fmt.Errorf("task %s: exhausted %d retry attempts", r.task.ID, r.maxRetries)
}

func (r *TaskRunnerV2) commitAndRecord(tmpDir string, files []dag.GeneratedFile) error {
	if err := r.writer.Commit(tmpDir, files); err != nil {
		return fmt.Errorf("task %s: commit: %w", r.task.ID, err)
	}
	os.RemoveAll(tmpDir)
	r.memory.Record(r.task, files)
	r.state.MarkCompleted(r.task.ID)
	r.log("[%s] done (%d files)", r.task.ID, len(files))
	return nil
}

// findGoMod walks tmpDir to find a go.mod file.
func findGoMod(dir string) string {
	var result string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "go.mod" && result == "" {
			result = path
		}
		return nil
	})
	return result
}

func runGoModTidy(ctx context.Context, dir string) {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	cmd.CombinedOutput()
}
