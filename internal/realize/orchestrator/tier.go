package orchestrator

import "github.com/vibe-menu/internal/realize/dag"

// defaultTierForKind maps each task kind to its default Claude model ID.
// Simple/boilerplate tasks use Haiku; medium-complexity tasks use Sonnet.
// All tiers escalate toward Opus on repeated retry failures.
//
// Rationale:
//   - Haiku: contracts, docs, docker, CI — well-understood output formats with
//     minimal reasoning required; 87% of Opus quality at ~20% of the cost.
//   - Sonnet: services, auth, data, frontend, terraform, testing — require
//     multi-file reasoning and correctness guarantees.
//   - Opus: reached via escalation only; reserved for tasks that fail Sonnet twice.
var defaultTierForKind = map[dag.TaskKind]string{
	dag.TaskKindDataSchemas:    "claude-sonnet-4-6",
	dag.TaskKindDataMigrations: "claude-haiku-4-5-20251001", // pure SQL, no reasoning needed

	// Plan task: architectural reasoning about interfaces and dependency versions.
	dag.TaskKindServicePlan: "claude-sonnet-4-6",

	// Service layers: each is a focused unit; Haiku handles repetitive patterns,
	// Sonnet handles logic and integration complexity.
	dag.TaskKindServiceRepository: "claude-haiku-4-5-20251001", // repetitive CRUD boilerplate
	dag.TaskKindServiceLogic:      "claude-sonnet-4-6",         // business rules need reasoning
	dag.TaskKindServiceHandler:    "claude-sonnet-4-6",         // routing + auth integration
	dag.TaskKindServiceBootstrap:  "claude-haiku-4-5-20251001", // wiring boilerplate

	dag.TaskKindAuth:     "claude-sonnet-4-6",
	dag.TaskKindMessaging: "claude-sonnet-4-6",
	dag.TaskKindGateway:  "claude-sonnet-4-6",

	dag.TaskKindContracts:       "claude-haiku-4-5-20251001",
	dag.TaskKindFrontend:        "claude-sonnet-4-6",
	dag.TaskKindInfraDocker:     "claude-haiku-4-5-20251001",
	dag.TaskKindInfraTerraform:  "claude-sonnet-4-6",
	dag.TaskKindInfraCI:         "claude-haiku-4-5-20251001",
	dag.TaskKindCrossCutTesting: "claude-sonnet-4-6",
	dag.TaskKindCrossCutDocs:    "claude-haiku-4-5-20251001",
}

// tierForKind returns the default Claude model ID for a task kind.
// Falls back to defaultModel (Sonnet) for unknown kinds.
func tierForKind(kind dag.TaskKind) string {
	if model, ok := defaultTierForKind[kind]; ok {
		return model
	}
	return defaultModel
}

// escalateModel returns the next model tier up when a task fails verification.
// Escalation path: Haiku → Sonnet → Opus.
// On attempt 0 the base model is returned unchanged.
func escalateModel(baseModel string, attempt int) string {
	if attempt == 0 {
		return baseModel
	}
	switch baseModel {
	case "claude-haiku-4-5-20251001":
		return "claude-sonnet-4-6"
	case "claude-sonnet-4-6":
		if attempt >= 2 {
			return "claude-opus-4-6"
		}
		return "claude-sonnet-4-6"
	default:
		return baseModel
	}
}
