package dag

import "github.com/vibe-menu/internal/manifest"

// TaskPayload is the scoped manifest slice fed to the agent for one task.
// Only the fields relevant to the task's Kind are populated — others are nil/zero.
// This keeps agent prompts tight and prevents cross-contamination between pillars.
// omitempty tags ensure nil/zero fields are omitted from JSON to reduce token usage.
type TaskPayload struct {
	// ModulePath is the Go module path for this service (e.g. "core-api").
	// Derived deterministically from the service name and shared across all
	// sub-tasks so every layer uses identical import paths.
	ModulePath  string               `json:"module_path,omitempty"`
	ArchPattern manifest.ArchPattern `json:"arch_pattern,omitempty"`
	EnvConfig   manifest.EnvConfig   `json:"env_config,omitempty"`

	// Data pillar
	Domains      []manifest.DomainDef      `json:"domains,omitempty"`
	Databases    []manifest.DBSourceDef    `json:"databases,omitempty"`
	Cachings     []manifest.CachingConfig  `json:"cachings,omitempty"`
	FileStorages []manifest.FileStorageDef `json:"file_storages,omitempty"`

	// Backend pillar — per-service tasks set Service; others see AllServices.
	Service     *manifest.ServiceDef       `json:"service,omitempty"`
	AllServices []manifest.ServiceDef      `json:"all_services,omitempty"`
	CommLinks   []manifest.CommLink        `json:"comm_links,omitempty"`
	Messaging   *manifest.MessagingConfig  `json:"messaging,omitempty"`
	APIGateway  *manifest.APIGatewayConfig `json:"api_gateway,omitempty"`
	Auth        *manifest.AuthConfig       `json:"auth,omitempty"`

	// JobQueues and CronJobs are the background task configurations assigned to
	// this service. Populated for service-layer and bootstrap tasks so agents
	// know which job technologies to wire in.
	JobQueues []manifest.JobQueueDef `json:"job_queues,omitempty"`
	CronJobs  []manifest.CronJobDef  `json:"cron_jobs,omitempty"`

	// Contracts pillar
	DTOs         []manifest.DTODef         `json:"dtos,omitempty"`
	Endpoints    []manifest.EndpointDef    `json:"endpoints,omitempty"`
	Versioning   manifest.APIVersioning    `json:"versioning,omitempty"`
	ExternalAPIs []manifest.ExternalAPIDef `json:"external_apis,omitempty"`

	// Frontend pillar
	Frontend *manifest.FrontendPillar `json:"frontend,omitempty"`

	// ServiceDirs maps each service slug to the directory containing its generated
	// source files, relative to the output root. Used by infra tasks to set the
	// correct Docker build context without guessing or inventing subdirectories.
	// For a monolith with a frontend: {"monolith": "backend", "frontend": "frontend"}.
	// For microservices: {"user-api": "services/user-api", "frontend": "frontend"}.
	// For a backend-only monolith: {"monolith": "."} — all files at the output root.
	ServiceDirs map[string]string `json:"service_dirs,omitempty"`

	// OutputDir is the directory (relative to the output root) where this task's
	// generated files will be placed. Empty or "." means the output root.
	// Example: "backend", "frontend", "services/user-api".
	// Agents output paths relative to their component — the pipeline applies this
	// prefix at commit time. Temp-dir verification always stays prefix-free.
	OutputDir string `json:"output_dir,omitempty"`

	// Infrastructure pillar
	Infra *manifest.InfraPillar `json:"infra,omitempty"`

	// Cross-cutting pillar
	CrossCut *manifest.CrossCutPillar `json:"cross_cut,omitempty"`
}
