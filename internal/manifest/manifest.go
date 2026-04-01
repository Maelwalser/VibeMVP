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
	ArchHybrid          ArchPattern = "hybrid"
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
	ComputeServerless    ComputeEnv = "serverless"
	ComputeContainerized ComputeEnv = "containerized"
	ComputeBareMetalVM   ComputeEnv = "bare-metal/VM"
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

// ── Database source definitions ───────────────────────────────────────────────

// DBSourceDef describes a named database or cache source used in the project.
type DBSourceDef struct {
	Alias     string       `json:"alias"`
	Type      DatabaseType `json:"type"`
	Version   string       `json:"version,omitempty"`
	Namespace string       `json:"namespace,omitempty"`
	IsCache   bool         `json:"is_cache"`
	Notes     string       `json:"notes,omitempty"`
}

// ── Column / Entity definitions ───────────────────────────────────────────────

type ColumnType string

const (
	ColTypeText        ColumnType = "text"
	ColTypeVarchar     ColumnType = "varchar"
	ColTypeChar        ColumnType = "char"
	ColTypeInt         ColumnType = "int"
	ColTypeBigInt      ColumnType = "bigint"
	ColTypeSmallInt    ColumnType = "smallint"
	ColTypeSerial      ColumnType = "serial"
	ColTypeBigSerial   ColumnType = "bigserial"
	ColTypeBoolean     ColumnType = "boolean"
	ColTypeFloat       ColumnType = "float"
	ColTypeDouble      ColumnType = "double"
	ColTypeDecimal     ColumnType = "decimal"
	ColTypeJSON        ColumnType = "json"
	ColTypeJSONB       ColumnType = "jsonb"
	ColTypeUUID        ColumnType = "uuid"
	ColTypeTimestamp   ColumnType = "timestamp"
	ColTypeTimestampTZ ColumnType = "timestamptz"
	ColTypeDate        ColumnType = "date"
	ColTypeTime        ColumnType = "time"
	ColTypeBytea       ColumnType = "bytea"
	ColTypeEnum        ColumnType = "enum"
	ColTypeArray       ColumnType = "array"
	ColTypeOther       ColumnType = "other"
)

type CascadeAction string

const (
	CascadeNoAction   CascadeAction = "NO ACTION"
	CascadeRestrict   CascadeAction = "RESTRICT"
	CascadeCascade    CascadeAction = "CASCADE"
	CascadeSetNull    CascadeAction = "SET NULL"
	CascadeSetDefault CascadeAction = "SET DEFAULT"
)

type IndexType string

const (
	IndexBTree IndexType = "btree"
	IndexHash  IndexType = "hash"
	IndexGIN   IndexType = "gin"
	IndexGIST  IndexType = "gist"
	IndexBRIN  IndexType = "brin"
)

type ForeignKey struct {
	RefEntity string        `json:"ref_entity"`
	RefColumn string        `json:"ref_column"`
	OnDelete  CascadeAction `json:"on_delete"`
	OnUpdate  CascadeAction `json:"on_update"`
}

type ColumnDef struct {
	Name       string      `json:"name"`
	Type       ColumnType  `json:"type"`
	Length     string      `json:"length,omitempty"`
	Nullable   bool        `json:"nullable"`
	PrimaryKey bool        `json:"primary_key"`
	Unique     bool        `json:"unique"`
	Default    string      `json:"default,omitempty"`
	Check      string      `json:"check,omitempty"`
	ForeignKey *ForeignKey `json:"foreign_key,omitempty"`
	Index      bool        `json:"index"`
	IndexType  IndexType   `json:"index_type,omitempty"`
	Notes      string      `json:"notes,omitempty"`
}

type UniqueConstraint struct {
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns"`
}

type EntityDef struct {
	Name        string `json:"name"`
	Database    string `json:"database,omitempty"`
	Description string `json:"description,omitempty"`

	Cached     bool   `json:"cached"`
	CacheStore string `json:"cache_store,omitempty"`
	CacheTTL   string `json:"cache_ttl,omitempty"`

	Columns           []ColumnDef        `json:"columns"`
	UniqueConstraints []UniqueConstraint `json:"unique_constraints,omitempty"`
	Notes             string             `json:"notes,omitempty"`
}

