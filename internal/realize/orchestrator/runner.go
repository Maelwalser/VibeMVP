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
	"github.com/vibe-menu/internal/realize/codegen"
	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/output"
	"github.com/vibe-menu/internal/realize/skills"
	"github.com/vibe-menu/internal/realize/state"
	"github.com/vibe-menu/internal/realize/verify"
)

// testOnlyRetryGuidance is prepended to the verification error output when only
// test files fail (go build passes). It instructs the LLM to fix test code only
// and not touch the working implementation files.
const testOnlyRetryGuidance = `⚠ TEST-ONLY FAILURE: go build PASSED — your implementation code is CORRECT.
ONLY the _test.go files have errors. Fix the test files ONLY.
DO NOT modify any non-test files (*.go without _test suffix) — they compile and work.
Return ALL files (implementation + fixed tests) in your output.

COMMON TEST MISTAKES TO FIX:
1. "for _, t := range tests" shadows *testing.T — use "for _, tc := range tests" and tc.field for struct fields
2. mock.ExpectQueryRow() does NOT exist — use mock.ExpectQuery() for both Query() and QueryRow()
3. pgx.PgError does NOT exist in pgx v5 — use pgconn.PgError from "github.com/jackc/pgx/v5/pgconn"
4. Use ONLY sentinel error names that exist in errors.go / domain files (check Shared Team Context)

`

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

// maxRetriesForKind overrides the default retry count for specific task kinds.
// Bootstrap and handler tasks are the most fragile (constructor wiring, auth
// middleware) and deserve more attempts. Boilerplate tasks (docs, contracts,
// docker) rarely need retries.
var maxRetriesForKind = map[dag.TaskKind]int{
	dag.TaskKindDataMigrations:   1,
	dag.TaskKindContracts:        1,
	dag.TaskKindInfraDocker:      1,
	dag.TaskKindInfraCI:          1,
	dag.TaskKindCrossCutDocs:     1,
	dag.TaskKindServiceBootstrap: 4, // most fragile — constructor wiring
	dag.TaskKindServiceHandler:   3, // auth middleware is tricky
	dag.TaskKindReconciliation:   3, // cross-task reasoning
}

// MaxRetriesFor returns the per-kind retry count if defined, otherwise the fallback.
func MaxRetriesFor(kind dag.TaskKind, fallback int) int {
	if n, ok := maxRetriesForKind[kind]; ok {
		return n
	}
	return fallback
}

