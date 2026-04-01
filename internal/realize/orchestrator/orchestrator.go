package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/vibe-mvp/internal/manifest"
	"github.com/vibe-mvp/internal/realize/agent"
	"github.com/vibe-mvp/internal/realize/dag"
	"github.com/vibe-mvp/internal/realize/memory"
	"github.com/vibe-mvp/internal/realize/output"
	"github.com/vibe-mvp/internal/realize/skills"
	"github.com/vibe-mvp/internal/realize/state"
	"github.com/vibe-mvp/internal/realize/verify"
)

const (
	defaultModel     = "claude-opus-4-6"
	defaultMaxTokens = int64(32000)
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

	// Print plan in dry-run mode.
	if o.cfg.DryRun {
		return o.printPlan(d, m.Providers)
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

	// Build a default agent; per-section agents are resolved below.
	defaultAgent := agent.NewClaudeAgent(defaultModel, defaultMaxTokens, o.cfg.Verbose)

	// Log configured per-section model assignments.
	for sectionID, pa := range m.Providers {
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

		if err := o.runWave(ctx, wave, d, m.Providers, reg, defaultAgent, verifiers, writer, st, mem); err != nil {
			return fmt.Errorf("wave %d: %w", waveIdx, err)
		}
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
	defaultAgent agent.Agent,
	verifiers *verify.Registry,
	writer *output.Writer,
	st *state.Store,
	mem *memory.SharedMemory,
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

			// Resolve per-section agent if a provider assignment exists.
			a := resolveAgent(task.ID, providers, defaultAgent, o.cfg.Verbose)

			runner := &TaskRunner{
				task:       task,
				agent:      a,
				verifier:   verifiers.ForTask(task),
				writer:     writer,
				state:      st,
				memory:     mem,
				skillDocs:  skillDocs,
				maxRetries: o.cfg.MaxRetries,
				verbose:    o.cfg.Verbose,
				logFn:      o.cfg.LogFunc,
			}
			return runner.Run(gctx)
		})
	}

	return g.Wait()
}

// resolveAgent returns a task-specific agent if the manifest has a provider
// assignment for the task's section, otherwise returns the default agent.
func resolveAgent(taskID string, providers manifest.ProviderAssignments, def agent.Agent, verbose bool) agent.Agent {
	pa, ok := providerFor(taskID, providers)
	if !ok || pa.Credential == "" {
		return def
	}
	switch pa.Provider {
	case "Claude":
		model := claudeModelID(pa.Model, pa.Version)
		return agent.NewClaudeAgentWithKey(model, defaultMaxTokens, verbose, pa.Credential)
	case "ChatGPT":
		model := openaiModelID(pa.Model, pa.Version)
		return agent.NewOpenAIAgent("https://api.openai.com", pa.Credential, model, defaultMaxTokens, verbose)
	case "Gemini":
		model := geminiModelID(pa.Model, pa.Version)
		return agent.NewGeminiAgent(pa.Credential, model, defaultMaxTokens, verbose)
	case "Mistral":
		model := mistralModelID(pa.Model, pa.Version)
		return agent.NewOpenAIAgent("https://api.mistral.ai", pa.Credential, model, defaultMaxTokens, verbose)
	case "Llama":
		model := llamaModelID(pa.Model, pa.Version)
		return agent.NewOpenAIAgent("https://api.groq.com/openai", pa.Credential, model, defaultMaxTokens, verbose)
	default:
		return def
	}
}

// providerFor returns the ProviderAssignment for the section that owns taskID.
// Task IDs follow "<section>.<name>" or just "<section>".
func providerFor(taskID string, providers manifest.ProviderAssignments) (manifest.ProviderAssignment, bool) {
	if providers == nil {
		return manifest.ProviderAssignment{}, false
	}
	sectionID := taskID
	if dot := strings.Index(taskID, "."); dot >= 0 {
		sectionID = taskID[:dot]
	}
	pa, ok := providers[sectionID]
	return pa, ok
}

