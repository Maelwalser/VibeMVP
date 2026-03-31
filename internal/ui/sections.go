package ui

// FieldKind enumerates the types of form fields.
type FieldKind int

const (
	KindText     FieldKind = iota // single-line text input
	KindSelect                   // cycle through a list of options
	KindTextArea                 // multi-line text input
)

// Field represents a single form field within a section.
type Field struct {
	Key     string    // machine key (e.g. "arch_pattern")
	Label   string    // padded display label — must be exactly 14 chars
	Kind    FieldKind
	Value   string   // current string value
	Options []string // KindSelect: available choices
	SelIdx  int      // KindSelect: currently selected index
}

// DisplayValue returns the rendered value string for NORMAL mode.
func (f Field) DisplayValue() string {
	if f.Kind == KindSelect {
		if len(f.Options) == 0 {
			return ""
		}
		return f.Options[f.SelIdx]
	}
	// Show a one-line preview for textarea fields.
	v := f.Value
	if f.Kind == KindTextArea && len(v) > 0 {
		lines := splitLines(v)
		if len(lines) > 1 {
			return lines[0] + " …"
		}
	}
	return v
}

// CycleNext advances a KindSelect field to the next option.
func (f *Field) CycleNext() {
	if f.Kind != KindSelect || len(f.Options) == 0 {
		return
	}
	f.SelIdx = (f.SelIdx + 1) % len(f.Options)
	f.Value = f.Options[f.SelIdx]
}

// CyclePrev moves a KindSelect field to the previous option.
func (f *Field) CyclePrev() {
	if f.Kind != KindSelect || len(f.Options) == 0 {
		return
	}
	f.SelIdx = (f.SelIdx - 1 + len(f.Options)) % len(f.Options)
	f.Value = f.Options[f.SelIdx]
}

// Section groups related fields under a phase pillar.
type Section struct {
	ID     string  // short identifier (e.g. "domain")
	Abbr   string  // tab label
	Title  string  // full title
	Desc   string  // one-line description shown as a comment
	Fields []Field
}

