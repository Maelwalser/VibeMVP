package dag

import (
	"fmt"
	"strings"

	"github.com/vibe-mvp/internal/manifest"
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

func serviceIDs(m *manifest.Manifest) []string {
	ids := make([]string, 0, len(m.Backend.Services))
	for _, svc := range m.Backend.Services {
		ids = append(ids, serviceTaskID(svc.Name))
	}
	return ids
}

func serviceTaskID(name string) string {
	return "service." + strings.ToLower(strings.ReplaceAll(name, " ", "-"))
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
		b.addMonolithTask(m, d, dataDeps)
	default:
		// microservices, event-driven, hybrid: one task per service
		for i := range m.Backend.Services {
			b.addServiceTask(m, &m.Backend.Services[i], d, dataDeps)
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

func (b *Builder) addMonolithTask(m *manifest.Manifest, d *DAG, deps []string) {
	// For monolith/modular-monolith, represent as a single service task.
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
	add(d, &Task{
		ID:           "service.monolith",
		Kind:         TaskKindService,
		Label:        "Generate monolith service",
		Dependencies: deps,
		Payload: TaskPayload{
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			Service:     &svc,
			AllServices: m.Backend.Services,
			Domains:     m.Data.Domains,
			Databases:   m.Data.Databases,
			CommLinks:   m.Backend.CommLinks,
			Auth:        &m.Backend.Auth,
		},
	})
}

func (b *Builder) addServiceTask(m *manifest.Manifest, svc *manifest.ServiceDef, d *DAG, deps []string) {
	id := serviceTaskID(svc.Name)
	svcCopy := *svc
	add(d, &Task{
		ID:           id,
		Kind:         TaskKindService,
		Label:        fmt.Sprintf("Generate service: %s", svc.Name),
		Dependencies: deps,
		Payload: TaskPayload{
			ArchPattern: m.Backend.ArchPattern,
			EnvConfig:   m.Backend.Env,
			Service:     &svcCopy,
			AllServices: m.Backend.Services,
			Domains:     m.Data.Domains,
			Databases:   m.Data.Databases,
			CommLinks:   commLinksFor(svc.Name, m.Backend.CommLinks),
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
			},
		})
	}
}

// ── Wave 5: cross-cutting ─────────────────────────────────────────────────────

func (b *Builder) addCrossCutTasks(m *manifest.Manifest, d *DAG) {
	// Depends on everything that was emitted.
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
