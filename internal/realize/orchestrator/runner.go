package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/config"
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
type TaskRunner struct {
	task       *dag.Task
	agent      agent.Agent
	verifier   verify.Verifier
	writer     *output.Writer
	state      *state.Store
	memory     *memory.SharedMemory
	skillDocs  []skills.Doc
	maxRetries int
	verbose    bool
	logFn      func(string) // optional; nil falls back to os.Stderr
	// providerAssignment is the provider config for this task (Provider, Credential, Version).
	// For the default Claude path, Provider="Claude" with empty Credential (uses env var).
	providerAssignment manifest.ProviderAssignment
	// initialTier is the baseline model tier for this task (from tierForKind).
	// Each retry escalates: TierFast → TierMedium → TierSlow, regardless of provider.
	initialTier ModelTier
	// depsContext is pre-computed dependency & API reference text injected into
	// the system prompt to prevent module version hallucination.
	depsContext string
}

func (r *TaskRunner) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if r.logFn != nil {
		r.logFn(msg)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// writeDebugLog appends a structured attempt record to .realize/debug/<task-id>.log.
// Failures are logged always; successes only when verbose is on.
func (r *TaskRunner) writeDebugLog(attempt int, passed bool, output string) {
	dir := filepath.Join(r.writer.BaseDir(), ".realize", "debug")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	logPath := filepath.Join(dir, r.task.ID+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "=== attempt %d | %s | passed=%v ===\n%s\n",
		attempt, time.Now().Format(time.RFC3339), passed, output)
}

// Run executes the task, retrying up to maxRetries times on verification failure.
// On each retry the previous verification output is fed back to the agent.
// When baseModel is set, each attempt escalates through Haiku → Sonnet → Opus.
//
// Each attempt applies deterministic fixes (gofmt, escape sequences, duplicate type
// removal) before running the verifier, so common LLM mistakes are corrected without
// burning a retry slot. On retry, errors are classified and mechanically-fixable issues
// are resolved without an LLM call.
func (r *TaskRunner) Run(ctx context.Context) error {
	var lastVerifyOutput string
	var lastFiles []dag.GeneratedFile // files from the most recent agent call

	// Determine shared temp directory. Service-chain tasks share a single directory
	// so each layer accumulates files for go build (go.mod, interfaces, impls).
	var tmpDir string
	if slug, ok := serviceSlug(r.task.ID); ok {
		tmpDir = filepath.Join(r.writer.BaseDir(), ".tmp", "svc."+slug)
	} else {
		tmpDir = filepath.Join(r.writer.BaseDir(), ".tmp", r.task.ID)
	}

	// Read the locked go.mod produced by the deps resolution task (if any).
	// Implementation layers must not overwrite it with a freshly-hallucinated one.
	lockedGoMod := r.readLockedGoMod(tmpDir)
	if lockedGoMod != "" {
		r.log("[%s] go.mod locked from deps phase", r.task.ID)
	}

	// Stage files from completed dependency tasks so go build can resolve
	// cross-task packages (e.g. internal/domain from data.schemas is visible
	// to svc.monolith.plan's verifier without burning a retry slot).
	if err := r.stageDependencyFiles(tmpDir); err != nil {
		r.log("[%s] warning: staging dependency files: %v", r.task.ID, err)
	}

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			r.log("[%s] retry %d/%d", r.task.ID, attempt, r.maxRetries)

			// On retry: classify the error and try mechanically-fixable issues first
			// before spending an LLM call.
			if classifyError(lastVerifyOutput) == errTypeDeps && lockedGoMod != "" {
				if modPath := findGoMod(tmpDir); modPath != "" {
					os.WriteFile(modPath, []byte(lockedGoMod), 0644)
					runGoModTidy(ctx, filepath.Dir(modPath))
					if vr, err := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(lastFiles)); err == nil && vr.Passed {
						r.log("[%s] deps fix resolved verification — skipping LLM retry", r.task.ID)
						return r.commit(ctx, tmpDir, lastFiles)
					}
				}
			}
		}

		// Resolve agent for this attempt: escalate model tier on retries when
		// a baseModel is set (i.e. when using the default Claude agent, not a
		// per-section manifest override which may be a different provider).
		a := r.agentForAttempt(attempt)

		ac := &agent.Context{
			Task:                 r.task,
			SkillDocs:            r.skillDocs,
			PreviousErrors:       lastVerifyOutput,
			DependencyOutputs:    r.memory.DepsOf(r.task),
			AttemptNumber:        attempt,
			DepsContext:          r.depsContext,
			ExistingTypeRegistry: r.memory.TypeRegistry(),
		}

		result, err := a.Run(ctx, ac)
		if err != nil {
			if attempt == r.maxRetries {
				return fmt.Errorf("task %s: agent failed after %d attempts: %w", r.task.ID, attempt+1, err)
			}
			if isRateLimitError(err) {
				wait := time.Duration(attempt+1) * config.RateLimitBackoffBase * time.Second
				r.log("[%s] rate limited — waiting %s before retry", r.task.ID, wait)
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

		if err := r.writer.WriteAllTo(tmpDir, result.Files); err != nil {
			return fmt.Errorf("task %s: write to temp dir: %w", r.task.ID, err)
		}

		// Restore locked go.mod if the agent overwrote it, then run go mod tidy
		// so the checksum database is up to date without inventing new versions.
		if lockedGoMod != "" {
			if modPath := findGoMod(tmpDir); modPath != "" {
				os.WriteFile(modPath, []byte(lockedGoMod), 0644)
				runGoModTidy(ctx, filepath.Dir(modPath))
			}
		}

		// Apply deterministic fixes (gofmt, invalid escape sequences, duplicate
		// type declarations) before every verification — not just on retries.
		if fixes := verify.ApplyDeterministicFixes(tmpDir, verify.FilePaths(result.Files)); fixes != "" {
			r.log("[%s] applied deterministic fixes: %s", r.task.ID, fixes)
		}

		// Run verification.
		vResult, verifyErr := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(result.Files))
		if verifyErr != nil {
			return fmt.Errorf("task %s: verifier error: %w", r.task.ID, verifyErr)
		}

		r.log("[%s] verify: passed=%v", r.task.ID, vResult.Passed)
		if r.verbose || !vResult.Passed {
			r.writeDebugLog(attempt, vResult.Passed, vResult.Output)
		}
		if r.verbose {
			r.log("%s", vResult.Output)
		}

		if vResult.Passed {
			return r.commit(ctx, tmpDir, result.Files)
		}

		// Before consuming a retry slot, try the in-memory fix layer (unused
		// import removal + gofmt via go/format). This catches issues that the
		// disk-based fixes above miss (e.g. imports added after file was written).
		if attempt < r.maxRetries {
			if fixed, fixedFiles := verify.TryFix(result.Files, vResult.Output); fixed {
				r.log("[%s] applying in-memory fixes before retry", r.task.ID)
				if err := r.writer.WriteAllTo(tmpDir, fixedFiles); err == nil {
					fixResult, ferr := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(fixedFiles))
					if ferr == nil && fixResult.Passed {
						r.log("[%s] in-memory fix resolved verification — skipping LLM retry", r.task.ID)
						return r.commit(ctx, tmpDir, fixedFiles)
					}
					if ferr == nil {
						vResult = fixResult
						lastFiles = fixedFiles
					}
				}
			}
		}

		lastVerifyOutput = vResult.Output
	}

	return fmt.Errorf("task %s: exhausted %d retry attempts", r.task.ID, r.maxRetries)
}

