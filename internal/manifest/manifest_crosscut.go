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
	// PerProtocolFormats maps each API protocol (e.g. "REST", "GraphQL", "gRPC")
	// to its documentation format (e.g. "OpenAPI/Swagger", "GraphQL Playground").
	PerProtocolFormats map[string]string `json:"per_protocol_formats,omitempty"`
	AutoGenerate       bool              `json:"auto_generate"`
	Changelog          string            `json:"changelog,omitempty"`
	// APIDocs is a legacy single-format field retained for JSON backwards compatibility.
	APIDocs string `json:"api_docs,omitempty"`
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
