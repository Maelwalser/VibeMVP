package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/deps"
	"github.com/vibe-menu/internal/realize/memory"
	"github.com/vibe-menu/internal/realize/output"
	"github.com/vibe-menu/internal/realize/skills"
	"github.com/vibe-menu/internal/realize/state"
	"github.com/vibe-menu/internal/realize/verify"
)

const (
	defaultModel     = config.DefaultModel
	defaultMaxTokens = config.DefaultMaxTokens
)

// Config holds all runtime configuration for the orchestrator.
type Config struct {
	ManifestPath string
	OutputDir    string
	SkillsDir    string
	MaxRetries   int
	Parallelism  int
	DryRun       bool
	Verbose      bool
	// Provider overrides the default LLM provider for all tasks that have no
	// per-section manifest assignment (e.g. "Gemini", "ChatGPT", "Mistral").
	// Ignored when empty — falls back to Claude via the ANTHROPIC_API_KEY env var.
	Provider string
	// APIKey is the credential for Provider. When empty, the standard env var
	// for that provider is tried (GEMINI_API_KEY, OPENAI_API_KEY, etc.).
	APIKey string
	// LogFunc, if non-nil, receives status lines instead of os.Stderr.
	LogFunc func(string)
}

// log emits a formatted status line via LogFunc or os.Stderr.
func (o *Orchestrator) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.cfg.LogFunc != nil {
		o.cfg.LogFunc(msg)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// Orchestrator drives the full DAG-based code generation pipeline.
type Orchestrator struct {
	cfg Config
}

// New returns a configured Orchestrator.
func New(cfg Config) *Orchestrator {
	return &Orchestrator{cfg: cfg}
}

// providerEnvVars maps provider names to their conventional API key env vars.
var providerEnvVars = map[string]string{
	"Claude":  "ANTHROPIC_API_KEY",
	"ChatGPT": "OPENAI_API_KEY",
	"Gemini":  "GEMINI_API_KEY",
	"Mistral": "MISTRAL_API_KEY",
	"Llama":   "GROQ_API_KEY",
}

// resolveDefaultProvider returns the ProviderAssignment to use for tasks that
// have no per-section manifest override.
// Priority: --provider + --api-key flags → env vars → default Claude (env-var key).
func (o *Orchestrator) resolveDefaultProvider() manifest.ProviderAssignment {
	provider := o.cfg.Provider
	if provider == "" {
		provider = "Claude"
	}
	key := o.cfg.APIKey
	if key == "" {
		if envVar, ok := providerEnvVars[provider]; ok {
			key = os.Getenv(envVar)
		}
	}
	return manifest.ProviderAssignment{Provider: provider, Credential: key}
}

// Run loads the manifest, builds the DAG, and executes all tasks.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Load and parse manifest.
	m, err := loadManifest(o.cfg.ManifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Build execution DAG.
	d, err := (&dag.Builder{}).Build(m)
	if err != nil {
		return fmt.Errorf("build dag: %w", err)
	}

	// Build provider assignments from the new manifest structure.
	providers := buildProviderAssignments(m)

	// Print plan in dry-run mode.
	if o.cfg.DryRun {
		return o.printPlan(d, providers)
	}

	// Load skill registry.
	reg, err := skills.Load(o.cfg.SkillsDir)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	// Set up output writer.
	writer, err := output.New(o.cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("create output writer: %w", err)
	}

	// Load (or create) the progress state for resume support.
	st, err := state.Load(o.cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("load progress state: %w", err)
	}
	if n := st.CompletedCount(); n > 0 {
		o.log("realize: resuming — %d task(s) already completed, skipping them", n)
	}

	// Set up verifier registry and shared memory.
	verifiers := verify.NewRegistry()
	mem := memory.New()

	// defaultProvider is used for tasks that have no per-section manifest override.
	// Priority: --provider flag → env vars → Claude (uses ANTHROPIC_API_KEY).
	defaultProvider := o.resolveDefaultProvider()

	// Resolve the minimum Go runtime version from the Go module proxy once here
	// so every task — plan, infra, frontend — uses the same consistent version.
	// The result flows into go.mod (via plan task prompt) and Dockerfiles (via
	// infra task prompt), preventing mismatches between the two.
	resolvedGoVersion := deps.ResolveGoVersion(ctx)
	if resolvedGoVersion != "" {
		o.log("realize: resolved Go version %s from dev tool requirements", resolvedGoVersion)
	}

	// Resolve all Go library module versions from the Go module proxy once at
	// startup. Every service task prompt then uses live versions instead of
	// hardcoded fallbacks, preventing version staleness across all frameworks,
	// drivers, and utility libraries.
	resolvedGoModules := deps.ResolveGoModuleVersions(ctx)
	o.log("realize: resolved %d Go library versions from module proxy", len(resolvedGoModules))

	// Log configured per-section model assignments.
	for sectionID, pa := range providers {
		if pa.Credential != "" {
			fmt.Fprintf(os.Stderr, "realize: section %q → %s %s %s\n",
				sectionID, pa.Provider, pa.Model, pa.Version)
		}
	}

	fmt.Fprintf(os.Stderr, "realize: starting %d tasks across %d wave(s)\n",
		len(d.Tasks), len(d.Levels()))

	// Execute waves in order; tasks within each wave run in parallel.
	for waveIdx, wave := range d.Levels() {
		o.log("realize: wave %d (%d tasks): %v", waveIdx, len(wave), wave)

		if err := o.runWave(ctx, wave, d, providers, reg, defaultProvider, verifiers, writer, st, mem, resolvedGoVersion, resolvedGoModules); err != nil {
			return fmt.Errorf("wave %d: %w", waveIdx, err)
		}
	}

	// Run a project-wide integration build after all tasks complete.
	// This catches cross-task compilation errors (import path mismatches, type
	// field access on wrong struct, missing multi-return handling) that per-task
	// verification misses because each task is verified in isolation.
	// Non-blocking: failures are logged with a targeted summary but do not abort
	// the pipeline — the user receives all generated code plus a clear diagnostic.
	o.log("realize: running integration build across all output files...")
	intResult := verify.RunIntegrationBuild(ctx, o.cfg.OutputDir)
	if intResult.Passed {
		o.log("realize: integration build passed ✓ — all generated code compiles together")
	} else {
		o.log("realize: ⚠ integration build found cross-task errors:\n%s", intResult.Output)
		o.log("realize: NOTE — the above errors were not caught by per-task verification because")
		o.log("realize:       tasks are verified in isolation. Common causes:")
		o.log("realize:       1. Wrong import paths — check that all internal imports use module path '%s'", modulePathFromOutput(o.cfg.OutputDir))
		o.log("realize:       2. Duplicate type declarations — two tasks defined conflicting interfaces")
		o.log("realize:       3. Function signature mismatch — caller ignores an error return value")
		o.log("realize:       Run 'go build ./...' inside the backend directory to reproduce the errors.")
	}

	o.log("realize: complete — output written to %s", o.cfg.OutputDir)
	return nil
}

// runWave executes all tasks in a wave concurrently, bounded by cfg.Parallelism.
// Tasks that are already recorded as completed in st are skipped.
func (o *Orchestrator) runWave(
	ctx context.Context,
	taskIDs []string,
	d *dag.DAG,
	providers manifest.ProviderAssignments,
	reg *skills.FileRegistry,
	defaultProvider manifest.ProviderAssignment,
	verifiers *verify.Registry,
	writer *output.Writer,
	st *state.Store,
	mem *memory.SharedMemory,
	resolvedGoVersion string,
	resolvedGoModules map[string]deps.ModuleInfo,
) error {
	sem := make(chan struct{}, o.cfg.Parallelism)
	g, gctx := errgroup.WithContext(ctx)

	for _, id := range taskIDs {
		id := id // capture for goroutine

		if st.IsCompleted(id) {
			o.log("[%s] skipping (already completed)", id)
			continue
		}

		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			task := d.Tasks[id]
			techs := technologiesFor(task)
			skillDocs := reg.LookupAll(task.Kind, techs)

			o.log("[%s] starting: %s", task.ID, task.Label)

			// Dependency resolution tasks run a package manager directly — no LLM.
			// All other tasks resolve a provider (manifest override or default Claude)
			// and apply abstract model tiering for provider-agnostic tier escalation.
			var a agent.Agent
			initialTier := tierForKind(task.Kind)
			pa := defaultProvider

			if task.Kind == dag.TaskKindDependencyResolution {
				var svcTmpDir string
				if slug, ok := serviceSlug(task.ID); ok {
					svcTmpDir = filepath.Join(writer.BaseDir(), ".tmp", "svc."+slug)
				}
				a = deps.New(svcTmpDir, o.cfg.Verbose)
			} else {
				// Use per-section manifest override when configured; otherwise default.
				if override, ok := providerFor(task.ID, providers); ok && override.Credential != "" {
					pa = override
				}
				a = buildAgentForTier(pa, initialTier, defaultMaxTokens, o.cfg.Verbose)
			}

			runner := &TaskRunner{
				task:               task,
				agent:              a,
				verifier:           verifiers.ForTask(task),
				writer:             writer,
				state:              st,
				memory:             mem,
				skillDocs:          skillDocs,
				maxRetries:         o.cfg.MaxRetries,
				verbose:            o.cfg.Verbose,
				logFn:              o.cfg.LogFunc,
				providerAssignment: pa,
				initialTier:        initialTier,
				depsContext:        buildDepsContext(gctx, task, resolvedGoVersion, resolvedGoModules),
			}
			return runner.Run(gctx)
		})
	}

	return g.Wait()
}

