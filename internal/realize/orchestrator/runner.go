package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibe-mvp/internal/realize/agent"
	"github.com/vibe-mvp/internal/realize/dag"
	"github.com/vibe-mvp/internal/realize/output"
	"github.com/vibe-mvp/internal/realize/skills"
	"github.com/vibe-mvp/internal/realize/state"
	"github.com/vibe-mvp/internal/realize/verify"
)

// TaskRunner handles one task's agent invocation + verification retry loop.
type TaskRunner struct {
	task       *dag.Task
	agent      agent.Agent
	verifier   verify.Verifier
	writer     *output.Writer
	state      *state.Store
	skillDocs  []skills.Doc
	maxRetries int
	verbose    bool
}

// Run executes the task, retrying up to maxRetries times on verification failure.
// On each retry, the previous verification output is fed back to the agent.
func (r *TaskRunner) Run(ctx context.Context) error {
	var lastVerifyOutput string

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Fprintf(os.Stderr, "[%s] retry %d/%d\n", r.task.ID, attempt, r.maxRetries)
		}

		ac := &agent.Context{
			Task:           r.task,
			SkillDocs:      r.skillDocs,
			PreviousErrors: lastVerifyOutput,
		}

		result, err := r.agent.Run(ctx, ac)
		if err != nil {
			if attempt == r.maxRetries {
				return fmt.Errorf("task %s: agent failed after %d attempts: %w", r.task.ID, attempt+1, err)
			}
			if isRateLimitError(err) {
				wait := time.Duration(attempt+1) * 60 * time.Second
				fmt.Fprintf(os.Stderr, "[%s] rate limited — waiting %s before retry\n", r.task.ID, wait)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			lastVerifyOutput = fmt.Sprintf("Agent error: %v", err)
			continue
		}

		// Write to a task-scoped temp directory.
		tmpDir := filepath.Join(r.writer.BaseDir(), ".tmp", r.task.ID)
		if err := r.writer.WriteAllTo(tmpDir, result.Files); err != nil {
			return fmt.Errorf("task %s: write to temp dir: %w", r.task.ID, err)
		}

		// Run verification.
		vResult, err := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(result.Files))
		if err != nil {
			return fmt.Errorf("task %s: verifier error: %w", r.task.ID, err)
		}

		if r.verbose {
			fmt.Fprintf(os.Stderr, "[%s] verify: passed=%v\n%s\n", r.task.ID, vResult.Passed, vResult.Output)
		}

		if vResult.Passed {
			if err := r.writer.Commit(tmpDir, result.Files); err != nil {
				return fmt.Errorf("task %s: commit files: %w", r.task.ID, err)
			}
			if err := os.RemoveAll(tmpDir); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] warning: failed to remove temp dir %s: %v\n", r.task.ID, tmpDir, err)
			}
			if err := r.state.MarkCompleted(r.task.ID); err != nil {
				// Non-fatal: files are committed; losing the progress marker just
				// means this task might re-run on the next resume.
				fmt.Fprintf(os.Stderr, "[%s] warning: failed to persist progress: %v\n", r.task.ID, err)
			}
			fmt.Fprintf(os.Stderr, "[%s] done (%d files)\n", r.task.ID, len(result.Files))
			return nil
		}

		lastVerifyOutput = vResult.Output
	}

	return fmt.Errorf("task %s: exhausted %d retry attempts", r.task.ID, r.maxRetries)
}

// isRateLimitError reports whether err is an API 429 rate-limit response.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate_limit_error")
}
