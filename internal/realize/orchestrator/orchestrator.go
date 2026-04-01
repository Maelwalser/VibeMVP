package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/vibe-mvp/internal/manifest"
	"github.com/vibe-mvp/internal/realize/agent"
	"github.com/vibe-mvp/internal/realize/dag"
	"github.com/vibe-mvp/internal/realize/output"
	"github.com/vibe-mvp/internal/realize/skills"
	"github.com/vibe-mvp/internal/realize/verify"
)

const (
	defaultModel     = "claude-opus-4-6"
	defaultMaxTokens = int64(16000)
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
	AuthToken    string // Claude Pro/Max session token; empty = use ANTHROPIC_API_KEY
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
		return o.printPlan(d)
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

	// Set up agent and verifier registry.
	a := agent.NewClaudeAgent(defaultModel, defaultMaxTokens, o.cfg.Verbose, o.cfg.AuthToken)
	verifiers := verify.NewRegistry()

	fmt.Fprintf(os.Stderr, "realize: starting %d tasks across %d wave(s)\n",
		len(d.Tasks), len(d.Levels()))

	// Execute waves in order; tasks within each wave run in parallel.
	for waveIdx, wave := range d.Levels() {
		fmt.Fprintf(os.Stderr, "realize: wave %d (%d tasks): %v\n", waveIdx, len(wave), wave)

		if err := o.runWave(ctx, wave, d, reg, a, verifiers, writer); err != nil {
			return fmt.Errorf("wave %d: %w", waveIdx, err)
		}
	}

	fmt.Fprintf(os.Stderr, "realize: complete — output written to %s\n", o.cfg.OutputDir)
	return nil
}

// runWave executes all tasks in a wave concurrently, bounded by cfg.Parallelism.
func (o *Orchestrator) runWave(
	ctx context.Context,
	taskIDs []string,
	d *dag.DAG,
	reg *skills.FileRegistry,
	a agent.Agent,
	verifiers *verify.Registry,
	writer *output.Writer,
) error {
	sem := make(chan struct{}, o.cfg.Parallelism)
	g, gctx := errgroup.WithContext(ctx)

	for _, id := range taskIDs {
		id := id // capture for goroutine
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			task := d.Tasks[id]
			techs := technologiesFor(task)
			skillDocs := reg.LookupAll(task.Kind, techs)

			fmt.Fprintf(os.Stderr, "[%s] starting: %s\n", task.ID, task.Label)

			runner := &TaskRunner{
				task:       task,
				agent:      a,
				verifier:   verifiers.ForTask(task),
				writer:     writer,
				skillDocs:  skillDocs,
				maxRetries: o.cfg.MaxRetries,
				verbose:    o.cfg.Verbose,
			}
			return runner.Run(gctx)
		})
	}

	return g.Wait()
}

// printPlan prints the task DAG in dry-run mode without invoking any agents.
func (o *Orchestrator) printPlan(d *dag.DAG) error {
	fmt.Printf("Execution plan (%d tasks, %d waves):\n\n", len(d.Tasks), len(d.Levels()))
	for i, wave := range d.Levels() {
		fmt.Printf("Wave %d:\n", i)
		for _, id := range wave {
			task := d.Tasks[id]
			fmt.Printf("  [%s] %s\n", task.Kind, task.Label)
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
