package dag

import (
	"fmt"
	"strings"

	"github.com/vibe-menu/internal/manifest"
)

// Builder converts a manifest.Manifest into an execution DAG.
type Builder struct{}

// Build reads m and emits the complete task DAG with all dependencies wired.
func (b *Builder) Build(m *manifest.Manifest) (*DAG, error) {
	d := &DAG{Tasks: make(map[string]*Task)}

	b.addDataTasks(m, d)
	b.addBackendTasks(m, d)
	b.addContractsTask(m, d)
	b.addFrontendTask(m, d)
	b.addInfraTasks(m, d)
	b.addCrossCutTasks(m, d)

	if err := d.build(); err != nil {
		return nil, fmt.Errorf("build dag: %w", err)
	}
	return d, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func add(d *DAG, t *Task) {
	d.Tasks[t.ID] = t
}

// serviceIDs returns the final task ID in each service's generation chain
// (the bootstrap task). Used by downstream tasks (contracts, infra, crosscut)
// to declare that all service layers must be complete before they run.
func serviceIDs(m *manifest.Manifest) []string {
	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		return []string{svcBootstrapID("monolith")}
	default:
		ids := make([]string, 0, len(m.Backend.Services))
		for _, svc := range m.Backend.Services {
			ids = append(ids, svcBootstrapID(svc.Name))
		}
		return ids
	}
}

// serviceAllLayerIDs returns all task IDs for a service (plan + deps + four layers).
// Used by crosscut tasks that need to depend on every layer.
func serviceAllLayerIDs(name string) []string {
	slug := svcSlug(name)
	return []string{
		"svc." + slug + ".plan",
		"svc." + slug + ".deps",
		"svc." + slug + ".repository",
		"svc." + slug + ".service",
		"svc." + slug + ".handler",
		"svc." + slug + ".bootstrap",
	}
}

func svcBootstrapID(name string) string { return "svc." + svcSlug(name) + ".bootstrap" }
func svcPlanID(name string) string      { return "svc." + svcSlug(name) + ".plan" }
func svcDepsID(name string) string      { return "svc." + svcSlug(name) + ".deps" }
func svcSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// deriveModulePath returns a deterministic Go module path for a service.
// Uses the service name directly (e.g. "core-api") so all agents in the
// chain produce consistent import paths without guessing an org name.
func deriveModulePath(svcName string) string {
	return svcSlug(svcName)
}

// ── Wave 0: data tasks ────────────────────────────────────────────────────────

func (b *Builder) addDataTasks(m *manifest.Manifest, d *DAG) {
	svcDirs := serviceOutputDirs(m)
	basePayload := TaskPayload{
		ArchPattern:  m.Backend.ArchPattern,
		EnvConfig:    m.Backend.Env,
		Domains:      m.Data.Domains,
		Databases:    m.Data.Databases,
		Cachings:     m.Data.Cachings,
		FileStorages: m.Data.FileStorages,
		AllServices:  m.Backend.Services,
		OutputDir:    backendBaseDir(svcDirs),
	}

	add(d, &Task{
		ID:           "data.schemas",
		Kind:         TaskKindDataSchemas,
		Label:        "Generate domain schemas / ORM models",
		Dependencies: nil,
		Payload:      basePayload,
	})

	add(d, &Task{
		ID:           "data.migrations",
		Kind:         TaskKindDataMigrations,
		Label:        "Generate database migration files",
		Dependencies: nil,
		Payload:      basePayload,
	})
}

// ── Wave 1: backend tasks ─────────────────────────────────────────────────────