// ── Phase 1: Universal Global Constants ──────────────────────────────────────

type DomainPillar struct {
	Entities   []EntityDef `json:"entities,omitempty"`
	RBACMatrix string      `json:"rbac_matrix"`
	Compliance string      `json:"compliance"`
}

type TopologyPillar struct {
	ArchPattern   ArchPattern      `json:"arch_pattern"`
	CommProtocol  CommProtocol     `json:"comm_protocol"`
	Serialization SerializationFmt `json:"serialization"`
	DomainNotes   string           `json:"domain_notes,omitempty"`
}

type GlobalNFRPillar struct {
	UptimeSLO      string `json:"uptime_slo"`
	ConcurrentConn string `json:"concurrent_conn"`
	RTO            string `json:"rto"`
	RPO            string `json:"rpo"`
	NFRNotes       string `json:"nfr_notes,omitempty"`
}

// ── Backend types ─────────────────────────────────────────────────────────────

// ServiceDef represents one backend module or microservice.
type ServiceDef struct {
	Name             string             `json:"name"`
	Responsibility   string             `json:"responsibility"`
	Language         string             `json:"language"`
	Framework        string             `json:"framework"`
	PatternTag       string             `json:"pattern_tag,omitempty"` // hybrid only
	Technologies     []string           `json:"technologies,omitempty"`
	HealthcheckPath  string             `json:"healthcheck_path,omitempty"`
	ErrorFormat      string             `json:"error_format,omitempty"`
	ServiceDiscovery string             `json:"service_discovery,omitempty"`
	Interfaces       []ExposedInterface `json:"interfaces,omitempty"`
}

// ExposedInterface describes one interface a service unit exposes.
type ExposedInterface struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// CommLink describes a directed communication link between two service units.
type CommLink struct {
	From               string   `json:"from"`
	To                 string   `json:"to"`
	Direction          string   `json:"direction"`
	Protocol           string   `json:"protocol"`
	Trigger            string   `json:"trigger,omitempty"`
	SyncAsync          string   `json:"sync_async"`
	ResiliencePatterns []string `json:"resilience_patterns,omitempty"`
}

// MessagingConfig describes the message broker configuration.
type MessagingConfig struct {
	BrokerTech    string `json:"broker_tech"`
	Deployment    string `json:"deployment"`
	Serialization string `json:"serialization"`
	Delivery      string `json:"delivery"`
}

// EventDef describes a single entry in the event catalog.
type EventDef struct {
	Name        string `json:"name"`
	Domain      string `json:"domain,omitempty"`
	Description string `json:"description,omitempty"`
}

// APIGatewayConfig describes API gateway configuration.
type APIGatewayConfig struct {
	Technology string `json:"technology"`
	Routing    string `json:"routing"`
	Features   string `json:"features,omitempty"`
}