// initSections returns the nine section definitions across three phases.
func initSections() []Section {
	return []Section{
		// ── Phase 1 · Universal Global Constants ──────────────────────────────

		{
			ID:    "domain",
			Abbr:  "DOMAIN",
			Title: "Phase 1 · Domain & Business Logic",
			Desc:  "Entity relationships, RBAC matrix, and compliance boundaries.",
			Fields: []Field{
				{
					Key:   "er_model",
					Label: "er_model      ",
					Kind:  KindTextArea,
				},
				{
					Key:     "cardinality",
					Label:   "cardinality   ",
					Kind:    KindSelect,
					Options: []string{"1:1", "1:N", "M:N", "mixed"},
					SelIdx:  3,
					Value:   "mixed",
				},
				{
					Key:   "cascading",
					Label: "cascading     ",
					Kind:  KindText,
				},
				{
					Key:   "rbac_matrix",
					Label: "rbac_matrix   ",
					Kind:  KindTextArea,
				},
				{
					Key:     "compliance",
					Label:   "compliance    ",
					Kind:    KindSelect,
					Options: []string{"none", "GDPR", "HIPAA", "PCI-DSS", "GDPR+HIPAA", "custom"},
				},
			},
		},

		{
			ID:    "topology",
			Abbr:  "TOPO",
			Title: "Phase 1 · System Topology & Communication",
			Desc:  "Architecture pattern and inter-domain communication contracts.",
			Fields: []Field{
				{
					Key:     "arch_pattern",
					Label:   "arch_pattern  ",
					Kind:    KindSelect,
					Options: []string{"monolith", "modular-monolith", "microservices", "event-driven"},
				},
				{
					Key:     "comm_protocol",
					Label:   "comm_protocol ",
					Kind:    KindSelect,
					Options: []string{"REST", "GraphQL", "gRPC", "WebSockets", "mixed"},
				},
				{
					Key:     "serialization",
					Label:   "serialization ",
					Kind:    KindSelect,
					Options: []string{"JSON", "Protobuf", "MessagePack", "mixed"},
				},
				{
					Key:   "domain_notes",
					Label: "domain_notes  ",
					Kind:  KindTextArea,
				},
			},
		},

		{
			ID:    "gnfr",
			Abbr:  "GNFR",
			Title: "Phase 1 · Global Non-Functional Requirements",
			Desc:  "Quantifiable SLOs and disaster recovery objectives.",
			Fields: []Field{
				{
					Key:   "uptime_slo",
					Label: "uptime_slo    ",
					Kind:  KindText,
					Value: "99.9%",
				},
				{
					Key:   "concurrent_conn",
					Label: "concurrent_con",
					Kind:  KindText,
				},
				{
					Key:   "rto",
					Label: "rto           ",
					Kind:  KindText,
				},
				{
					Key:   "rpo",
					Label: "rpo           ",
					Kind:  KindText,
				},
				{
					Key:   "nfr_notes",
					Label: "nfr_notes     ",
					Kind:  KindTextArea,
				},
			},
		},

		// ── Phase 2 · Domain-Specific Execution Paths ─────────────────────────

		{
			ID:    "backend",
			Abbr:  "BACK",
			Title: "Phase 2 · Path A: Backend Server / API",
			Desc:  "Compute environment, runtime, data persistence, queuing, and external APIs.",
			Fields: []Field{
				{
					Key:     "compute_env",
					Label:   "compute_env   ",
					Kind:    KindSelect,
					Options: []string{"serverless", "containerized", "bare-metal/VM"},
				},
				{
					Key:     "cloud_provider",
					Label:   "cloud_provider",
					Kind:    KindSelect,
					Options: []string{"N/A", "AWS", "GCP", "Azure"},
				},
				{
					Key:   "runtime",
					Label: "runtime       ",
					Kind:  KindText,
				},
				{
					Key:   "be_framework",
					Label: "be_framework  ",
					Kind:  KindText,
				},
				{
					Key:     "primary_db",
					Label:   "primary_db    ",
					Kind:    KindSelect,
					Options: []string{"PostgreSQL", "MySQL", "MongoDB", "DynamoDB", "SQLite", "other"},
				},
				{
					Key:     "cache_store",
					Label:   "cache_store   ",
					Kind:    KindSelect,
					Options: []string{"none", "Redis", "Memcached"},
				},
				{
					Key:     "cache_strategy",
					Label:   "cache_strategy",
					Kind:    KindSelect,
					Options: []string{"none", "TTL", "event-driven", "mixed"},
				},
				{
					Key:     "msg_broker",
					Label:   "msg_broker    ",
					Kind:    KindSelect,
					Options: []string{"none", "RabbitMQ", "Kafka", "SQS", "other"},
				},
				{
					Key:   "external_apis",
					Label: "external_apis ",
					Kind:  KindTextArea,
				},
			},
		},

		{
			ID:    "frontend",
			Abbr:  "FRONT",
			Title: "Phase 2 · Path B: Web Frontend",
			Desc:  "Rendering topology, framework, state management, styling, and browser matrix.",
			Fields: []Field{
				{
					Key:     "rendering",
					Label:   "rendering     ",
					Kind:    KindSelect,
					Options: []string{"SPA", "SSR", "SSG", "ISR"},
				},
				{
					Key:   "fe_framework",
					Label: "fe_framework  ",
					Kind:  KindText,
				},
				{
					Key:   "server_state",
					Label: "server_state  ",
					Kind:  KindText,
				},
				{
					Key:   "client_state",
					Label: "client_state  ",
					Kind:  KindText,
				},
				{
					Key:     "styling",
					Label:   "styling       ",
					Kind:    KindSelect,
					Options: []string{"Tailwind", "CSS-in-JS", "SASS", "CSS Modules", "other"},
				},
				{
					Key:   "browser_matrix",
					Label: "browser_matrix",
					Kind:  KindText,
				},
			},
		},

		{
			ID:    "desktop",
			Abbr:  "DESK",
			Title: "Phase 2 · Path C: Desktop Application",
			Desc:  "Target OS, app framework, hardware access, IPC, and distribution.",
			Fields: []Field{
				{
					Key:   "target_os",
					Label: "target_os     ",
					Kind:  KindText,
				},
				{
					Key:     "app_framework",
					Label:   "app_framework ",
					Kind:    KindSelect,
					Options: []string{"N/A", "Electron", "Tauri", "Qt", "native (Swift/WPF)", "other"},
				},
				{
					Key:   "hw_access",
					Label: "hw_access     ",
					Kind:  KindTextArea,
				},
				{
					Key:   "ipc_model",
					Label: "ipc_model     ",
					Kind:  KindText,
				},
				{
					Key:     "distribution",
					Label:   "distribution  ",
					Kind:    KindSelect,
					Options: []string{"N/A", ".exe (Squirrel)", ".dmg", ".AppImage", "all platforms", "other"},
				},
			},
		},

		// ── Phase 3 · Lifecycle Operations & Tooling ──────────────────────────

		{
			ID:    "testing",
			Abbr:  "TEST",
			Title: "Phase 3 · Verification & Testing",
			Desc:  "Coverage targets per test taxonomy: unit, integration, and E2E.",
			Fields: []Field{
				{
					Key:   "unit_coverage",
					Label: "unit_coverage ",
					Kind:  KindText,
					Value: "80%",
				},
				{
					Key:   "integ_coverage",
					Label: "integ_coverage",
					Kind:  KindText,
					Value: "70%",
				},
				{
					Key:     "e2e_framework",
					Label:   "e2e_framework ",
					Kind:    KindSelect,
					Options: []string{"none", "Playwright", "Cypress"},
				},
				{
					Key:   "e2e_coverage",
					Label: "e2e_coverage  ",
					Kind:  KindText,
				},
				{
					Key:   "test_strategy",
					Label: "test_strategy ",
					Kind:  KindTextArea,
				},
			},
		},

		{
			ID:    "cicd",
			Abbr:  "CICD",
			Title: "Phase 3 · CI/CD Pipeline",
			Desc:  "Automated pipeline gates, environment strategy, and secrets management.",
			Fields: []Field{
				{
					Key:     "ci_platform",
					Label:   "ci_platform   ",
					Kind:    KindSelect,
					Options: []string{"none", "GitHub Actions", "GitLab CI", "CircleCI", "Jenkins"},
				},
				{
					Key:   "pipeline_gates",
					Label: "pipeline_gates",
					Kind:  KindTextArea,
				},
				{
					Key:   "env_strategy",
					Label: "env_strategy  ",
					Kind:  KindText,
					Value: "dev / staging / prod",
				},
				{
					Key:     "secrets_mgmt",
					Label:   "secrets_mgmt  ",
					Kind:    KindSelect,
					Options: []string{"env files", "HashiCorp Vault", "AWS Secrets Manager", "GCP Secret Manager", "none"},
				},
			},
		},

		{
			ID:    "telemetry",
			Abbr:  "TELEM",
			Title: "Phase 3 · Telemetry & Observability",
			Desc:  "Logging, metrics, distributed tracing, and alerting strategy.",
			Fields: []Field{
				{
					Key:     "log_solution",
					Label:   "log_solution  ",
					Kind:    KindSelect,
					Options: []string{"other", "ELK Stack", "Datadog", "Splunk", "CloudWatch"},
				},
				{
					Key:     "log_format",
					Label:   "log_format    ",
					Kind:    KindSelect,
					Options: []string{"JSON structured", "plaintext", "mixed"},
				},
				{
					Key:     "metrics",
					Label:   "metrics       ",
					Kind:    KindSelect,
					Options: []string{"none", "Prometheus", "Datadog", "CloudWatch", "other"},
				},
				{
					Key:     "tracing",
					Label:   "tracing       ",
					Kind:    KindSelect,
					Options: []string{"none", "OpenTelemetry", "Jaeger", "Zipkin"},
				},
				{
					Key:   "alerting",
					Label: "alerting      ",
					Kind:  KindText,
				},
			},
		},
	}
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