// commit writes files to the output directory, publishes to shared memory,
// and marks the task as completed.
func (r *TaskRunner) commit(ctx context.Context, tmpDir string, files []dag.GeneratedFile) error {
	outputDir := r.task.Payload.OutputDir
	if err := r.writer.CommitWithPrefix(tmpDir, outputDir, files); err != nil {
		return fmt.Errorf("task %s: commit files: %w", r.task.ID, err)
	}
	// Record with prefixed paths so downstream tasks can locate files on disk.
	files = applyOutputDirPrefix(files, outputDir)
	// Keep the shared service temp dir alive until the bootstrap task completes
	// so each layer's files accumulate for go build verification.
	_, isSvcTask := serviceSlug(r.task.ID)
	if !isSvcTask || isBootstrapTask(r.task.ID) {
		if err := os.RemoveAll(tmpDir); err != nil {
			r.log("[%s] warning: failed to remove temp dir %s: %v", r.task.ID, tmpDir, err)
		}
	}
	r.memory.Record(r.task, files)
	r.registerExportedTypes(files)
	if err := r.state.MarkCompleted(r.task.ID); err != nil {
		r.log("[%s] warning: failed to persist progress: %v", r.task.ID, err)
	}
	r.log("[%s] done (%d files)", r.task.ID, len(files))
	return nil
}

// registerExportedTypes scans the committed Go files for exported type declarations
// and records them in the shared type registry. Downstream tasks see this registry
// and are instructed not to redeclare the listed types — preventing the dual-interface
// pattern where two tasks independently define conflicting type aliases.
func (r *TaskRunner) registerExportedTypes(files []dag.GeneratedFile) {
	types := make(map[string]memory.TypeEntry)
	for _, f := range files {
		for name, entry := range memory.ExtractGoExportedTypeNames(f.Path, f.Content) {
			types[name] = entry
		}
	}
	if len(types) > 0 {
		r.memory.RegisterTypes(types)
	}
}

// readLockedGoMod reads the go.mod produced by a prior phase (plan or deps resolution)
// from the shared service temp directory. Returns "" for non-implementation tasks
// or when no prior go.mod exists.
func (r *TaskRunner) readLockedGoMod(tmpDir string) string {
	if !isImplementationTask(r.task.ID) {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		return ""
	}
	return string(data)
}