// AuthConfig describes authentication and identity settings.
type AuthConfig struct {
	Strategy     string `json:"strategy"`
	Provider     string `json:"provider"`
	AuthzModel   string `json:"authz_model"`
	TokenStorage string `json:"token_storage"`
	MFA          string `json:"mfa"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Roles        string `json:"roles,omitempty"`
}

// WAFConfig describes Web Application Firewall and rate limiting settings.
type WAFConfig struct {
	Provider          string `json:"provider,omitempty"`
	Ruleset           string `json:"ruleset,omitempty"`
	CAPTCHA           string `json:"captcha,omitempty"`
	BotProtection     string `json:"bot_protection,omitempty"`
	RateLimitStrategy string `json:"rate_limit_strategy,omitempty"`
	RateLimitBackend  string `json:"rate_limit_backend,omitempty"`
	DDoSProtection    string `json:"ddos_protection,omitempty"`
}

// CronJobDef describes a scheduled/cron job.
type CronJobDef struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Handler  string `json:"handler,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

// JobQueueDef describes a worker pool or task queue.
type JobQueueDef struct {
	Name        string       `json:"name"`
	Technology  string       `json:"technology"`
	Concurrency string       `json:"concurrency,omitempty"`
	MaxRetries  string       `json:"max_retries,omitempty"`
	RetryPolicy string       `json:"retry_policy"`
	DLQ         string       `json:"dlq,omitempty"`
	CronJobs    []CronJobDef `json:"cron_jobs,omitempty"`
}

// EnvConfig describes the deployment environment configuration.
type EnvConfig struct {
	ComputeEnv    string `json:"compute_env"`
	CloudProvider string `json:"cloud_provider"`
	Orchestrator  string `json:"orchestrator"`
	Regions       string `json:"regions,omitempty"`
	Stages        string `json:"stages,omitempty"`
}

// BackendPillar covers the full backend configuration.
type BackendPillar struct {
	ArchPattern  ArchPattern       `json:"arch_pattern"`
	Env          EnvConfig         `json:"env"`
	Services     []ServiceDef      `json:"services,omitempty"`
	CommLinks    []CommLink        `json:"comm_links,omitempty"`
	Messaging    *MessagingConfig  `json:"messaging,omitempty"`
	APIGateway   *APIGatewayConfig `json:"api_gateway,omitempty"`
	Auth         AuthConfig        `json:"auth"`
	JobQueues    []JobQueueDef     `json:"job_queues,omitempty"`
	WAF          WAFConfig         `json:"waf,omitempty"`
	CORSStrategy  string            `json:"cors_strategy,omitempty"`
	CORSOrigins   string            `json:"cors_origins,omitempty"`
	SessionMgmt   string            `json:"session_mgmt,omitempty"`
	BackendLinter string            `json:"backend_linter,omitempty"`

	// Legacy monolith fields kept for backward compatibility.
	ComputeEnv    ComputeEnv `json:"compute_env,omitempty"`
	CloudProvider string     `json:"cloud_provider,omitempty"`
	Language      string     `json:"language,omitempty"`
	Framework     string     `json:"framework,omitempty"`
}

// ── Data tab types ────────────────────────────────────────────────────────────

// DomainDef is the new concept of a bounded-context domain (not a DB entity/column).
type DomainDef struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Databases   string              `json:"databases,omitempty"`
	Attributes  []DomainAttribute   `json:"attributes,omitempty"`
	Relationships []DomainRelationship `json:"relationships,omitempty"`
}

type DomainAttribute struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Constraints string `json:"constraints,omitempty"`
	Default     string `json:"default,omitempty"`
	Sensitive   bool   `json:"sensitive"`
	Validation  string `json:"validation,omitempty"`
	Indexed     bool   `json:"indexed,omitempty"`
	Unique      bool   `json:"unique,omitempty"`
}

type DomainRelationship struct {
	RelatedDomain string `json:"related_domain"`
	RelType       string `json:"rel_type"`
	ForeignKey    string `json:"foreign_key,omitempty"`
	Cascade       string `json:"cascade,omitempty"`
}

// CachingConfig describes the application-level caching strategy.
type CachingConfig struct {
	Layer        string `json:"layer"`
	Strategy     string `json:"strategy"`
	Invalidation string `json:"invalidation"`
	TTL          string `json:"ttl,omitempty"`
	Entities     string `json:"entities,omitempty"`
}

// FileStorageDef describes a file/object storage bucket.
type FileStorageDef struct {
	Technology   string `json:"technology"`
	Purpose      string `json:"purpose,omitempty"`
	Access       string `json:"access"`
	MaxSize      string `json:"max_size,omitempty"`
	Domains      string `json:"domains,omitempty"`
	TTLMinutes   string `json:"ttl_minutes,omitempty"`
	AllowedTypes string `json:"allowed_types,omitempty"`
}

// DataGovernanceConfig describes data lifecycle, privacy, and compliance settings.
type DataGovernanceConfig struct {
	RetentionPolicy      string `json:"retention_policy,omitempty"`
	DeleteStrategy       string `json:"delete_strategy,omitempty"`
	PIIEncryption        string `json:"pii_encryption,omitempty"`
	ComplianceFrameworks string `json:"compliance_frameworks,omitempty"`
	DataResidency        string `json:"data_residency,omitempty"`
	ArchivalStorage      string `json:"archival_storage,omitempty"`
}