// printPlan prints the task DAG in dry-run mode without invoking any agents.
func (o *Orchestrator) printPlan(d *dag.DAG, providers manifest.ProviderAssignments) error {
	defaultPA := o.resolveDefaultProvider()
	fmt.Printf("Execution plan (%d tasks, %d waves):\n\n", len(d.Tasks), len(d.Levels()))
	for i, wave := range d.Levels() {
		fmt.Printf("Wave %d:\n", i)
		for _, id := range wave {
			task := d.Tasks[id]
			model := describeProvider(id, providers, task.Kind, defaultPA)
			fmt.Printf("  [%s] %s  →  %s\n", task.Kind, task.Label, model)
			if len(task.Dependencies) > 0 {
				fmt.Printf("    deps: %v\n", task.Dependencies)
			}
		}
	}
	return nil
}

// modulePathFromOutput attempts to read the Go module path from the first go.mod
// found under outputDir, for use in diagnostic messages.
func modulePathFromOutput(outputDir string) string {
	var result string
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || result != "" {
			return nil
		}
		if info.IsDir() {
			if name := info.Name(); name == ".tmp" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() != "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				result = strings.TrimSpace(strings.TrimPrefix(line, "module "))
				return filepath.SkipAll
			}
		}
		return nil
	})
	if result == "" {
		return "<unknown>"
	}
	return result
}