// classifyError inspects verification output and returns the dominant error type.
// Uses the structured CompilerError parser for Go errors so classification is
// resilient to compiler output format changes across Go versions.
func classifyError(output string) errorType {
	parsed := verify.ParseGoErrors(output)

	// Check structured errors first — these are precise.
	if len(parsed) > 0 {
		if verify.HasCode(parsed, "redeclared") {
			return errTypeDuplicate
		}
		if verify.HasCode(parsed, "undefined") || verify.HasCode(parsed, "not_implemented") {
			return errTypeUndefined
		}
	}

	// Patterns that don't map to Go compiler errors (verifier-specific messages).
	switch {
	case strings.Contains(output, "unknown escape sequence"):
		return errTypeEscape
	case strings.Contains(output, "files not gofmt-clean") &&
		!verify.HasCode(parsed, "undefined") &&
		!strings.Contains(output, "FAIL"):
		return errTypeGofmt
	case strings.Contains(output, "missing go.sum entry") ||
		strings.Contains(output, "invalid version") ||
		strings.Contains(output, "cannot find module"):
		return errTypeDeps
	case strings.Contains(output, "--- FAIL:"):
		return errTypeTestFail
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
	// tierOverrides maps abstract ModelTier values to explicit model IDs from the
	// manifest's realize.tier_fast / tier_medium / tier_slow fields. When non-nil,
	// the override model ID is used instead of the default providerModels lookup.
	tierOverrides map[ModelTier]string
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

	// Augment the pre-computed deps context with the locked go.mod so the agent
	// sees the exact resolved versions and does not invent its own. This prevents
	// version drift between what go mod tidy resolved and what the agent writes.
	effectiveDepsContext := r.depsContext
	if lockedGoMod != "" {
		effectiveDepsContext += "\n## LOCKED DEPENDENCIES — DO NOT OVERRIDE\n\n"
		effectiveDepsContext += "The following go.mod was resolved by the package manager. "
		effectiveDepsContext += "Do NOT generate go.mod or go.sum. Do NOT change any version.\n\n"
		effectiveDepsContext += "```\n" + lockedGoMod + "\n```\n"
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
					_ = os.WriteFile(modPath, []byte(lockedGoMod), 0644)
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

		// Generate deterministic bootstrap skeleton for bootstrap tasks.
		var bootstrapSkeleton string
		if r.task.Kind == dag.TaskKindServiceBootstrap {
			modulePath := ""
			if r.task.Payload.ModulePath != "" {
				modulePath = r.task.Payload.ModulePath
			}
			framework := ""
			language := ""
			if r.task.Payload.Service != nil {
				framework = r.task.Payload.Service.Framework
				language = r.task.Payload.Service.Language
			}
			bootstrapSkeleton = codegen.WireBootstrap(
				r.memory.AllConstructors(),
				r.memory.AllServiceMethods(),
				modulePath,
				framework,
				language,
			)
		}

		ac := &agent.Context{
			Task:                 r.task,
			SkillDocs:            r.skillDocs,
			PreviousErrors:       lastVerifyOutput,
			DependencyOutputs:    r.memory.DepsOf(r.task),
			AttemptNumber:        attempt,
			DepsContext:          effectiveDepsContext,
			ExistingTypeRegistry: r.memory.TypeRegistry(),
			AllConstructors:      r.memory.AllConstructors(),
			AllServiceMethods:    r.memory.AllServiceMethods(),
			AllErrorSentinels:    r.memory.AllErrorSentinels(),
			InterfaceContracts:   r.memory.AllInterfaceContracts(),
			CrossTaskIssues:      r.memory.CrossTaskIssues(),
			BootstrapSkeleton:    bootstrapSkeleton,
			MaxContextTokens:     contextWindowForAttempt(r.providerAssignment, r.initialTier, r.tierOverrides, attempt),
		}

		result, err := a.Run(ctx, ac)
		if err != nil {
			r.log("[%s] agent error: %v", r.task.ID, err)
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

		// On retries, remove files from the previous attempt that are NOT in the
		// new output. This prevents stale files (with renamed/removed symbols) from
		// causing "undefined" errors when the retry LLM generates a different file set.
		if attempt > 0 && len(lastFiles) > 0 {
			newFileSet := make(map[string]bool, len(result.Files))
			for _, f := range result.Files {
				newFileSet[f.Path] = true
			}
			for _, old := range lastFiles {
				if !newFileSet[old.Path] {
					_ = os.Remove(filepath.Join(tmpDir, old.Path))
				}
			}
			// Re-stage dependency files — the stale cleanup above may have removed
			// files that a previous attempt overwrote (e.g. interfaces.go, errors.go)
			// and the new attempt doesn't regenerate.
			if err := r.stageDependencyFiles(tmpDir); err != nil {
				r.log("[%s] warning: re-staging dependency files: %v", r.task.ID, err)
			}
		}

		lastFiles = result.Files

		if err := r.writer.WriteAllTo(tmpDir, result.Files); err != nil {
			return fmt.Errorf("task %s: write to temp dir: %w", r.task.ID, err)
		}

		// Pre-verify structural check: catch wrong package declarations, empty
		// files, and duplicate paths before spending time on compilation.
		if structIssues := verify.ValidateStructure(result.Files, r.verifier.Language()); len(structIssues) > 0 {
			r.log("[%s] structural check: %d issue(s)", r.task.ID, len(structIssues))
			if attempt < r.maxRetries {
				lastVerifyOutput = "Structural validation errors (fix these before compilation):\n" + strings.Join(structIssues, "\n")
				lastFiles = result.Files
				continue
			}
			// On last attempt, log but proceed to verification anyway.
			for _, issue := range structIssues {
				r.log("[%s] structural: %s", r.task.ID, issue)
			}
		}

		// Restore locked go.mod if the agent overwrote it, then run go mod tidy
		// so the checksum database is up to date without inventing new versions.
		if lockedGoMod != "" {
			if modPath := findGoMod(tmpDir); modPath != "" {
				_ = os.WriteFile(modPath, []byte(lockedGoMod), 0644)
				runGoModTidy(ctx, filepath.Dir(modPath))
			}
		}

		// Apply deterministic fixes (language-specific formatting, import cleanup,
		// etc.) before every verification — not just on retries.
		// Use ALL files in tmpDir (including staged dependency files) so that fixes
		// like fixDuplicateTypes and fixGofmt cover the entire compilation unit.
		// IMPORTANT: collect files AFTER WriteAllTo so deterministic fixes operate
		// on the current agent output, not stale pre-write content.
		allFiles := allFilesInDir(tmpDir)
		if fixes := verify.ApplyDeterministicFixes(tmpDir, allFiles, r.verifier.Language()); fixes != "" {
			r.log("[%s] applied deterministic fixes: %s", r.task.ID, fixes)
		}

		// Run AST-based structural checks before the full build — catches import
		// consistency issues and constructor arity mismatches with targeted messages.
		if r.verifier.Language() == "go" {
			if importErrors := verify.CheckImportConsistency(tmpDir); len(importErrors) > 0 {
				r.log("[%s] AST import check: %d issue(s)", r.task.ID, len(importErrors))
			}
			if arityErrors := verify.CheckConstructorArity(tmpDir, r.memory.AllConstructors()); len(arityErrors) > 0 {
				r.log("[%s] AST arity check: %d issue(s): %s", r.task.ID, len(arityErrors), strings.Join(arityErrors, "; "))
			}
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
		} else if !vResult.Passed {
			// Print a brief error summary so users can diagnose failures without --verbose.
			if summary := briefVerifyErrors(vResult.Output); summary != "" {
				r.log("[%s] errors: %s", r.task.ID, summary)
			}
		}

		if vResult.Passed {
			// For data.schemas tasks, enforce domain completeness before committing.
			// If critical items are missing (domain files, attributes, Create*Input
			// structs), treat as a verification failure and continue to retry.
			if r.task.Kind == dag.TaskKindDataSchemas && attempt < r.maxRetries {
				if issues := r.validateDomainCompleteness(result.Files); len(issues) > 0 {
					r.log("[%s] domain completeness check failed (%d critical issues) — retrying", r.task.ID, len(issues))
					lastVerifyOutput = "Domain completeness errors (generated code compiles but is incomplete):\n" + strings.Join(issues, "\n")
					lastFiles = result.Files
					continue
				}
			}
			return r.commit(ctx, tmpDir, result.Files)
		}

		// Before consuming a retry slot, try disk-based error-driven fixes.
		// These parse the compiler output and apply mechanical corrections
		// without needing an LLM call.
		if attempt < r.maxRetries {
			anyDiskFix := false

			// Fix uuid.UUID → string type mismatches.
			if uuidFix := verify.ApplyUUIDToStringFixes(tmpDir, vResult.Output); uuidFix != "" {
				r.log("[%s] %s", r.task.ID, uuidFix)
				anyDiskFix = true
			}

			// Fix `:=` used where all vars are already declared.
			if declFix := verify.ApplyShortDeclFixes(tmpDir, vResult.Output); declFix != "" {
				r.log("[%s] %s", r.task.ID, declFix)
				anyDiskFix = true
			}

			// Fix constructor arity mismatches (too many/not enough arguments).
			if arityFix := verify.ApplyConstructorArityFixes(tmpDir, vResult.Output, r.memory.AllConstructors()); arityFix != "" {
				r.log("[%s] %s", r.task.ID, arityFix)
				anyDiskFix = true
			}

			if anyDiskFix {
				// Re-apply language fixes after rewriting (use all files to cover staged deps).
				verify.ApplyDeterministicFixes(tmpDir, allFilesInDir(tmpDir), r.verifier.Language())
				if fixResult, ferr := r.verifier.Verify(ctx, tmpDir, verify.FilePaths(lastFiles)); ferr == nil && fixResult.Passed {
					r.log("[%s] disk fix resolved verification — skipping LLM retry", r.task.ID)
					return r.commit(ctx, tmpDir, lastFiles)
				}
			}
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

		// When only test files fail, prepend guidance so the LLM fixes tests
		// without breaking the working implementation code.
		if isTestOnlyFailure(vResult.Output) {
			lastVerifyOutput = testOnlyRetryGuidance + lastVerifyOutput
		}
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
	// Apply output dir prefix to get disk-relative paths for rawPaths storage.
	// The un-prefixed files are passed to registerExportedTypes so the type registry
	// stores module-relative package paths (e.g. "internal/domain") rather than
	// filesystem paths (e.g. "backend/internal/domain") — preventing agents from
	// constructing wrong import paths like "backend/internal/domain" when the correct
	// Go import is "monolith/internal/domain".
	prefixedFiles := applyOutputDirPrefix(files, outputDir)
	// Keep the shared service temp dir alive until the bootstrap task completes
	// so each layer's files accumulate for go build verification.
	_, isSvcTask := serviceSlug(r.task.ID)
	if !isSvcTask || isBootstrapTask(r.task.ID) {
		if err := os.RemoveAll(tmpDir); err != nil {
			r.log("[%s] warning: failed to remove temp dir %s: %v", r.task.ID, tmpDir, err)
		}
	}
	// Record: passes prefixedFiles for rawPaths (disk staging) but Record internally
	// strips outputDir when building excerpts for agent context.
	r.memory.Record(r.task, prefixedFiles, outputDir)
	// For dependency resolution tasks, store the locked go.mod and go.sum in
	// shared memory so downstream tasks can find them even when temp dirs
	// don't match or when parallel microservice tasks need consistent checksums.
	if r.task.Kind == dag.TaskKindDependencyResolution {
		var modulePath string
		for _, f := range files {
			if filepath.Base(f.Path) == "go.mod" {
				modulePath = extractModulePath(f.Content)
				r.memory.StoreLockedGoMod(modulePath, f.Content)
			}
		}
		if modulePath != "" {
			for _, f := range files {
				if filepath.Base(f.Path) == "go.sum" {
					r.memory.StoreLockedGoSum(modulePath, f.Content)
					break
				}
			}
		}
	}

	// Register types and constructors from un-prefixed files so package paths are
	// module-relative. Constructors are extracted from full content here — before
	// any excerpt truncation — so downstream prompts see accurate signatures.
	r.registerExportedTypes(files)
	r.registerConstructors(files)
	r.registerServiceMethods(files)
	r.registerErrorSentinels(files)
	r.registerInterfaceContracts(files)
	// For data.schemas tasks, log any remaining completeness issues (advisory at
	// this point — critical issues were already enforced in the retry loop).
	if r.task.Kind == dag.TaskKindDataSchemas {
		_ = r.validateDomainCompleteness(files)
	}
	// Check that implementations satisfy interface contracts from shared memory.
	// Advisory only — logs warnings and records in CrossTaskIssues.
	if contracts := r.memory.AllInterfaceContracts(); len(contracts) > 0 {
		if issues := verify.CheckInterfaceCompliance(files, contracts); len(issues) > 0 {
			for _, issue := range issues {
				r.log("[%s] interface compliance: %s", r.task.ID, issue)
			}
			existing := r.memory.CrossTaskIssues()
			r.memory.SetCrossTaskIssues(existing + "\n" + strings.Join(issues, "\n"))
		}
	}

	// Incremental compilation: run go build in the output directory after each task
	// commits. If cross-task errors are found, record them as advisory context for
	// downstream tasks — but don't fail the pipeline (reconciliation handles repairs).
	r.runIncrementalBuild(ctx)
	if err := r.state.MarkCompleted(r.task.ID); err != nil {
		r.log("[%s] warning: failed to persist progress: %v", r.task.ID, err)
	}
	r.log("[%s] done (%d files)", r.task.ID, len(prefixedFiles))
	return nil
}

// registerExportedTypes scans the committed Go files for exported type declarations
// and records them in the shared type registry. Downstream tasks see this registry
// and are instructed not to redeclare the listed types — preventing the dual-interface
// pattern where two tasks independently define conflicting type aliases.
func (r *TaskRunner) registerExportedTypes(files []dag.GeneratedFile) {
	types := make(map[string]memory.TypeEntry)
	for _, f := range files {
		// Go types.
		for name, entry := range memory.ExtractGoExportedTypeNames(f.Path, f.Content) {
			types[name] = entry
		}
		// TypeScript and Python types.
		for name, entry := range memory.ExtractExportedTypeNames(f.Path, f.Content) {
			types[name] = entry
		}
	}
	if len(types) > 0 {
		r.memory.RegisterTypes(types)
	}
}

// registerConstructors extracts exported constructor and factory signatures from
// each committed file (at original, untruncated content) and stores them in
// shared memory. This ensures downstream tasks — especially bootstrap wiring — see
// accurate signatures even when file excerpts are truncated by the memory budget.
func (r *TaskRunner) registerConstructors(files []dag.GeneratedFile) {
	for _, f := range files {
		if sigs := memory.ExtractConstructorSigs(f.Path, f.Content); len(sigs) > 0 {
			r.memory.RegisterConstructors(f.Path, sigs)
		}
	}
}

// registerServiceMethods extracts exported method signatures (non-constructor) from
// each committed file and stores them in shared memory. This ensures handler/bootstrap
// tasks see accurate method signatures even when file excerpts are truncated.
func (r *TaskRunner) registerServiceMethods(files []dag.GeneratedFile) {
	for _, f := range files {
		if sigs := memory.ExtractServiceMethodSigs(f.Path, f.Content); len(sigs) > 0 {
			r.memory.RegisterServiceMethods(f.Path, sigs)
		}
	}
}

// registerInterfaceContracts extracts exported Go interface declarations from each
// committed file and stores them in shared memory. Downstream implementation tasks
// receive these as a hard checklist of method signatures to implement exactly.
func (r *TaskRunner) registerInterfaceContracts(files []dag.GeneratedFile) {
	for _, f := range files {
		if contracts := memory.ExtractGoInterfaceContracts(f.Path, f.Content); len(contracts) > 0 {
			r.memory.RegisterInterfaceContracts(contracts)
		}
	}
}

// registerErrorSentinels extracts exported error sentinels from each committed
// file (Go Err* vars, TypeScript exported error consts/classes, Python exception
// classes) and stores them in shared memory. This ensures downstream tasks see an
// explicit list of available sentinel names and do not invent new ones.
func (r *TaskRunner) registerErrorSentinels(files []dag.GeneratedFile) {
	for _, f := range files {
		if sentinels := memory.ExtractErrorSentinels(f.Path, f.Content); len(sentinels) > 0 {
			r.memory.RegisterErrorSentinels(sentinels)
		}
	}
}

// validateDomainCompleteness checks that the data.schemas output includes all domain
// attributes from the manifest and the input/update structs needed by repository
// operations. Returns critical issues (missing domain files, missing attributes,
// missing Create*Input structs) that should block the commit and trigger a retry.
// Non-critical issues (missing update structs) are logged as warnings only.
func (r *TaskRunner) validateDomainCompleteness(files []dag.GeneratedFile) []string {
	var critical []string

	// Build a content index by domain name using exact base filename matching.
	fileContent := make(map[string]string) // lowercase domain name → file content
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.Path))
		for _, domain := range r.task.Payload.Domains {
			expected := strings.ToLower(domain.Name) + ".go"
			if base == expected {
				fileContent[strings.ToLower(domain.Name)] = f.Content
			}
		}
	}

	for _, domain := range r.task.Payload.Domains {
		content, ok := fileContent[strings.ToLower(domain.Name)]
		if !ok {
			msg := fmt.Sprintf("no file generated for domain %q — expected %s.go", domain.Name, strings.ToLower(domain.Name))
			critical = append(critical, msg)
			r.log("[%s] CRITICAL: %s", r.task.ID, msg)
			continue
		}
		// Check each attribute is present as a Go struct field declaration.
		// Match "FieldName " (field name followed by a space/tab) to avoid false
		// positives from substring matches in comments or other identifiers.
		for _, attr := range domain.Attributes {
			goField := snakeToPascal(attr.Name)
			if !strings.Contains(content, goField+"\t") &&
				!strings.Contains(content, goField+" ") {
				msg := fmt.Sprintf("domain %q missing attribute %q (expected Go field %q)", domain.Name, attr.Name, goField)
				critical = append(critical, msg)
				r.log("[%s] CRITICAL: %s", r.task.ID, msg)
			}
		}
	}

	// Check that input/update structs for repository operations were generated.
	// Look for "type <TypeName> struct" declarations to avoid substring false positives.
	var sb strings.Builder
	for _, f := range files {
		sb.WriteString(f.Content)
		sb.WriteByte('\n')
	}
	allContent := sb.String()
	for _, svc := range r.task.Payload.AllServices {
		for _, repo := range svc.Repositories {
			for _, op := range repo.Operations {
				var typeName string
				isCritical := false
				switch {
				case op.OpType == "insert" || strings.HasPrefix(op.Name, "Create"):
					typeName = "Create" + repo.EntityRef + "Input"
					isCritical = true // Create inputs are critical — repos need them
				case op.OpType == "update":
					typeName = op.Name + "Input"
					// Update inputs are advisory — not blocking
				}
				if typeName != "" && !strings.Contains(allContent, "type "+typeName+" struct") {
					if isCritical {
						msg := fmt.Sprintf("missing input struct %q for %s.%s operation", typeName, repo.Name, op.Name)
						critical = append(critical, msg)
						r.log("[%s] CRITICAL: %s", r.task.ID, msg)
					} else {
						r.log("[%s] WARNING: missing input struct %q for %s.%s operation",
							r.task.ID, typeName, repo.Name, op.Name)
					}
				}
			}
		}
	}

	return critical
}

// snakeToPascal converts a snake_case string to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		// Handle common acronyms.
		upper := strings.ToUpper(p)
		switch upper {
		case "ID", "URL", "API", "HTTP", "UUID", "IP", "SQL", "DB":
			b.WriteString(upper)
		default:
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return b.String()
}

// readLockedGoMod reads the go.mod produced by a prior phase (plan or deps resolution)
// from the shared service temp directory. Returns "" for non-implementation tasks
// or when no prior go.mod exists.
//
// For cross-service backend tasks (auth, messaging, gateway) that are not part of a
// service chain, the locked go.mod is looked up from the monolith service chain's
// temp directory (svc.monolith) as these tasks share the same Go module.
func (r *TaskRunner) readLockedGoMod(tmpDir string) string {
	if !isImplementationTask(r.task.ID) {
		return ""
	}
	// Try the task's own temp dir first (service chain tasks).
	data, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err == nil {
		return string(data)
	}
	// For cross-service backend tasks, look in the monolith service chain's temp dir.
	if _, ok := serviceSlug(r.task.ID); !ok {
		monolithDir := filepath.Join(r.writer.BaseDir(), ".tmp", "svc.monolith")
		if data, err := os.ReadFile(filepath.Join(monolithDir, "go.mod")); err == nil {
			return string(data)
		}
	}
	// Fallback: check the output directory where the deps task committed go.mod.
	outputDir := r.task.Payload.OutputDir
	if outputDir != "" && outputDir != "." {
		outPath := filepath.Join(r.writer.BaseDir(), outputDir, "go.mod")
		if data, err := os.ReadFile(outPath); err == nil {
			return string(data)
		}
	}
	// Final fallback: check shared memory for any locked go.mod.
	if content := r.memory.AnyLockedGoMod(); content != "" {
		return content
	}
	return ""
}

// contextWindowForAttempt computes the context window size for the model that
// will be used at the given attempt (accounting for tier escalation).
func contextWindowForAttempt(pa manifest.ProviderAssignment, initialTier ModelTier, tierOverrides map[ModelTier]string, attempt int) int {
	tier := initialTier
	for i := 0; i < attempt; i++ {
		next, _ := escalateTier(tier)
		tier = next
	}
	if tierOverrides != nil {
		if modelID, ok := tierOverrides[tier]; ok && modelID != "" {
			return ContextWindowFor(modelID)
		}
	}
	modelID := resolveModelIDForTier(pa.Provider, tier, pa.Version)
	return ContextWindowFor(modelID)
}

// extractModulePath parses a go.mod content string and returns the module path.
func extractModulePath(goModContent string) string {
	for _, line := range strings.Split(goModContent, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
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

	// Restore locked go.sum from shared memory if no go.sum exists in tmpDir.
	// This prevents checksum drift between parallel microservice deps tasks.
	goSumPath := filepath.Join(tmpDir, "go.sum")
	if _, err := os.Stat(goSumPath); os.IsNotExist(err) {
		if goSum := r.memory.AnyLockedGoSum(); goSum != "" {
			_ = os.WriteFile(goSumPath, []byte(goSum), 0o644)
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

// allFilesInDir collects all file paths in dir (relative to dir) so that
// deterministic fixes can operate on staged dependency files in addition to the
// current task's own generated files. Without this, fixes like fixDuplicateTypes
// and fixGofmt miss staged files, causing verification failures.
func allFilesInDir(dir string) []string {
	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files
}

// runIncrementalBuild runs `go build ./...` in the output directory after a task
// commits to detect cross-task compilation errors early. Errors are stored as
// advisory context in shared memory — they don't block the pipeline.
func (r *TaskRunner) runIncrementalBuild(ctx context.Context) {
	outputDir := r.writer.BaseDir()
	// Only run for Go tasks that produce compilable code.
	if r.verifier.Language() != "go" {
		return
	}
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = outputDir
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) > 0 {
		// Trim to first few errors to keep context manageable.
		lines := strings.Split(string(out), "\n")
		if len(lines) > 20 {
			lines = lines[:20]
		}
		issues := strings.Join(lines, "\n")
		r.memory.SetCrossTaskIssues(issues)
		r.log("[%s] incremental build: %d cross-task issue(s) detected (advisory)", r.task.ID, len(lines))
	}
}

// isImplementationTask reports whether the task is an implementation layer that
// should not regenerate the project's go.mod. Includes service chain layers AND
// cross-service backend tasks (auth, messaging, gateway) that share the same module.
func isImplementationTask(taskID string) bool {
	return strings.HasSuffix(taskID, ".repository") ||
		strings.HasSuffix(taskID, ".service") ||
		strings.HasSuffix(taskID, ".handler") ||
		strings.HasSuffix(taskID, ".bootstrap") ||
		taskID == "backend.auth" ||
		taskID == "backend.messaging" ||
		taskID == "backend.gateway"
}

// findGoMod walks dir to find the first go.mod file.
func findGoMod(dir string) string {
	var result string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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
	_, _ = cmd.CombinedOutput()
}

// agentForAttempt returns the agent to use for a given attempt number.
// Attempt 0 reuses the pre-built r.agent. Each subsequent retry escalates the
// ModelTier (TierFast → TierMedium → TierSlow) and rebuilds the agent, respecting
// any explicit tier model overrides from the manifest.
func (r *TaskRunner) agentForAttempt(attempt int) agent.Agent {
	if attempt == 0 {
		return r.agent
	}
	tier := r.initialTier
	for i := 0; i < attempt; i++ {
		next, _ := escalateTier(tier)
		tier = next
	}
	// Use explicit override model ID when present.
	if r.tierOverrides != nil {
		if modelID, ok := r.tierOverrides[tier]; ok && modelID != "" {
			if r.verbose {
				r.log("[%s] escalating to %s (%s override) for attempt %d", r.task.ID, modelID, r.providerAssignment.Provider, attempt)
			}
			return buildAgentWithModel(r.providerAssignment, modelID, defaultMaxTokens, thinkingBudgetForTier(tier), reasoningEffortForTier(tier), r.verbose)
		}
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

// isTestOnlyFailure checks whether the verification output indicates that
// go build and go vet passed but go test failed. In this case, the implementation
// code is correct — only the test files have issues (typically hallucinated APIs).
func isTestOnlyFailure(output string) bool {
	// go build must have passed (no errors after "=== go build" section).
	// go test must have failed.
	hasBuildPass := true
	hasTestFail := false
	hasVetFail := false

	sections := strings.Split(output, "=== ")
	for _, section := range sections {
		if strings.HasPrefix(section, "go build") {
			// Check if there are actual errors (non-empty after the header line).
			lines := strings.SplitN(section, "\n", 2)
			if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
				hasBuildPass = false
			}
		}
		if strings.HasPrefix(section, "go vet") {
			lines := strings.SplitN(section, "\n", 2)
			if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
				hasVetFail = true
			}
		}
		if strings.HasPrefix(section, "go test") {
			if strings.Contains(section, "FAIL") {
				hasTestFail = true
			}
		}
	}

	// go vet errors in test files also count as test-only failures since
	// vet runs on test files too.
	if hasVetFail && !hasTestFail {
		// Check if all vet errors are in _test.go files.
		vetOnlyInTests := true
		for _, section := range sections {
			if !strings.HasPrefix(section, "go vet") {
				continue
			}
			for _, line := range strings.Split(section, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "vet:") {
					// Check if vet error references a test file.
					if strings.Contains(trimmed, "_test.go:") || strings.Contains(trimmed, "_test.go ") {
						continue
					}
					if strings.Contains(trimmed, ".go:") && !strings.Contains(trimmed, "_test.go") {
						vetOnlyInTests = false
					}
				}
			}
		}
		if vetOnlyInTests {
			hasTestFail = true // treat vet-only-in-tests as test-only failure
		}
	}

	return hasBuildPass && hasTestFail
}

// briefVerifyErrors extracts the first few unique error messages from verification
// output for display in non-verbose mode. Returns at most 3 distinct error patterns
// joined by "; ", or empty string if no recognizable errors are found.
func briefVerifyErrors(output string) string {
	seen := make(map[string]bool)
	var errors []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		// Go compiler errors: "file.go:line:col: message"
		if idx := strings.Index(trimmed, ".go:"); idx >= 0 {
			// Extract just the error message after the file:line:col prefix.
			parts := strings.SplitN(trimmed[idx:], ": ", 2)
			if len(parts) == 2 {
				msg := parts[1]
				if !seen[msg] && len(errors) < 3 {
					seen[msg] = true
					errors = append(errors, msg)
				}
			}
		}
	}
	return strings.Join(errors, "; ")
}

// isRateLimitError reports whether err is an API 429 rate-limit response.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate_limit_error")
}
