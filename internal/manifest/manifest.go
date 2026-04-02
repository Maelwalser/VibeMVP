package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ── Realize options ───────────────────────────────────────────────────────────

// RealizeOptions holds configuration for the code-generation agent run.
type RealizeOptions struct {
	AppName      string            `json:"app_name"`
	OutputDir    string            `json:"output_dir"`
	Model        string            `json:"model"`
	Concurrency  int               `json:"concurrency"`
	Verify       bool              `json:"verify"`
	DryRun       bool              `json:"dry_run"`
	SectionModels map[string]string `json:"section_models,omitempty"` // per-pillar model override
}

// ── Provider assignments ──────────────────────────────────────────────────────

// ProviderAssignment maps a pillar section to a specific AI provider and auth config.
type ProviderAssignment struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Version    string `json:"version"`
	Auth       string `json:"auth"`
	Credential string `json:"credential,omitempty"` // API key or OAuth token
}

// ProviderAssignments maps section IDs (backend, data, etc.) to their provider config.
type ProviderAssignments map[string]ProviderAssignment

// ── Legacy pillars (preserved for existing code compatibility) ────────────────

type TestingPillar struct {
	UnitCoverage    string       `json:"unit_coverage"`
	IntegCoverage   string       `json:"integ_coverage"`
	E2EFramework    E2EFramework `json:"e2e_framework"`
	E2ECoverage     string       `json:"e2e_coverage"`
	TestingStrategy string       `json:"testing_strategy,omitempty"`
}

type CICDPillar struct {
	CIPlatform    CIPlatform     `json:"ci_platform"`
	PipelineGates string         `json:"pipeline_gates"`
	EnvStrategy   string         `json:"env_strategy"`
	SecretsMgmt   SecretsBackend `json:"secrets_mgmt"`
}

type TelemetryPillar struct {
	LogSolution LogSolution `json:"log_solution"`
	LogFormat   string      `json:"log_format"`
	Metrics     string      `json:"metrics"`
	Tracing     string      `json:"tracing"`
	Alerting    string      `json:"alerting,omitempty"`
}

// ── Root manifest ─────────────────────────────────────────────────────────────

// Manifest is the root document holding all configuration.
type Manifest struct {
	CreatedAt time.Time `json:"created_at"`

	// Structured pillars
	Data      DataPillar      `json:"data"`
	Backend   BackendPillar   `json:"backend"`
	Contracts ContractsPillar `json:"contracts"`
	Frontend  FrontendPillar  `json:"frontend"`
	Infra     InfraPillar     `json:"infrastructure"`
	CrossCut  CrossCutPillar  `json:"cross_cutting"`
	Realize   RealizeOptions  `json:"realize,omitempty"`

	// Configured providers (from the Shift+M provider menu), keyed by provider label.
	ConfiguredProviders ProviderAssignments `json:"configured_providers,omitempty"`

	// Legacy fields kept for backward compatibility during transition.
	Databases []DBSourceDef   `json:"databases,omitempty"`
	Entities  []EntityDef     `json:"entities,omitempty"`
	Testing   TestingPillar   `json:"testing,omitempty"`
	CICD      CICDPillar      `json:"cicd,omitempty"`
	Telemetry TelemetryPillar `json:"telemetry,omitempty"`
}

// Load reads and parses a Manifest from a JSON file at path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest %s: %w", path, err)
	}
	return &m, nil
}

// Save writes the manifest to path as indented JSON.
func (m *Manifest) Save(path string) error {
	m.CreatedAt = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", path, err)
	}
	return nil
}
