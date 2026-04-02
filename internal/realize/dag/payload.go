package dag

import "github.com/vibe-mvp/internal/manifest"

// TaskPayload is the scoped manifest slice fed to the agent for one task.
// Only the fields relevant to the task's Kind are populated — others are nil/zero.
// This keeps agent prompts tight and prevents cross-contamination between pillars.
type TaskPayload struct {
	ArchPattern manifest.ArchPattern
	EnvConfig   manifest.EnvConfig

	// Data pillar
	Domains      []manifest.DomainDef
	Databases    []manifest.DBSourceDef
	Cachings     []manifest.CachingConfig
	FileStorages []manifest.FileStorageDef

	// Backend pillar — per-service tasks set Service; others see AllServices.
	Service     *manifest.ServiceDef
	AllServices []manifest.ServiceDef
	CommLinks   []manifest.CommLink
	Messaging   *manifest.MessagingConfig
	APIGateway  *manifest.APIGatewayConfig
	Auth        *manifest.AuthConfig

	// Contracts pillar
	DTOs       []manifest.DTODef
	Endpoints  []manifest.EndpointDef
	Versioning manifest.APIVersioning

	// Frontend pillar
	Frontend *manifest.FrontendPillar

	// Infrastructure pillar
	Infra *manifest.InfraPillar

	// Cross-cutting pillar
	CrossCut *manifest.CrossCutPillar
}