func (b *Builder) addBackendTasks(m *manifest.Manifest, d *DAG) {
	dataDeps := []string{"data.schemas", "data.migrations"}
	svcDirs := serviceOutputDirs(m)

	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		svc := manifest.ServiceDef{
			Name:           "monolith",
			Responsibility: "All application logic",
			Language:       m.Backend.Language,
			Framework:      m.Backend.Framework,
		}
		if len(m.Backend.Services) > 0 {
			svc = m.Backend.Services[0]
			svc.Name = "monolith"
		}
		b.addServiceTaskChain(m, &svc, d, dataDeps, svcDirs)
	default:
		// microservices, event-driven, hybrid: one chain per service
		for i := range m.Backend.Services {
			b.addServiceTaskChain(m, &m.Backend.Services[i], d, dataDeps, svcDirs)
		}
	}

	// Auth middleware generates Go code — it belongs alongside the backend module.
	if m.Backend.Auth.Strategy != "" {
		add(d, &Task{
			ID:    "backend.auth",
			Kind:  TaskKindAuth,
			Label: "Generate authentication middleware",
			Dependencies: dataDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				EnvConfig:   m.Backend.Env,
				Domains:     m.Data.Domains,
				Databases:   m.Data.Databases,
				AllServices: m.Backend.Services,
				Auth:        &m.Backend.Auth,
				OutputDir:   backendBaseDir(svcDirs),
			},
		})
	}

	// Messaging broker + event stubs (config-level, stays at root)
	if m.Backend.Messaging != nil {
		add(d, &Task{
			ID:           "backend.messaging",
			Kind:         TaskKindMessaging,
			Label:        "Generate message broker configuration and event stubs",
			Dependencies: dataDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				EnvConfig:   m.Backend.Env,
				Domains:     m.Data.Domains,
				AllServices: m.Backend.Services,
				Messaging:   m.Backend.Messaging,
				CommLinks:   m.Backend.CommLinks,
			},
		})
	}

	// API gateway configuration (config-level, stays at root)
	if m.Backend.APIGateway != nil {
		add(d, &Task{
			ID:           "backend.gateway",
			Kind:         TaskKindGateway,
			Label:        "Generate API gateway configuration",
			Dependencies: dataDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				EnvConfig:   m.Backend.Env,
				AllServices: m.Backend.Services,
				APIGateway:  m.Backend.APIGateway,
			},
		})
	}
}

// addServiceTaskChain emits four focused tasks for one service:
//
//	repository → service → handler → bootstrap
//
// Each task is small (~2–6K output tokens), independently verifiable with
// go build, and receives only the context it actually needs. Module path is
// derived from the service name and injected into every task so all layers
// share identical import paths with no guessing.
//
// svcDirs is the full service-slug → output-dir map used to set OutputDir and
// ServiceDirs on every task in the chain so the runner and infra agents know
// where to place and find the generated files.
func (b *Builder) addServiceTaskChain(m *manifest.Manifest, svc *manifest.ServiceDef, d *DAG, dataDeps []string, svcDirs map[string]string) {
	slug := svcSlug(svc.Name)
	modPath := deriveModulePath(svc.Name)
	svcCopy := *svc
	links := commLinksFor(svc.Name, m.Backend.CommLinks)
	jobQueues := jobQueuesForService(svc.Name, m.Backend.JobQueues)
	cronJobs := cronJobsForService(svc.Name, m.Backend.JobQueues)
	outputDir := svcDirs[slug]

	planID := svcPlanID(slug)
	depsID := svcDepsID(slug)
	repoID := "svc." + slug + ".repository"
	svcID := "svc." + slug + ".service"
	handlerID := "svc." + slug + ".handler"
	bootID := "svc." + slug + ".bootstrap"

	// Layer 0a — plan: project skeleton (go.mod with direct deps + repository interfaces).
	// The LLM lists only direct dependencies; it does NOT pin transitive packages.
	// Runs before all implementation layers so every downstream agent
	// has a stable contract to implement against.
	add(d, &Task{
		ID:           planID,
		Kind:         TaskKindServicePlan,
		Label:        fmt.Sprintf("%s — project skeleton (interfaces + go.mod)", svc.Name),
		Dependencies: dataDeps,
		Payload: TaskPayload{
			ModulePath:   modPath,
			ArchPattern:  m.Backend.ArchPattern,
			EnvConfig:    m.Backend.Env,
			Service:      &svcCopy,
			Domains:      m.Data.Domains,
			Databases:    m.Data.Databases,
			Cachings:     m.Data.Cachings,
			FileStorages: m.Data.FileStorages,
			Auth:         &m.Backend.Auth,
			ServiceDirs:  svcDirs,
			OutputDir:    outputDir,
		},
	})

	// Layer 0b — deps: dependency resolution (no LLM).
	// Runs go mod tidy / npm install / pip-compile on the plan task's
	// dependency manifest to lock all transitive versions using the live
	// package registry. Committed go.mod+go.sum become the canonical module
	// graph for every subsequent layer.
	add(d, &Task{
		ID:           depsID,
		Kind:         TaskKindDependencyResolution,
		Label:        fmt.Sprintf("%s — resolve & lock dependencies", svc.Name),
		Dependencies: []string{planID},
		Payload: TaskPayload{
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			Service:     &svcCopy,
			ServiceDirs: svcDirs,
			OutputDir:   outputDir,
		},
	})

	// Layer 1 — repository: data-access interfaces + DB implementations.
	// Small: ~200–400 lines of Go. Depends on resolved module graph from deps.
	add(d, &Task{
		ID:           repoID,
		Kind:         TaskKindServiceRepository,
		Label:        fmt.Sprintf("%s — repository layer", svc.Name),
		Dependencies: []string{depsID},
		Payload: TaskPayload{
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			Service:     &svcCopy,
			Domains:     m.Data.Domains,
			Databases:   m.Data.Databases,
			Cachings:    m.Data.Cachings,
			ServiceDirs: svcDirs,
			OutputDir:   outputDir,
		},
	})

	// Layer 2 — service/business logic: orchestrates repositories.
	// Small: ~150–300 lines of Go per domain entity.
	add(d, &Task{
		ID:           svcID,
		Kind:         TaskKindServiceLogic,
		Label:        fmt.Sprintf("%s — service layer", svc.Name),
		Dependencies: []string{repoID},
		Payload: TaskPayload{
			ModulePath:   modPath,
			ArchPattern:  m.Backend.ArchPattern,
			Service:      &svcCopy,
			Domains:      m.Data.Domains,
			Cachings:     m.Data.Cachings,
			FileStorages: m.Data.FileStorages,
			JobQueues:    jobQueues,
			CronJobs:     cronJobs,
			ServiceDirs:  svcDirs,
			OutputDir:    outputDir,
		},
	})

	// Layer 3 — handlers: HTTP routes and request/response mapping.
	// Small: ~200–400 lines of Go. Auth config injected so handlers can
	// apply the correct middleware.
	add(d, &Task{
		ID:           handlerID,
		Kind:         TaskKindServiceHandler,
		Label:        fmt.Sprintf("%s — handler layer", svc.Name),
		Dependencies: []string{svcID},
		Payload: TaskPayload{
			ModulePath:   modPath,
			ArchPattern:  m.Backend.ArchPattern,
			Service:      &svcCopy,
			Domains:      m.Data.Domains,
			Endpoints:    m.Contracts.Endpoints,
			CommLinks:    links,
			Auth:         &m.Backend.Auth,
			FileStorages: m.Data.FileStorages,
			JobQueues:    jobQueues,
			CronJobs:     cronJobs,
			ServiceDirs:  svcDirs,
			OutputDir:    outputDir,
		},
	})

	// Layer 4 — bootstrap: main.go, go.mod, config, middleware wiring.
	// Small: ~100–200 lines. Wires all layers together.
	add(d, &Task{
		ID:           bootID,
		Kind:         TaskKindServiceBootstrap,
		Label:        fmt.Sprintf("%s — bootstrap", svc.Name),
		Dependencies: []string{handlerID},
		Payload: TaskPayload{
			ModulePath:   modPath,
			ArchPattern:  m.Backend.ArchPattern,
			EnvConfig:    m.Backend.Env,
			Service:      &svcCopy,
			AllServices:  m.Backend.Services,
			Databases:    m.Data.Databases,
			Cachings:     m.Data.Cachings,
			FileStorages: m.Data.FileStorages,
			Auth:         &m.Backend.Auth,
			JobQueues:    jobQueues,
			CronJobs:     cronJobs,
			ServiceDirs:  svcDirs,
			OutputDir:    outputDir,
		},
	})
}

