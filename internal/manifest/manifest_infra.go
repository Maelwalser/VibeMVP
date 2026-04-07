package manifest

// ── Infrastructure tab types ──────────────────────────────────────────────────

// NetworkingConfig describes networking and connectivity settings.
type NetworkingConfig struct {
	DNSProvider     string `json:"dns_provider,omitempty"`
	TLSSSL          string `json:"tls_ssl,omitempty"`
	ReverseProxy    string `json:"reverse_proxy,omitempty"`
	CDN             string `json:"cdn,omitempty"`
	PrimaryDomain   string `json:"primary_domain,omitempty"`
	DomainStrategy  string `json:"domain_strategy,omitempty"`
	CORSEnforcement string `json:"cors_enforcement,omitempty"`
	CORSStrategy    string `json:"cors_strategy,omitempty"`
	CORSOrigins     string `json:"cors_origins,omitempty"`
	SSLCertMgmt     string `json:"ssl_cert_mgmt,omitempty"`
}

// CICDConfig describes CI/CD pipeline settings.
type CICDConfig struct {
	Platform          string `json:"platform,omitempty"`
	ContainerRegistry string `json:"container_registry,omitempty"`
	DeployStrategy    string `json:"deploy_strategy,omitempty"`
	IaCTool           string `json:"iac_tool,omitempty"`
	SecretsMgmt       string `json:"secrets_mgmt,omitempty"`
	ContainerRuntime  string `json:"container_runtime,omitempty"`
	BackupDR          string `json:"backup_dr,omitempty"`
}

// ObservabilityConfig describes logging, metrics, tracing, and alerting settings.
type ObservabilityConfig struct {
	Logging       string `json:"logging,omitempty"`
	Metrics       string `json:"metrics,omitempty"`
	Tracing       string `json:"tracing,omitempty"`
	ErrorTracking string `json:"error_tracking,omitempty"`
	HealthChecks  bool   `json:"health_checks,omitempty"`
	Alerting      string `json:"alerting,omitempty"`
	LogRetention  string `json:"log_retention,omitempty"`
}

// ServerEnvironmentDef describes one named deployment environment (e.g. dev, staging, prod).
type ServerEnvironmentDef struct {
	Name          string `json:"name"`
	ComputeEnv    string `json:"compute_env,omitempty"`
	CloudProvider string `json:"cloud_provider,omitempty"`
	Orchestrator  string `json:"orchestrator,omitempty"`
	Regions       string `json:"regions,omitempty"`
}

// InfraPillar groups infrastructure configuration.
type InfraPillar struct {
	Networking    *NetworkingConfig      `json:"networking,omitempty"`
	CICD          *CICDConfig            `json:"cicd,omitempty"`
	Observability *ObservabilityConfig   `json:"observability,omitempty"`
	Environments  []ServerEnvironmentDef `json:"environments,omitempty"`
}
