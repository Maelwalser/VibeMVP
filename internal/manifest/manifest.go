package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ── Enum types ────────────────────────────────────────────────────────────────

type ArchPattern string

const (
	ArchMonolith        ArchPattern = "monolith"
	ArchModularMonolith ArchPattern = "modular-monolith"
	ArchMicroservices   ArchPattern = "microservices"
	ArchEventDriven     ArchPattern = "event-driven"
)

type CommProtocol string

const (
	ProtoREST       CommProtocol = "REST"
	ProtoGraphQL    CommProtocol = "GraphQL"
	ProtoGRPC       CommProtocol = "gRPC"
	ProtoWebSockets CommProtocol = "WebSockets"
	ProtoMixed      CommProtocol = "mixed"
)

type SerializationFmt string

const (
	SerialJSON        SerializationFmt = "JSON"
	SerialProtobuf    SerializationFmt = "Protobuf"
	SerialMessagePack SerializationFmt = "MessagePack"
	SerialMixed       SerializationFmt = "mixed"
)

type ComputeEnv string

const (
	ComputeServerless     ComputeEnv = "serverless"
	ComputeContainerized  ComputeEnv = "containerized"
	ComputeBareMetalVM    ComputeEnv = "bare-metal/VM"
)

type DatabaseType string

const (
	DBPostgres DatabaseType = "PostgreSQL"
	DBMySQL    DatabaseType = "MySQL"
	DBMongo    DatabaseType = "MongoDB"
	DBDynamo   DatabaseType = "DynamoDB"
	DBSQLite   DatabaseType = "SQLite"
	DBOther    DatabaseType = "other"
)

type CacheStore string

const (
	CacheRedis     CacheStore = "Redis"
	CacheMemcached CacheStore = "Memcached"
	CacheNone      CacheStore = "none"
)

type RenderingMode string

const (
	RenderSPA RenderingMode = "SPA"
	RenderSSR RenderingMode = "SSR"
	RenderSSG RenderingMode = "SSG"
	RenderISR RenderingMode = "ISR"
)

type DesktopFramework string

const (
	DeskElectron DesktopFramework = "Electron"
	DeskTauri    DesktopFramework = "Tauri"
	DeskQt       DesktopFramework = "Qt"
	DeskNative   DesktopFramework = "native (Swift/WPF)"
	DeskOther    DesktopFramework = "other"
)

type E2EFramework string

const (
	E2EPlaywright E2EFramework = "Playwright"
	E2ECypress    E2EFramework = "Cypress"
	E2ENone       E2EFramework = "none"
)

type CIPlatform string

const (
	CIGitHubActions CIPlatform = "GitHub Actions"
	CIGitLabCI      CIPlatform = "GitLab CI"
	CICircleCI      CIPlatform = "CircleCI"
	CIJenkins       CIPlatform = "Jenkins"
	CINone          CIPlatform = "none"
)

type SecretsBackend string

const (
	SecretsVault    SecretsBackend = "HashiCorp Vault"
	SecretsAWS      SecretsBackend = "AWS Secrets Manager"
	SecretsGCP      SecretsBackend = "GCP Secret Manager"
	SecretsEnvFiles SecretsBackend = "env files"
	SecretsNone     SecretsBackend = "none"
)

type LogSolution string

const (
	LogELK        LogSolution = "ELK Stack"
	LogDatadog    LogSolution = "Datadog"
	LogSplunk     LogSolution = "Splunk"
	LogCloudWatch LogSolution = "CloudWatch"
	LogOther      LogSolution = "other"
)

// ── Phase 1: Universal Global Constants ──────────────────────────────────────

// DomainPillar captures entity relationships, RBAC, and compliance boundaries.
type DomainPillar struct {
	EntityRelationships string `json:"entity_relationships"` // ER model description
	Cardinality         string `json:"cardinality"`          // e.g. "User 1:N Order"
	CascadingRules      string `json:"cascading_rules"`
	RBACMatrix          string `json:"rbac_matrix"`
	Compliance          string `json:"compliance"` // GDPR, HIPAA, PCI-DSS, none
}

// TopologyPillar defines the structural model and inter-domain contracts.
type TopologyPillar struct {
	ArchPattern     ArchPattern      `json:"arch_pattern"`
	CommProtocol    CommProtocol     `json:"comm_protocol"`
	Serialization   SerializationFmt `json:"serialization"`
	DomainNotes     string           `json:"domain_notes,omitempty"`
}

// GlobalNFRPillar holds SLOs and disaster recovery parameters.
type GlobalNFRPillar struct {
	UptimeSLO      string `json:"uptime_slo"`       // e.g. "99.9%"
	ConcurrentConn string `json:"concurrent_conn"`  // e.g. "5000"
	RTO            string `json:"rto"`              // Recovery Time Objective
	RPO            string `json:"rpo"`              // Recovery Point Objective
	NFRNotes       string `json:"nfr_notes,omitempty"`
}