// DataPillar groups all data-related configuration.
type DataPillar struct {
	Databases        []DBSourceDef        `json:"databases,omitempty"`
	Domains          []DomainDef          `json:"domains,omitempty"`
	Entities         []EntityDef          `json:"entities,omitempty"` // legacy
	Caching          CachingConfig        `json:"caching"`
	FileStorages     []FileStorageDef     `json:"file_storages,omitempty"`
	Governance       DataGovernanceConfig `json:"governance,omitempty"`
	MigrationTool    string               `json:"migration_tool,omitempty"`
	BackupStrategy   string               `json:"backup_strategy,omitempty"`
	SearchTech       string               `json:"search_tech,omitempty"`
	SearchableDomains []string            `json:"searchable_domains,omitempty"`
}

// ── Contracts tab types ───────────────────────────────────────────────────────

// DTOField describes a single field within a DTO.
type DTOField struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Nullable   bool   `json:"nullable"`
	Validation string `json:"validation,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// DTODef describes a Data Transfer Object.
type DTODef struct {
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	SourceDomains string     `json:"source_domains,omitempty"`
	Description   string     `json:"description,omitempty"`
	Fields        []DTOField `json:"fields,omitempty"`
}

// EndpointDef describes an API endpoint or operation.
type EndpointDef struct {
	ServiceUnit        string `json:"service_unit"`
	NamePath           string `json:"name_path"`
	Protocol           string `json:"protocol"`
	AuthRequired       string `json:"auth_required"`
	RequestDTO         string `json:"request_dto,omitempty"`
	ResponseDTO        string `json:"response_dto,omitempty"`
	HTTPMethod         string `json:"http_method,omitempty"`
	Description        string `json:"description,omitempty"`
	GraphQLOpType      string `json:"graphql_op_type,omitempty"`
	GRPCStreamType     string `json:"grpc_stream_type,omitempty"`
	WSDirection        string `json:"ws_direction,omitempty"`
	PaginationStrategy string `json:"pagination_strategy,omitempty"`
	RateLimit          string `json:"rate_limit,omitempty"`
}

// APIVersioning describes how the API handles versioning.
type APIVersioning struct {
	Strategy           string `json:"strategy"`
	CurrentVersion     string `json:"current_version,omitempty"`
	DeprecationPolicy  string `json:"deprecation_policy,omitempty"`
	PaginationStrategy string `json:"pagination_strategy,omitempty"`
}

// ExternalAPIDef describes a third-party API that the system consumes.
type ExternalAPIDef struct {
	Provider        string `json:"provider"`
	AuthMechanism   string `json:"auth_mechanism"`
	RateLimit       string `json:"rate_limit,omitempty"`
	WebhookEndpoint string `json:"webhook_endpoint,omitempty"`
	FailureStrategy string `json:"failure_strategy"`
	BaseURL         string `json:"base_url,omitempty"`
}

// ContractsPillar groups all contract-related configuration.
type ContractsPillar struct {
	DTOs         []DTODef        `json:"dtos,omitempty"`
	Endpoints    []EndpointDef   `json:"endpoints,omitempty"`
	Versioning   APIVersioning   `json:"versioning"`
	ExternalAPIs []ExternalAPIDef `json:"external_apis,omitempty"`
}

// ── Frontend tab types ────────────────────────────────────────────────────────

// FrontendTechConfig describes the technology stack choices for the frontend.
type FrontendTechConfig struct {
	Language           string `json:"language"`
	Platform           string `json:"platform"`
	Framework          string `json:"framework"`
	MetaFramework      string `json:"meta_framework,omitempty"`
	PackageManager     string `json:"package_manager"`
	Styling            string `json:"styling"`
	ComponentLib       string `json:"component_lib,omitempty"`
	StateManagement    string `json:"state_management,omitempty"`
	DataFetching       string `json:"data_fetching,omitempty"`
	FormHandling       string `json:"form_handling,omitempty"`
	Validation         string `json:"validation,omitempty"`
	PWASupport         string `json:"pwa_support,omitempty"`
	RealtimeStrategy   string `json:"realtime_strategy,omitempty"`
	ImageOptimization  string `json:"image_optimization,omitempty"`
	AuthFlowType       string `json:"auth_flow_type,omitempty"`
	ErrorBoundary      string `json:"error_boundary,omitempty"`
	BundleOptimization string `json:"bundle_optimization,omitempty"`
	FrontendTesting    string `json:"frontend_testing,omitempty"`
	FrontendLinter     string `json:"frontend_linter,omitempty"`
}

// FrontendTheme describes the visual theme settings.
type FrontendTheme struct {
	DarkMode     string `json:"dark_mode"`
	BorderRadius string `json:"border_radius"`
	Spacing      string `json:"spacing"`
	Elevation    string `json:"elevation"`
	Motion       string `json:"motion"`
	Vibe         string `json:"vibe,omitempty"`
	Colors       string `json:"colors,omitempty"`
	Description  string `json:"description,omitempty"`
}

// PageDef describes a frontend page.
type PageDef struct {
	Name          string `json:"name"`
	Route         string `json:"route"`
	AuthRequired  string `json:"auth_required"`
	Layout        string `json:"layout"`
	Description   string `json:"description,omitempty"`
	CoreActions   string `json:"core_actions,omitempty"`
	Loading       string `json:"loading"`
	ErrorHandling string `json:"error_handling"`
	AuthRoles     string `json:"auth_roles,omitempty"`
	LinkedPages   string `json:"linked_pages,omitempty"`
}

// NavigationConfig describes frontend navigation settings.
type NavigationConfig struct {
	NavType     string `json:"nav_type"`
	Breadcrumbs bool   `json:"breadcrumbs"`
	AuthAware   bool   `json:"auth_aware"`
}

// I18nConfig describes internationalization and localization settings.
type I18nConfig struct {
	Enabled             string `json:"enabled,omitempty"`
	DefaultLocale       string `json:"default_locale,omitempty"`
	SupportedLocales    string `json:"supported_locales,omitempty"`
	TranslationStrategy string `json:"translation_strategy,omitempty"`
	TimezoneHandling    string `json:"timezone_handling,omitempty"`
}

// A11ySEOConfig describes accessibility and SEO settings.
type A11ySEOConfig struct {
	WCAGLevel         string `json:"wcag_level,omitempty"`
	SEORenderStrategy string `json:"seo_render_strategy,omitempty"`
	Sitemap           string `json:"sitemap,omitempty"`
	MetaTagInjection  string `json:"meta_tag_injection,omitempty"`
	Analytics         string `json:"analytics,omitempty"`
	Telemetry         string `json:"telemetry,omitempty"`
}

// FrontendPillar covers the full frontend configuration.
type FrontendPillar struct {
	Tech       FrontendTechConfig `json:"tech"`
	Theme      FrontendTheme      `json:"theme"`
	Pages      []PageDef          `json:"pages,omitempty"`
	Navigation NavigationConfig   `json:"navigation"`
	I18n       I18nConfig         `json:"i18n,omitempty"`
	A11ySEO    A11ySEOConfig      `json:"a11y_seo,omitempty"`

	// Legacy fields preserved for backward compatibility.
	Rendering     RenderingMode `json:"rendering,omitempty"`
	Framework     string        `json:"framework,omitempty"`
	ServerState   string        `json:"server_state,omitempty"`
	ClientState   string        `json:"client_state,omitempty"`
	Styling       string        `json:"styling,omitempty"`
	BrowserMatrix string        `json:"browser_matrix,omitempty"`
}

// ── Infrastructure tab types ──────────────────────────────────────────────────

// NetworkingConfig describes networking and connectivity settings.
type NetworkingConfig struct {
	DNSProvider    string `json:"dns_provider"`
	TLSSSL         string `json:"tls_ssl"`
	ReverseProxy   string `json:"reverse_proxy"`
	CDN            string `json:"cdn"`
	PrimaryDomain  string `json:"primary_domain,omitempty"`
	DomainStrategy string `json:"domain_strategy,omitempty"`
	CORSEnforcement string `json:"cors_enforcement,omitempty"`
	SSLCertMgmt    string `json:"ssl_cert_mgmt,omitempty"`
}

// CICDConfig describes CI/CD pipeline settings.
type CICDConfig struct {
	Platform          string `json:"platform"`
	ContainerRegistry string `json:"container_registry"`
	DeployStrategy    string `json:"deploy_strategy"`
	IaCTool           string `json:"iac_tool"`
	SecretsMgmt       string `json:"secrets_mgmt,omitempty"`
	ContainerRuntime  string `json:"container_runtime,omitempty"`
	BackupDR          string `json:"backup_dr,omitempty"`
}

// ObservabilityConfig describes logging, metrics, tracing, and alerting settings.
type ObservabilityConfig struct {
	Logging       string `json:"logging"`
	Metrics       string `json:"metrics"`
	Tracing       string `json:"tracing"`
	ErrorTracking string `json:"error_tracking"`
	HealthChecks  bool   `json:"health_checks"`
	Alerting      string `json:"alerting"`
	LogRetention  string `json:"log_retention,omitempty"`
}

// EnvTopologyConfig describes environment staging, promotion, and secret topology.
type EnvTopologyConfig struct {
	Stages            string `json:"stages,omitempty"`
	PromotionPipeline string `json:"promotion_pipeline,omitempty"`
	SecretKeyStrategy string `json:"secret_key_strategy,omitempty"`
	MigrationStrategy string `json:"migration_strategy,omitempty"`
	DBSeeding         string `json:"db_seeding,omitempty"`
	PreviewEnvs       string `json:"preview_envs,omitempty"`
}

// InfraPillar groups infrastructure configuration.
type InfraPillar struct {
	Networking    NetworkingConfig    `json:"networking"`
	CICD          CICDConfig          `json:"cicd"`
	Observability ObservabilityConfig `json:"observability"`
	EnvTopology   EnvTopologyConfig   `json:"env_topology,omitempty"`
}

// ── Cross-cutting tab types ───────────────────────────────────────────────────

// TestingConfig describes testing strategy and tool choices.
type TestingConfig struct {
	Unit        string `json:"unit"`
	Integration string `json:"integration"`
	E2E         string `json:"e2e"`
	API         string `json:"api"`
	Load        string `json:"load"`
	Contract    string `json:"contract"`
}

// DocsConfig describes documentation tooling.
type DocsConfig struct {
	APIDocs      string `json:"api_docs"`
	AutoGenerate bool   `json:"auto_generate"`
	Changelog    string `json:"changelog"`
}

// CrossCutPillar groups cross-cutting concerns.
type CrossCutPillar struct {
	Testing           TestingConfig `json:"testing"`
	Docs              DocsConfig    `json:"docs"`
	BranchStrategy    string        `json:"branch_strategy,omitempty"`
	DependencyUpdates string        `json:"dependency_updates,omitempty"`
	CodeReview        string        `json:"code_review,omitempty"`
	FeatureFlags      string        `json:"feature_flags,omitempty"`
	UptimeSLO         string        `json:"uptime_slo,omitempty"`
	LatencyP99        string        `json:"latency_p99,omitempty"`
}

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

// ── Realize options ───────────────────────────────────────────────────────────

// RealizeOptions holds configuration for the code-generation agent run.
type RealizeOptions struct {
	AppName     string `json:"app_name"`
	OutputDir   string `json:"output_dir"`
	Model       string `json:"model"`
	Concurrency int    `json:"concurrency"`
	Verify      bool   `json:"verify"`
	DryRun      bool   `json:"dry_run"`
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

	// Legacy fields kept for backward compatibility during transition.
	Databases []DBSourceDef `json:"databases,omitempty"`
	Entities  []EntityDef   `json:"entities,omitempty"`
	Testing   TestingPillar   `json:"testing,omitempty"`
	CICD      CICDPillar      `json:"cicd,omitempty"`
	Telemetry TelemetryPillar `json:"telemetry,omitempty"`
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
