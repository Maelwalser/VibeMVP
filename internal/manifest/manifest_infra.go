package manifest

// ── Infrastructure tab types ──────────────────────────────────────────────────

// NetworkingConfig describes networking and connectivity settings.
type NetworkingConfig struct {
	DNSProvider     string `json:"dns_provider"`
	TLSSSL          string `json:"tls_ssl"`
	ReverseProxy    string `json:"reverse_proxy"`
	CDN             string `json:"cdn"`
	PrimaryDomain   string `json:"primary_domain,omitempty"`
	DomainStrategy  string `json:"domain_strategy,omitempty"`
	CORSEnforcement string `json:"cors_enforcement,omitempty"`
	CORSStrategy    string `json:"cors_strategy,omitempty"`
	CORSOrigins     string `json:"cors_origins,omitempty"`
	SSLCertMgmt     string `json:"ssl_cert_mgmt,omitempty"`
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

// ServerEnvironmentDef describes one named deployment environment (e.g. dev, staging, prod).
type ServerEnvironmentDef struct {
	Name          string `json:"name"`
	ComputeEnv    string `json:"compute_env"`
	CloudProvider string `json:"cloud_provider"`
	Orchestrator  string `json:"orchestrator"`
	Regions       string `json:"regions,omitempty"`
}

// InfraPillar groups infrastructure configuration.
type InfraPillar struct {
	Networking    NetworkingConfig       `json:"networking"`
	CICD          CICDConfig             `json:"cicd"`
	Observability ObservabilityConfig    `json:"observability"`
	Environments  []ServerEnvironmentDef `json:"environments,omitempty"`
}