// commLinksFor returns the comm links involving the given service.
func commLinksFor(name string, links []manifest.CommLink) []manifest.CommLink {
	out := make([]manifest.CommLink, 0)
	for _, l := range links {
		if l.From == name || l.To == name {
			out = append(out, l)
		}
	}
	return out
}

// jobQueuesForService returns job queues belonging to the given service.
// If a queue has no WorkerService set, it is included for all services.
func jobQueuesForService(name string, queues []manifest.JobQueueDef) []manifest.JobQueueDef {
	out := make([]manifest.JobQueueDef, 0)
	for _, q := range queues {
		if q.WorkerService == "" || q.WorkerService == name {
			out = append(out, q)
		}
	}
	return out
}

// cronJobsForService collects all cron jobs nested within the job queues that
// belong to the given service. CronJobDef has no direct service linkage field —
// it is always a child of a JobQueueDef, so we filter by the queue's WorkerService.
func cronJobsForService(name string, queues []manifest.JobQueueDef) []manifest.CronJobDef {
	out := make([]manifest.CronJobDef, 0)
	for _, q := range queues {
		if q.WorkerService == "" || q.WorkerService == name {
			out = append(out, q.CronJobs...)
		}
	}
	return out
}

// ── Wave 2: contracts ─────────────────────────────────────────────────────────