// loadManifest reads and parses a manifest.json file.
func loadManifest(path string) (*manifest.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}

// buildDepsContext computes the dependency & API reference context for a task's
// system prompt. Returns "" for tasks with no relevant context (data tasks, etc.).
// For infra and frontend tasks, it resolves live package versions from npm and the
// Go module proxy (falling back to the static entries in WellKnownNpmPackages /
// WellKnownGoDevTools when the registries are unreachable).
// resolvedGoVersion is the minimum Go runtime version resolved once at startup via
// deps.ResolveGoVersion; it is injected into service task prompts so the plan LLM
// generates a go.mod whose `go X.Y` directive matches the infra Dockerfile base image.
// resolvedGoModules is the live-fetched module version map from deps.ResolveGoModuleVersions;
// it replaces hardcoded fallback versions for all Go framework and library dependencies.
func buildDepsContext(ctx context.Context, task *dag.Task, resolvedGoVersion string, resolvedGoModules map[string]deps.ModuleInfo) string {
	switch task.Kind {
	case dag.TaskKindInfraDocker, dag.TaskKindInfraCI, dag.TaskKindInfraTerraform:
		hasGoServices := len(task.Payload.AllServices) > 0
		hasFrontend := task.Payload.Frontend != nil
		return deps.InfraPromptContext(ctx, hasGoServices, hasFrontend)

	case dag.TaskKindFrontend:
		return deps.InfraPromptContext(ctx, false, true)

	default:
		// Backend service tasks: inject Go module versions + library API docs.
		if task.Payload.Service == nil {
			return ""
		}
		var technologies []string
		technologies = append(technologies, task.Payload.Service.Language)
		for _, db := range task.Payload.Databases {
			technologies = append(technologies, string(db.Type))
		}
		if task.Payload.Auth != nil {
			technologies = append(technologies, task.Payload.Auth.Strategy)
		}
		return deps.PromptContext(task.Payload.Service.Framework, technologies, resolvedGoVersion, resolvedGoModules)
	}
}

