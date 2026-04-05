package orchestrator

import "github.com/vibe-menu/internal/realize/dag"

// ModelTier represents an abstract intelligence level for a generation task.
// The same tier maps to different model IDs depending on the active provider.
type ModelTier int

const (
	// TierFast maps to the fastest/cheapest model class (e.g. Haiku, gpt-4o-mini, Gemini Flash).
	// Use for boilerplate tasks with well-understood output formats.
	TierFast ModelTier = iota
	// TierMedium maps to the balanced reasoning class (e.g. Sonnet, gpt-4o, Gemini Pro).
	// Use for tasks requiring multi-file reasoning or correctness guarantees.
	TierMedium
	// TierSlow maps to the highest-capability class (e.g. Opus, o1, Gemini Ultra).
	// Reached only via escalation on repeated verification failures.
	TierSlow
)

// defaultTierForKind maps each task kind to its baseline intelligence requirement.
// Simple/boilerplate tasks use TierFast; medium-complexity tasks use TierMedium.
// All tiers escalate toward TierSlow on repeated retry failures.
//
// Rationale:
//   - TierFast: contracts, docs, docker, CI — well-understood output formats with
//     minimal reasoning required; ~87% of Opus quality at ~20% of the cost.
//   - TierMedium: services, auth, data, frontend, terraform, testing — require
//     multi-file reasoning and correctness guarantees.
//   - TierSlow: reached via escalation only; reserved for tasks that fail TierMedium twice.
var defaultTierForKind = map[dag.TaskKind]ModelTier{
	dag.TaskKindDataSchemas:    TierMedium,
	dag.TaskKindDataMigrations: TierFast, // pure SQL, no reasoning needed

	// Plan task: architectural reasoning about interfaces and dependency versions.
	dag.TaskKindServicePlan: TierMedium,

	// Service layers: each is a focused unit; TierFast handles repetitive patterns,
	// TierMedium handles logic and integration complexity.
	dag.TaskKindServiceRepository: TierFast,   // repetitive CRUD boilerplate
	dag.TaskKindServiceLogic:      TierMedium, // business rules need reasoning
	dag.TaskKindServiceHandler:    TierMedium, // routing + auth integration
	dag.TaskKindServiceBootstrap:  TierFast,   // wiring boilerplate

	dag.TaskKindAuth:      TierMedium,
	dag.TaskKindMessaging: TierMedium,
	dag.TaskKindGateway:   TierMedium,

	dag.TaskKindContracts:       TierFast,
	dag.TaskKindFrontend:        TierMedium,
	dag.TaskKindInfraDocker:     TierFast,
	dag.TaskKindInfraTerraform:  TierMedium,
	dag.TaskKindInfraCI:         TierFast,
	dag.TaskKindCrossCutTesting: TierMedium,
	dag.TaskKindCrossCutDocs:    TierFast,

	// Reconciliation runs at TierSlow: it must reason across ALL generated files
	// in the backend module to identify the root cause of cross-task compile errors.
	dag.TaskKindReconciliation: TierSlow,

	// Integration repair runs at TierSlow: it must reason across multiple files
	// to identify the root cause of cross-task compilation failures.
	dag.TaskKindIntegrationRepair: TierSlow,
}

// tierForKind returns the default ModelTier for a task kind.
// Falls back to TierMedium for unknown kinds.
func tierForKind(kind dag.TaskKind) ModelTier {
	if tier, ok := defaultTierForKind[kind]; ok {
		return tier
	}
	return TierMedium
}

// escalateTier returns the next tier up when a task fails verification.
// Escalation path: TierFast → TierMedium → TierSlow.
// Returns (TierSlow, false) when already at the maximum tier.
func escalateTier(tier ModelTier) (ModelTier, bool) {
	switch tier {
	case TierFast:
		return TierMedium, true
	case TierMedium:
		return TierSlow, true
	default:
		return TierSlow, false
	}
}