// stageDependencyFiles copies files committed by upstream tasks into tmpDir so
// the verifier (go build) can resolve cross-task packages without an LLM retry.
// Files that already exist in tmpDir are skipped — earlier layers take precedence.
// Memory stores prefixed output paths (e.g. "backend/internal/domain/user.go");
// this function strips the service-dir prefix so temp dir stays prefix-free.
func (r *TaskRunner) stageDependencyFiles(tmpDir string) error {
	paths := r.memory.CommittedPaths(r.task.Dependencies)
	for _, p := range paths {
		// Strip any service-dir prefix — temp dir is always component-relative.
		stripped := stripOutputDirPrefix(p, r.task.Payload.ServiceDirs)
		dst := filepath.Join(tmpDir, stripped)
		if _, err := os.Stat(dst); err == nil {
			continue // already present — don't overwrite
		}
		src := filepath.Join(r.writer.BaseDir(), p) // read from prefixed output location
		data, err := os.ReadFile(src)
		if err != nil {
			continue // source may not exist (e.g. non-Go task with no-op verifier)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// applyOutputDirPrefix prepends outputDir to every file path.
// Returns the original slice unchanged if outputDir is empty or ".".
func applyOutputDirPrefix(files []dag.GeneratedFile, outputDir string) []dag.GeneratedFile {
	if outputDir == "" || outputDir == "." {
		return files
	}
	result := make([]dag.GeneratedFile, len(files))
	for i, f := range files {
		result[i] = dag.GeneratedFile{Path: filepath.Join(outputDir, f.Path), Content: f.Content}
	}
	return result
}

// stripOutputDirPrefix removes a leading service-dir prefix from path so it can
// be written to a prefix-free temp directory for verification.
// E.g. "backend/internal/domain/user.go" → "internal/domain/user.go"
func stripOutputDirPrefix(path string, serviceDirs map[string]string) string {
	normalized := filepath.ToSlash(path)
	for _, dir := range serviceDirs {
		if dir == "" || dir == "." {
			continue
		}
		prefix := dir + "/"
		if strings.HasPrefix(normalized, prefix) {
			return strings.TrimPrefix(normalized, prefix)
		}
	}
	return path
}

// isImplementationTask reports whether the task is an implementation layer that
// should not regenerate the project's go.mod.
func isImplementationTask(taskID string) bool {
	return strings.HasSuffix(taskID, ".repository") ||
		strings.HasSuffix(taskID, ".service") ||
		strings.HasSuffix(taskID, ".handler") ||
		strings.HasSuffix(taskID, ".bootstrap")
}

// findGoMod walks dir to find the first go.mod file.
func findGoMod(dir string) string {
	var result string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || result != "" {
			return nil
		}
		if info.Name() == "go.mod" {
			result = path
		}
		return nil
	})
	return result
}

// runGoModTidy runs go mod tidy in dir, silently ignoring failures (the verifier
// will surface any remaining issues on the next verification pass).
func runGoModTidy(ctx context.Context, dir string) {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	cmd.CombinedOutput()
}

// agentForAttempt returns the agent to use for a given attempt number.
// Attempt 0 reuses the pre-built r.agent. Each subsequent retry escalates the
// ModelTier (TierFast → TierMedium → TierSlow) and rebuilds the agent via
// buildAgentForTier — which works for any configured provider, not just Claude.
func (r *TaskRunner) agentForAttempt(attempt int) agent.Agent {
	if attempt == 0 {
		return r.agent
	}
	tier := r.initialTier
	for i := 0; i < attempt; i++ {
		next, _ := escalateTier(tier)
		tier = next
	}
	if r.verbose {
		model := resolveModelIDForTier(r.providerAssignment.Provider, tier, r.providerAssignment.Version)
		r.log("[%s] escalating to %s (%s) for attempt %d", r.task.ID, model, r.providerAssignment.Provider, attempt)
	}
	return buildAgentForTier(r.providerAssignment, tier, defaultMaxTokens, r.verbose)
}

// serviceSlug extracts the service slug from a task ID of the form "svc.<slug>.<layer>".
// Returns ("", false) for tasks that are not part of a service chain.
func serviceSlug(taskID string) (string, bool) {
	parts := strings.SplitN(taskID, ".", 3)
	if len(parts) == 3 && parts[0] == "svc" {
		return parts[1], true
	}
	return "", false
}

// isBootstrapTask reports whether the task is the final layer in a service chain.
// Only the bootstrap task should clean up the shared service temp directory.
func isBootstrapTask(taskID string) bool {
	return strings.HasSuffix(taskID, ".bootstrap")
}

// isRateLimitError reports whether err is an API 429 rate-limit response.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate_limit_error")
}