// ── Phase 2: Domain-Specific Execution Paths ─────────────────────────────────

// BackendPillar covers server compute, runtime, databases, queues, and external APIs.
type BackendPillar struct {
	ComputeEnv    ComputeEnv   `json:"compute_env"`
	CloudProvider string       `json:"cloud_provider,omitempty"`
	Runtime       string       `json:"runtime"`        // e.g. "Go 1.23"
	Framework     string       `json:"framework"`      // e.g. "Gin", "Echo"
	PrimaryDB     DatabaseType `json:"primary_db"`
	CacheStore    CacheStore   `json:"cache_store"`
	CacheStrategy string       `json:"cache_strategy"` // TTL / event-driven / mixed
	MessageBroker string       `json:"message_broker"` // Kafka, RabbitMQ, SQS, none
	ExternalAPIs  string       `json:"external_apis"`  // retry, rate-limit, fallback notes
}

// FrontendPillar covers web rendering, framework, state management, styling, and browser support.
type FrontendPillar struct {
	Rendering     RenderingMode `json:"rendering"`
	Framework     string        `json:"framework"`      // e.g. "React 18", "Next.js 14"
	ServerState   string        `json:"server_state"`   // e.g. "React Query", "Apollo"
	ClientState   string        `json:"client_state"`   // e.g. "Zustand", "Redux"
	Styling       string        `json:"styling"`        // Tailwind, CSS-in-JS, SASS
	BrowserMatrix string        `json:"browser_matrix"` // e.g. "Chromium>100, Safari>15"
}

// DesktopPillar covers OS targets, frameworks, hardware access, IPC, and distribution.
type DesktopPillar struct {
	TargetOS      string           `json:"target_os"`      // e.g. "Windows 10/11, macOS Silicon"
	AppFramework  DesktopFramework `json:"app_framework"`
	HWAccess      string           `json:"hw_access"`      // GPU, USB, filesystem
	IPCModel      string           `json:"ipc_model"`      // main/renderer IPC security
	Distribution  string           `json:"distribution"`   // .exe, .dmg, .AppImage, OTA strategy
}

// ── Phase 3: Lifecycle Operations & Tooling ───────────────────────────────────

// TestingPillar defines coverage targets per test taxonomy.
type TestingPillar struct {
	UnitCoverage    string       `json:"unit_coverage"`    // e.g. "80%"
	IntegCoverage   string       `json:"integ_coverage"`
	E2EFramework    E2EFramework `json:"e2e_framework"`
	E2ECoverage     string       `json:"e2e_coverage"`
	TestingStrategy string       `json:"testing_strategy,omitempty"` // additional notes
}

// CICDPillar defines pipeline gates, environment strategy, and secrets management.
type CICDPillar struct {
	CIPlatform    CIPlatform     `json:"ci_platform"`
	PipelineGates string         `json:"pipeline_gates"` // blocking checks: lint, tests, vuln scan
	EnvStrategy   string         `json:"env_strategy"`   // dev/staging/prod definitions
	SecretsMgmt   SecretsBackend `json:"secrets_mgmt"`
}

// TelemetryPillar defines logging, metrics, tracing, and alerting strategy.
type TelemetryPillar struct {
	LogSolution LogSolution `json:"log_solution"`
	LogFormat   string      `json:"log_format"`  // JSON structured, plaintext
	Metrics     string      `json:"metrics"`     // Prometheus, Datadog, CloudWatch, none
	Tracing     string      `json:"tracing"`     // Jaeger, Zipkin, OpenTelemetry, none
	Alerting    string      `json:"alerting,omitempty"`
}

// ── Root manifest ─────────────────────────────────────────────────────────────

// Manifest is the root document holding all three phases.
type Manifest struct {
	CreatedAt time.Time `json:"created_at"`

	// Phase 1 – Universal Global Constants
	Domain    DomainPillar    `json:"domain"`
	Topology  TopologyPillar  `json:"topology"`
	GlobalNFR GlobalNFRPillar `json:"global_nfr"`

	// Phase 2 – Domain-Specific Execution Paths
	Backend  BackendPillar  `json:"backend"`
	Frontend FrontendPillar `json:"frontend"`
	Desktop  DesktopPillar  `json:"desktop"`

	// Phase 3 – Lifecycle Operations & Tooling
	Testing   TestingPillar   `json:"testing"`
	CICD      CICDPillar      `json:"cicd"`
	Telemetry TelemetryPillar `json:"telemetry"`
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
