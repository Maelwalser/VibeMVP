package manifest

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

// PermissionDef defines a named permission (e.g. "users:read").
type PermissionDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RoleDef defines an authorization role with its permissions.
type RoleDef struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Inherits    []string `json:"inherits,omitempty"`
}

// PolicyRule maps a role to a resource and the set of allowed actions.
type PolicyRule struct {
	Role     string   `json:"role"`
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

// AuthConfig describes authentication and identity settings.
type AuthConfig struct {
	Strategy     string       `json:"strategy"`
	Provider     string       `json:"provider"`
	AuthzModel   string       `json:"authz_model"`
	TokenStorage string       `json:"token_storage"`
	MFA          string       `json:"mfa"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	Permissions  []PermissionDef `json:"permissions,omitempty"`
	Roles        []RoleDef       `json:"roles,omitempty"`
	PolicyRules  []PolicyRule    `json:"policy_rules,omitempty"`
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
	Name          string       `json:"name"`
	Technology    string       `json:"technology"`
	Concurrency   string       `json:"concurrency,omitempty"`
	MaxRetries    string       `json:"max_retries,omitempty"`
	RetryPolicy   string       `json:"retry_policy"`
	DLQ           string       `json:"dlq,omitempty"`
	WorkerService string       `json:"worker_service,omitempty"`
	PayloadDTO    string       `json:"payload_dto,omitempty"`
	CronJobs      []CronJobDef `json:"cron_jobs,omitempty"`
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
	ArchPattern   ArchPattern       `json:"arch_pattern"`
	Env           EnvConfig         `json:"env"`
	Services      []ServiceDef      `json:"services,omitempty"`
	CommLinks     []CommLink        `json:"comm_links,omitempty"`
	Messaging     *MessagingConfig  `json:"messaging,omitempty"`
	APIGateway    *APIGatewayConfig `json:"api_gateway,omitempty"`
	Auth          AuthConfig        `json:"auth"`
	JobQueues     []JobQueueDef     `json:"job_queues,omitempty"`
	WAF           WAFConfig         `json:"waf,omitempty"`
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
