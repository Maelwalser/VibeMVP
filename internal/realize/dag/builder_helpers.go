package dag

import "github.com/vibe-menu/internal/manifest"

// ── config ref resolution ────────────────────────────────────────────────────

// resolveConfigRefs populates empty Language/Framework fields on services that
// reference a StackConfig via ConfigRef. Mutates services in place. No-op when
// configs is empty or no service uses ConfigRef.
func resolveConfigRefs(services []manifest.ServiceDef, configs []manifest.StackConfig) {
	if len(configs) == 0 {
		return
	}
	idx := make(map[string]manifest.StackConfig, len(configs))
	for _, c := range configs {
		idx[c.Name] = c
	}
	for i := range services {
		if services[i].ConfigRef == "" {
			continue
		}
		sc, ok := idx[services[i].ConfigRef]
		if !ok {
			continue
		}
		// Only fill fields that are empty — inline values take precedence.
		if services[i].Language == "" {
			services[i].Language = sc.Language
		}
		if services[i].LanguageVersion == "" {
			services[i].LanguageVersion = sc.LanguageVersion
		}
		if services[i].Framework == "" {
			services[i].Framework = sc.Framework
		}
		if services[i].FrameworkVersion == "" {
			services[i].FrameworkVersion = sc.FrameworkVersion
		}
	}
}

// ── manifest cross-reference resolvers ───────────────────────────────────────

// endpointsForService filters endpoints to those whose ServiceUnit matches name.
func endpointsForService(name string, all []manifest.EndpointDef) []manifest.EndpointDef {
	out := make([]manifest.EndpointDef, 0)
	for _, e := range all {
		if e.ServiceUnit == name {
			out = append(out, e)
		}
	}
	return out
}

// externalAPIsForService returns the external APIs that belong to the given
// service (CalledByService == name) plus any that are unassigned (empty),
// since unassigned APIs are assumed shared across all services.
// Returns nil (not an empty slice) when nothing matches, so omitempty omits it.
func externalAPIsForService(name string, apis []manifest.ExternalAPIDef) []manifest.ExternalAPIDef {
	var out []manifest.ExternalAPIDef
	for _, a := range apis {
		if a.CalledByService == "" || a.CalledByService == name {
			out = append(out, a)
		}
	}
	return out
}

// fileStoragesForService returns the file storage buckets that belong to the
// given service (UsedByService == name) plus any that are unassigned (empty).
// Returns nil when nothing matches so omitempty omits it.
func fileStoragesForService(name string, storages []manifest.FileStorageDef) []manifest.FileStorageDef {
	var out []manifest.FileStorageDef
	for _, s := range storages {
		if s.UsedByService == "" || s.UsedByService == name {
			out = append(out, s)
		}
	}
	return out
}

// dtosByName resolves a slice of DTO name strings to their full DTODef objects.
// Preserves order, deduplicates by name.
func dtosByName(names []string, all []manifest.DTODef) []manifest.DTODef {
	seen := make(map[string]bool, len(names))
	idx := make(map[string]manifest.DTODef, len(all))
	for _, d := range all {
		idx[d.Name] = d
	}
	out := make([]manifest.DTODef, 0, len(names))
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		if d, ok := idx[n]; ok {
			out = append(out, d)
			seen[n] = true
		}
	}
	return out
}

// dtosForEndpoints collects the DTOs referenced by RequestDTO and ResponseDTO fields.
func dtosForEndpoints(endpoints []manifest.EndpointDef, all []manifest.DTODef) []manifest.DTODef {
	names := make([]string, 0, len(endpoints)*2)
	for _, e := range endpoints {
		names = append(names, e.RequestDTO, e.ResponseDTO)
	}
	return dtosByName(names, all)
}

// dtosForCommLinks collects the DTOs referenced in CommLink.DTOs and CommLink.ResponseDTOs.
func dtosForCommLinks(links []manifest.CommLink, all []manifest.DTODef) []manifest.DTODef {
	names := make([]string, 0)
	for _, l := range links {
		names = append(names, l.DTOs...)
		names = append(names, l.ResponseDTOs...)
	}
	return dtosByName(names, all)
}

// dtosForJobQueues collects the DTOs referenced as PayloadDTO in job queue definitions.
func dtosForJobQueues(queues []manifest.JobQueueDef, all []manifest.DTODef) []manifest.DTODef {
	names := make([]string, 0, len(queues))
	for _, q := range queues {
		names = append(names, q.PayloadDTO)
	}
	return dtosByName(names, all)
}

// dtosForEvents collects the DTOs referenced as DTO in EventDef entries.
func dtosForEvents(events []manifest.EventDef, all []manifest.DTODef) []manifest.DTODef {
	names := make([]string, 0, len(events))
	for _, e := range events {
		names = append(names, e.DTO)
	}
	return dtosByName(names, all)
}

// mergeDTOs merges multiple DTO slices, deduplicating by Name.
func mergeDTOs(slices ...[]manifest.DTODef) []manifest.DTODef {
	seen := make(map[string]bool)
	out := make([]manifest.DTODef, 0)
	for _, s := range slices {
		for _, d := range s {
			if !seen[d.Name] {
				out = append(out, d)
				seen[d.Name] = true
			}
		}
	}
	return out
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

// eventsForService returns the events where the given service is the publisher
// or consumer. Returns nil when nothing matches so omitempty omits it.
func eventsForService(name string, events []manifest.EventDef) []manifest.EventDef {
	var out []manifest.EventDef
	for _, e := range events {
		if e.PublisherService == name || e.ConsumerService == name {
			out = append(out, e)
		}
	}
	return out
}

// ── output directory helpers ─────────────────────────────────────────────────

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
	hasFrontend := m.Frontend.Tech != nil && m.Frontend.Tech.Framework != ""

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
	if m.Frontend.Tech == nil || m.Frontend.Tech.Framework == "" {
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