func (b *Builder) addContractsTask(m *manifest.Manifest, d *DAG) {
	if len(m.Contracts.DTOs) == 0 && len(m.Contracts.Endpoints) == 0 {
		return
	}

	// Depends on all service tasks.
	deps := append(serviceIDs(m), "data.schemas")
	if m.Backend.Auth.Strategy != "" {
		deps = append(deps, "backend.auth")
	}

	svcDirs := serviceOutputDirs(m)
	add(d, &Task{
		ID:           "contracts",
		Kind:         TaskKindContracts,
		Label:        "Generate DTOs, API types, and OpenAPI spec",
		Dependencies: deps,
		Payload: TaskPayload{
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			Domains:     m.Data.Domains,
			AllServices: m.Backend.Services,
			DTOs:        m.Contracts.DTOs,
			Endpoints:   m.Contracts.Endpoints,
			Versioning:  m.Contracts.Versioning,
			Auth:        &m.Backend.Auth,
			OutputDir:   contractsOutputDir(m, svcDirs),
			ServiceDirs: svcDirs,
		},
	})
}

// ── Wave 3: frontend ──────────────────────────────────────────────────────────

func (b *Builder) addFrontendTask(m *manifest.Manifest, d *DAG) {
	if m.Frontend.Tech.Framework == "" {
		return
	}

	deps := []string{"contracts"}
	// contracts may not exist if no DTOs/endpoints are defined.
	if _, ok := d.Tasks["contracts"]; !ok {
		deps = append(serviceIDs(m), "data.schemas")
	}

	svcDirs := serviceOutputDirs(m)
	fp := m.Frontend
	add(d, &Task{
		ID:           "frontend",
		Kind:         TaskKindFrontend,
		Label:        fmt.Sprintf("Generate frontend (%s)", m.Frontend.Tech.Framework),
		Dependencies: deps,
		Payload: TaskPayload{
			ArchPattern: m.Backend.ArchPattern,
			DTOs:        m.Contracts.DTOs,
			Endpoints:   m.Contracts.Endpoints,
			Versioning:  m.Contracts.Versioning,
			AllServices: m.Backend.Services,
			Auth:        &m.Backend.Auth,
			Frontend:    &fp,
			OutputDir:   svcDirs["frontend"],
		},
	})
}

// ── Wave 4: infrastructure ────────────────────────────────────────────────────

func (b *Builder) addInfraTasks(m *manifest.Manifest, d *DAG) {
	// Depends on all service tasks + optional frontend.
	baseDeps := append(serviceIDs(m), "data.schemas")
	if _, ok := d.Tasks["frontend"]; ok {
		baseDeps = append(baseDeps, "frontend")
	}
	if _, ok := d.Tasks["contracts"]; ok {
		baseDeps = append(baseDeps, "contracts")
	}

	infra := m.Infra
	svcDirs := serviceOutputDirs(m)

	add(d, &Task{
		ID:           "infra.docker",
		Kind:         TaskKindInfraDocker,
		Label:        "Generate Dockerfiles and docker-compose",
		Dependencies: baseDeps,
		Payload: TaskPayload{
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			AllServices: m.Backend.Services,
			Domains:     m.Data.Domains,
			Databases:   m.Data.Databases,
			Infra:       &infra,
			Frontend:    frontendOrNil(m),
			ServiceDirs: svcDirs,
		},
	})

	if m.Infra.CICD.IaCTool != "" && m.Infra.CICD.IaCTool != "None" {
		add(d, &Task{
			ID:           "infra.terraform",
			Kind:         TaskKindInfraTerraform,
			Label:        fmt.Sprintf("Generate IaC files (%s)", m.Infra.CICD.IaCTool),
			Dependencies: baseDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				EnvConfig:   m.Backend.Env,
				AllServices: m.Backend.Services,
				Databases:   m.Data.Databases,
				Infra:       &infra,
				ServiceDirs: svcDirs,
			},
		})
	}

	if m.Infra.CICD.Platform != "" && m.Infra.CICD.Platform != "none" {
		add(d, &Task{
			ID:           "infra.cicd",
			Kind:         TaskKindInfraCI,
			Label:        fmt.Sprintf("Generate CI/CD pipeline (%s)", m.Infra.CICD.Platform),
			Dependencies: baseDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				EnvConfig:   m.Backend.Env,
				AllServices: m.Backend.Services,
				Infra:       &infra,
				CrossCut:    crossCutOrNil(m),
				ServiceDirs: svcDirs,
			},
		})
	}
}

// ── Wave 5: cross-cutting ─────────────────────────────────────────────────────