// describeProvider returns a human-readable model label for dry-run output,
// e.g. "Claude Opus 4.6" or "Gemini Flash 2.0". Falls back to "default" when
// the section has no configured provider.
func describeProvider(taskID string, providers manifest.ProviderAssignments) string {
	pa, ok := providerFor(taskID, providers)
	if !ok || pa.Credential == "" {
		return "default (" + defaultModel + ")"
	}
	s := pa.Provider
	if pa.Model != "" {
		s += " " + pa.Model
	}
	if pa.Version != "" {
		s += " " + pa.Version
	}
	return s
}

// claudeModelID maps a tier name + version to the Anthropic model string.
func claudeModelID(tier, version string) string {
	switch tier {
	case "Haiku":
		return "claude-haiku-4-5-20251001"
	case "Sonnet":
		return "claude-sonnet-4-6"
	case "Opus":
		return "claude-opus-4-6"
	default:
		return defaultModel
	}
}

// openaiModelID maps ChatGPT tier + version to the OpenAI model string.
func openaiModelID(tier, version string) string {
	switch tier {
	case "Mini":
		if version == "o3-mini" {
			return "o3-mini"
		}
		return "gpt-4o-mini"
	case "4o":
		if version == "4o-2024" {
			return "gpt-4o-2024-11-20"
		}
		return "gpt-4o"
	case "o1":
		if version == "o1-preview" {
			return "o1-preview"
		}
		return "o1"
	default:
		return "gpt-4o"
	}
}

// geminiModelID maps a Gemini tier + version to the Google model string.
func geminiModelID(tier, version string) string {
	switch tier {
	case "Flash":
		if version == "1.5" {
			return "gemini-1.5-flash"
		}
		return "gemini-2.0-flash"
	case "Pro":
		if version == "1.5" {
			return "gemini-1.5-pro"
		}
		return "gemini-2.0-pro-exp"
	case "Ultra":
		return "gemini-ultra"
	default:
		return "gemini-2.0-flash"
	}
}

// mistralModelID maps a Mistral tier + version to the Mistral API model string.
func mistralModelID(tier, version string) string {
	switch tier {
	case "Nemo":
		return "open-mistral-nemo"
	case "Small":
		if version == "3.0" {
			return "mistral-small-2402"
		}
		return "mistral-small-2409"
	case "Large":
		if version == "2.0" {
			return "mistral-large-2407"
		}
		return "mistral-large-2411"
	default:
		return "mistral-large-2411"
	}
}

// llamaModelID maps a Llama tier + version to the Groq model string.
func llamaModelID(tier, version string) string {
	switch tier {
	case "8B":
		if version == "3.1" {
			return "llama-3.1-8b-instant"
		}
		return "llama-3.2-8b-preview"
	case "70B":
		if version == "3.1" {
			return "llama-3.1-70b-versatile"
		}
		return "llama-3.3-70b-versatile"
	case "405B":
		return "llama-3.1-405b-reasoning"
	default:
		return "llama-3.3-70b-versatile"
	}
}

// printPlan prints the task DAG in dry-run mode without invoking any agents.
// Only tasks whose section has a configured provider show the model label;
// unconfigured tasks show the default model.
func (o *Orchestrator) printPlan(d *dag.DAG, providers manifest.ProviderAssignments) error {
	fmt.Printf("Execution plan (%d tasks, %d waves):\n\n", len(d.Tasks), len(d.Levels()))
	for i, wave := range d.Levels() {
		fmt.Printf("Wave %d:\n", i)
		for _, id := range wave {
			task := d.Tasks[id]
			model := describeProvider(id, providers)
			fmt.Printf("  [%s] %s  →  %s\n", task.Kind, task.Label, model)
			if len(task.Dependencies) > 0 {
				fmt.Printf("    deps: %v\n", task.Dependencies)
			}
		}
	}
	return nil
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

// technologiesFor returns all technology strings relevant to a task for skill lookup.
func technologiesFor(task *dag.Task) []string {
	techs := make([]string, 0, 8)

	// Service language + framework.
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

	return techs
}