// technologiesFor returns all technology strings relevant to a task for skill lookup.
func technologiesFor(task *dag.Task) []string {
	techs := make([]string, 0, 8)

	// Service layer tasks and legacy service tasks: use Service or AllServices.
	if task.Payload.Service != nil {
		techs = append(techs, task.Payload.Service.Language, task.Payload.Service.Framework)
	} else if len(task.Payload.AllServices) > 0 {
		for _, svc := range task.Payload.AllServices {
			techs = append(techs, svc.Language, svc.Framework)
		}
	}

	// Databases.
	for _, db := range task.Payload.Databases {
		techs = append(techs, string(db.Type))
	}

	// Messaging broker.
	if task.Payload.Messaging != nil {
		techs = append(techs, task.Payload.Messaging.BrokerTech)
	}

	// Frontend framework.
	if task.Payload.Frontend != nil {
		techs = append(techs, task.Payload.Frontend.Tech.Framework)
		techs = append(techs, task.Payload.Frontend.Tech.Styling)
	}

	// Infrastructure.
	if task.Payload.Infra != nil {
		techs = append(techs, task.Payload.Infra.CICD.Platform)
		techs = append(techs, task.Payload.Infra.CICD.IaCTool)
	}

	// Job queue technologies so job queue skills get injected.
	for _, jq := range task.Payload.JobQueues {
		if jq.Technology != "" {
			techs = append(techs, jq.Technology)
		}
	}

	// Caching technologies.
	for _, c := range task.Payload.Cachings {
		if c.CacheDB != "" {
			techs = append(techs, c.CacheDB)
		}
	}

	// File storage technologies.
	for _, fs := range task.Payload.FileStorages {
		if fs.Technology != "" {
			techs = append(techs, fs.Technology)
		}
	}

	// Auth strategy.
	if task.Payload.Auth != nil && task.Payload.Auth.Strategy != "" {
		techs = append(techs, task.Payload.Auth.Strategy)
	}

	// Cross-cutting: testing frameworks and docs formats.
	if task.Payload.CrossCut != nil {
		t := task.Payload.CrossCut.Testing
		for _, fw := range []string{t.Unit, t.Integration, t.E2E, t.API, t.Load, t.Contract, t.FrontendTesting} {
			if fw != "" && fw != "none" {
				techs = append(techs, fw)
			}
		}
		if task.Payload.CrossCut.Docs.Changelog != "" {
			techs = append(techs, task.Payload.CrossCut.Docs.Changelog)
		}
	}

	// Frontend: realtime strategy, bundle optimization, error boundary.
	if task.Payload.Frontend != nil {
		ft := task.Payload.Frontend.Tech
		for _, v := range []string{ft.RealtimeStrategy, ft.BundleOptimization, ft.ErrorBoundary} {
			if v != "" && v != "none" {
				techs = append(techs, v)
			}
		}
	}

	return techs
}