func (b *Builder) addCrossCutTasks(m *manifest.Manifest, d *DAG) {
	// Depends on every task emitted so far — cross-cutting concerns test and
	// document the entire generated codebase.
	allDeps := make([]string, 0, len(d.Tasks))
	for id := range d.Tasks {
		allDeps = append(allDeps, id)
	}

	cc := m.CrossCut
	svcDirs := serviceOutputDirs(m)
	ccOutDir := crossCutOutputDir(m, svcDirs)

	if m.CrossCut.Testing.Unit != "" || m.CrossCut.Testing.E2E != "" {
		add(d, &Task{
			ID:           "crosscut.testing",
			Kind:         TaskKindCrossCutTesting,
			Label:        "Generate test scaffolding",
			Dependencies: allDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				AllServices: m.Backend.Services,
				Domains:     m.Data.Domains,
				DTOs:        m.Contracts.DTOs,
				Endpoints:   m.Contracts.Endpoints,
				Frontend:    frontendOrNil(m),
				Infra:       infraOrNil(m),
				CrossCut:    &cc,
				OutputDir:   ccOutDir,
				ServiceDirs: svcDirs,
			},
		})
	}

	if m.CrossCut.Docs.APIDocs != "" {
		add(d, &Task{
			ID:           "crosscut.docs",
			Kind:         TaskKindCrossCutDocs,
			Label:        "Generate API documentation",
			Dependencies: allDeps,
			Payload: TaskPayload{
				ArchPattern: m.Backend.ArchPattern,
				AllServices: m.Backend.Services,
				DTOs:        m.Contracts.DTOs,
				Endpoints:   m.Contracts.Endpoints,
				Versioning:  m.Contracts.Versioning,
				CrossCut:    &cc,
				OutputDir:   ".",
				ServiceDirs: svcDirs,
			},
		})
	}
}

// serviceOutputDirs returns a map from service slug → output directory (relative
// to the output root) where that service's generated source files reside.
//
// When a frontend exists alongside a monolith, backend files move to "backend/"
// so that go.mod and package.json are never in the same directory. Microservice
// projects place each service under "services/<slug>/". Frontend always goes to
// "frontend/". Backend-only monoliths stay at the root (".") unchanged.
//
// Infra tasks consume these values as Docker build contexts via the ServiceDirs
// payload field — agents must NOT invent subdirectories outside this map.
func serviceOutputDirs(m *manifest.Manifest) map[string]string {
	dirs := make(map[string]string)
	hasFrontend := m.Frontend.Tech.Framework != ""

	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		if hasFrontend {
			dirs["monolith"] = "backend"
		} else {
			dirs["monolith"] = "." // no separation needed without a frontend
		}
	default: // Microservices, Event-Driven, Hybrid
		for _, svc := range m.Backend.Services {
			dirs[svcSlug(svc.Name)] = "services/" + svcSlug(svc.Name)
		}
	}
	if hasFrontend {
		dirs["frontend"] = "frontend"
	}
	return dirs
}

// backendBaseDir returns the output directory for cross-service backend tasks
// (data schemas, migrations, auth middleware). For a monolith with a frontend
// these must live alongside the Go module in "backend/"; otherwise root ".".
func backendBaseDir(serviceDirs map[string]string) string {
	if dir, ok := serviceDirs["monolith"]; ok && dir != "." {
		return dir
	}
	return "."
}

// contractsOutputDir returns the output directory for the contracts task.
// For monolith/modular: co-located with the Go module ("backend" or ".").
// For distributed arches: "shared" — a top-level shared package consumed by all services.
func contractsOutputDir(m *manifest.Manifest, svcDirs map[string]string) string {
	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		return backendBaseDir(svcDirs)
	default: // Microservices, Event-Driven, Hybrid
		return "shared"
	}
}

// crossCutOutputDir returns the output directory for cross-cutting tasks.
// For monolith/modular: co-located with the Go module ("backend" or ".").
// For distributed arches: "." (root) — shared config and test utils live at the project root.
func crossCutOutputDir(m *manifest.Manifest, svcDirs map[string]string) string {
	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		return backendBaseDir(svcDirs)
	default:
		return "."
	}
}

// ── nil-safe helpers ──────────────────────────────────────────────────────────

func frontendOrNil(m *manifest.Manifest) *manifest.FrontendPillar {
	if m.Frontend.Tech.Framework == "" {
		return nil
	}
	fp := m.Frontend
	return &fp
}

func infraOrNil(m *manifest.Manifest) *manifest.InfraPillar {
	ip := m.Infra
	return &ip
}

func crossCutOrNil(m *manifest.Manifest) *manifest.CrossCutPillar {
	cc := m.CrossCut
	return &cc
}
