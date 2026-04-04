package manifest

// ── Cross-cutting tab types ───────────────────────────────────────────────────

// TestingConfig describes testing strategy and tool choices.
type TestingConfig struct {
	Unit              string `json:"unit"`
	Integration       string `json:"integration"`
	E2E               string `json:"e2e"`
	API               string `json:"api"`
	Load              string `json:"load"`
	Contract          string `json:"contract"`
	FrontendTesting   string `json:"frontend_testing,omitempty"`
}

// DocsConfig describes documentation tooling.
type DocsConfig struct {
	APIDocs      string `json:"api_docs"`
	AutoGenerate bool   `json:"auto_generate"`
}

// CrossCutPillar groups cross-cutting concerns.
type CrossCutPillar struct {
	Testing           TestingConfig `json:"testing"`
	Docs              DocsConfig    `json:"docs"`
	DependencyUpdates string        `json:"dependency_updates,omitempty"`
	FeatureFlags      string        `json:"feature_flags,omitempty"`
	UptimeSLO         string        `json:"uptime_slo,omitempty"`
	LatencyP99        string        `json:"latency_p99,omitempty"`
	BackendLinter     string        `json:"backend_linter,omitempty"`
	FrontendLinter    string        `json:"frontend_linter,omitempty"`
}
