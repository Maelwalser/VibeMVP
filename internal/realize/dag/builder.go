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
	basePayload := TaskPayload{
		ArchPattern:  m.Backend.ArchPattern,
		EnvConfig:    m.Backend.Env,
		Domains:      m.Data.Domains,
		Databases:    m.Data.Databases,
		Cachings:     m.Data.Cachings,
		FileStorages: m.Data.FileStorages,
		AllServices:  m.Backend.Services,
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
		b.addServiceTaskChain(m, &svc, d, dataDeps)
	default:
		// microservices, event-driven, hybrid: one chain per service
		for i := range m.Backend.Services {
			b.addServiceTaskChain(m, &m.Backend.Services[i], d, dataDeps)
		}
	}

	// Auth middleware (always emitted if strategy is set)
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
			},
		})
	}

	// Messaging broker + event stubs
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

	// API gateway configuration
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
func (b *Builder) addServiceTaskChain(m *manifest.Manifest, svc *manifest.ServiceDef, d *DAG, dataDeps []string) {
	slug := svcSlug(svc.Name)
	modPath := deriveModulePath(svc.Name)
	svcCopy := *svc
	links := commLinksFor(svc.Name, m.Backend.CommLinks)

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
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			Service:     &svcCopy,
			Domains:     m.Data.Domains,
			Databases:   m.Data.Databases,
			Auth:        &m.Backend.Auth,
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
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			Service:     &svcCopy,
			Domains:     m.Data.Domains,
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
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			Service:     &svcCopy,
			Domains:     m.Data.Domains,
			Endpoints:   m.Contracts.Endpoints,
			CommLinks:   links,
			Auth:        &m.Backend.Auth,
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
			ModulePath:  modPath,
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			Service:     &svcCopy,
			AllServices: m.Backend.Services,
			Databases:   m.Data.Databases,
			Auth:        &m.Backend.Auth,
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
			},
		})
	}
}

// serviceOutputDirs returns a map from service slug → output directory (relative
// to the output root) where that service's generated source files reside.
//
// All service task chains write their files directly to the output root — the
// writer uses f.Path as-is, so go.mod lands at "go.mod", not "services/api/go.mod".
// Infra tasks must use these paths as Docker build contexts instead of inventing
// a multi-service subdirectory layout.
func serviceOutputDirs(m *manifest.Manifest) map[string]string {
	dirs := make(map[string]string)
	switch m.Backend.ArchPattern {
	case manifest.ArchMonolith, manifest.ArchModularMonolith:
		dirs["monolith"] = "."
	default:
		for _, svc := range m.Backend.Services {
			dirs[svcSlug(svc.Name)] = "."
		}
	}
	// Tell the infra.docker task where frontend source files live so it can
	// set the correct docker-compose build context instead of inventing ./frontend.
	if m.Frontend.Tech.Framework != "" {
		dirs["frontend"] = "."
	}
	return dirs
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
