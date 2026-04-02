package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// ── modes ─────────────────────────────────────────────────────────────────────

type beMode int

const (
	beNormal beMode = iota
	beInsert
)

// ── arch options ──────────────────────────────────────────────────────────────

type archOption struct {
	value string
	label string
	desc  string
}

var beArchOptions = []archOption{
	{"monolith", "Monolith", "Single deployable unit — all features in one codebase"},
	{"modular-monolith", "Modular Monolith", "Clear domain boundaries, single deployment"},
	{"microservices", "Microservices", "Independent services communicating over a network"},
	{"event-driven", "Event-Driven", "Services communicate asynchronously via events"},
	{"hybrid", "Hybrid", "Mix of patterns — each service unit tagged with its own pattern"},
}

// ── sub-tab IDs per arch ──────────────────────────────────────────────────────

// backendSubTab enumerates the logical sub-tabs in the backend section.
type backendSubTab int

const (
	beTabEnv backendSubTab = iota
	beTabServices
	beTabComm
	beTabMessaging
	beTabAPIGW
	beTabJobs
	beTabSecurity
	beTabAuth
)

// subTabsForArch returns the ordered list of sub-tabs for the given arch value.
func subTabsForArch(arch string) []backendSubTab {
	switch arch {
	case "monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	case "modular-monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabJobs, beTabSecurity, beTabAuth}
	case "microservices":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	case "event-driven":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabJobs, beTabSecurity, beTabAuth}
	case "hybrid":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	default:
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	}
}

func subTabLabel(t backendSubTab) string {
	switch t {
	case beTabEnv:
		return "ENV"
	case beTabServices:
		return "SERVICES"
	case beTabComm:
		return "COMM"
	case beTabMessaging:
		return "MESSAGING"
	case beTabAPIGW:
		return "API GW"
	case beTabJobs:
		return "JOBS"
	case beTabSecurity:
		return "SECURITY"
	case beTabAuth:
		return "AUTH"
	}
	return "?"
}

// ── framework options per language ────────────────────────────────────────────

var backendFrameworksByLang = map[string][]string{
	"Go":              {"Fiber", "Gin", "Echo", "Chi", "net/http (stdlib)", "Connect"},
	"TypeScript/Node": {"Express", "Fastify", "NestJS", "Hono", "tRPC", "Elysia (Bun)"},
	"Python":          {"FastAPI", "Django", "Flask", "Litestar", "Starlette"},
	"Java":            {"Spring Boot", "Quarkus", "Micronaut", "Jakarta EE"},
	"Kotlin":          {"Ktor", "Spring Boot (Kotlin)", "http4k"},
	"C#/.NET":         {"ASP.NET Core", "Minimal APIs", "Carter"},
	"Rust":            {"Axum", "Actix-web", "Rocket", "Warp"},
	"Ruby":            {"Rails", "Sinatra", "Hanami", "Roda"},
	"PHP":             {"Laravel", "Symfony", "Slim", "Laminas"},
	"Elixir":          {"Phoenix", "Plug", "Bandit"},
	"Other":           {"Other"},
}

var backendLanguages = []string{
	"Go", "TypeScript/Node", "Python", "Java", "Kotlin",
	"C#/.NET", "Rust", "Ruby", "PHP", "Elixir", "Other",
}

// ── field definitions ─────────────────────────────────────────────────────────

func defaultEnvFields() []Field {
	return []Field{
		{
			Key: "compute_env", Label: "compute_env   ", Kind: KindSelect,
			Options: []string{
				"Bare Metal", "VM", "Containers (Docker)", "Kubernetes",
				"Serverless (FaaS)", "PaaS",
			},
			Value: "Containers (Docker)", SelIdx: 2,
		},
		{
			Key: "cloud_provider", Label: "cloud_provider", Kind: KindSelect,
			Options: []string{
				"AWS", "GCP", "Azure", "Cloudflare", "Hetzner",
				"Self-hosted", "Other (specify)",
			},
			Value: "AWS",
		},
		{
			Key: "orchestrator", Label: "orchestrator  ", Kind: KindSelect,
			Options: []string{
				"Docker Compose", "K3s", "K8s (managed)", "Nomad",
				"ECS", "Cloud Run", "None",
			},
			Value: "Docker Compose",
		},
		{
			Key: "regions", Label: "regions       ", Kind: KindMultiSelect,
			Options: []string{
				"us-east-1", "us-east-2", "us-west-1", "us-west-2",
				"eu-west-1", "eu-west-2", "eu-central-1",
				"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
				"sa-east-1", "ca-central-1", "af-south-1",
			},
		},
		{
			Key: "stages", Label: "stages        ", Kind: KindSelect,
			Options: []string{
				"Development", "Development + Staging", "Development + Staging + Production",
				"Staging + Production", "Production only",
			},
			Value: "Development + Staging + Production", SelIdx: 2,
		},
		// Monolith-only: language and framework defined once at top level
		{
			Key: "monolith_lang", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "monolith_fw", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "cors_strategy", Label: "CORS Strategy ", Kind: KindSelect,
			Options: []string{"Permissive", "Strict allowlist", "Same-origin"},
			Value:   "Permissive",
		},
		{Key: "cors_origins", Label: "CORS Origins  ", Kind: KindText},
		{
			Key: "session_mgmt", Label: "Session Mgmt  ", Kind: KindSelect,
			Options: []string{"Stateless (JWT only)", "Server-side sessions (Redis)", "Database sessions", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "be_linter", Label: "Linter        ", Kind: KindSelect,
			Options: []string{"golangci-lint", "Ruff", "ESLint", "Checkstyle", "ktlint", "Clippy", "RuboCop", "PHP-CS-Fixer", "Custom", "None"},
			Value:   "None", SelIdx: 9,
		},
	}
}

func defaultServiceFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "responsibility", Label: "responsibility", Kind: KindText},
		{
			Key: "language", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "framework", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "technologies", Label: "technologies  ", Kind: KindMultiSelect,
			Options: []string{"WebSocket", "gRPC", "REST", "GraphQL", "SSE", "tRPC", "MQTT", "Kafka consumer"},
		},
		{
			Key: "pattern_tag", Label: "pattern_tag   ", Kind: KindSelect,
			Options: []string{
				"Monolith part", "Modular module", "Microservice",
				"Event processor", "Serverless function",
			},
			Value: "Microservice",
		},
		{Key: "healthcheck_path", Label: "Healthcheck   ", Kind: KindText, Value: "/healthz"},
		{
			Key: "error_format", Label: "Error Format  ", Kind: KindSelect,
			Options: []string{"RFC 7807 (Problem Details)", "Custom JSON envelope", "Platform default"},
			Value:   "Platform default", SelIdx: 2,
		},
		{
			Key: "service_discovery", Label: "Svc Discovery ", Kind: KindSelect,
			Options: []string{"DNS-based", "Consul", "Kubernetes DNS", "Eureka", "Static config", "None"},
			Value:   "None", SelIdx: 5,
		},
	}
}

func serviceFieldsFromDef(s manifest.ServiceDef) []Field {
	f := defaultServiceFields()
	f = setFieldValue(f, "name", s.Name)
	f = setFieldValue(f, "responsibility", s.Responsibility)
	if s.Language != "" {
		f = setFieldValue(f, "language", s.Language)
		// update framework options based on language
		if opts, ok := backendFrameworksByLang[s.Language]; ok {
			for i := range f {
				if f[i].Key == "framework" {
					f[i].Options = opts
					f[i].SelIdx = 0
					f[i].Value = opts[0]
					if s.Framework != "" {
						f = setFieldValue(f, "framework", s.Framework)
					}
					break
				}
			}
		}
	}
	if s.PatternTag != "" {
		f = setFieldValue(f, "pattern_tag", s.PatternTag)
	}
	return f
}

func serviceDefFromFields(fields []Field) manifest.ServiceDef {
	// Read technologies multiselect
	var techs []string
	for _, f := range fields {
		if f.Key == "technologies" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					techs = append(techs, f.Options[idx])
				}
			}
			break
		}
	}
	return manifest.ServiceDef{
		Name:             fieldGet(fields, "name"),
		Responsibility:   fieldGet(fields, "responsibility"),
		Language:         fieldGet(fields, "language"),
		Framework:        fieldGet(fields, "framework"),
		PatternTag:       fieldGet(fields, "pattern_tag"),
		Technologies:     techs,
		HealthcheckPath:  fieldGet(fields, "healthcheck_path"),
		ErrorFormat:      fieldGet(fields, "error_format"),
		ServiceDiscovery: fieldGet(fields, "service_discovery"),
	}
}

func defaultCommFields() []Field {
	return []Field{
		{Key: "from", Label: "from          ", Kind: KindText},
		{Key: "to", Label: "to            ", Kind: KindText},
		{
			Key: "direction", Label: "direction     ", Kind: KindSelect,
			Options: []string{
				"Unidirectional (→)", "Bidirectional (↔)", "Pub/Sub (fan-out)",
			},
			Value: "Unidirectional (→)",
		},
		{
			Key: "protocol", Label: "protocol      ", Kind: KindSelect,
			Options: []string{
				"REST (HTTP)", "gRPC", "GraphQL", "WebSocket",
				"Message Queue", "Event Bus", "Internal (in-process)",
			},
			Value: "REST (HTTP)",
		},
		{Key: "trigger", Label: "trigger       ", Kind: KindText},
		{
			Key: "sync_async", Label: "sync_async    ", Kind: KindSelect,
			Options: []string{"Synchronous", "Asynchronous", "Fire-and-forget"},
			Value:   "Synchronous",
		},
		{
			Key: "resilience", Label: "resilience    ", Kind: KindMultiSelect,
			Options: []string{"Circuit breaker", "Retry with backoff", "Timeout", "Bulkhead", "None"},
		},
	}
}

func commLinkFromFields(fields []Field) manifest.CommLink {
	// Read resilience multiselect
	var resilience []string
	for _, f := range fields {
		if f.Key == "resilience" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					resilience = append(resilience, f.Options[idx])
				}
			}
			break
		}
	}
	return manifest.CommLink{
		From:               fieldGet(fields, "from"),
		To:                 fieldGet(fields, "to"),
		Direction:          fieldGet(fields, "direction"),
		Protocol:           fieldGet(fields, "protocol"),
		Trigger:            fieldGet(fields, "trigger"),
		SyncAsync:          fieldGet(fields, "sync_async"),
		ResiliencePatterns: resilience,
	}
}

func commFieldsFromLink(l manifest.CommLink) []Field {
	f := defaultCommFields()
	f = setFieldValue(f, "from", l.From)
	f = setFieldValue(f, "to", l.To)
	if l.Direction != "" {
		f = setFieldValue(f, "direction", l.Direction)
	}
	if l.Protocol != "" {
		f = setFieldValue(f, "protocol", l.Protocol)
	}
	f = setFieldValue(f, "trigger", l.Trigger)
	if l.SyncAsync != "" {
		f = setFieldValue(f, "sync_async", l.SyncAsync)
	}
	return f
}

func defaultMessagingFields() []Field {
	return []Field{
		{
			Key: "broker_tech", Label: "broker_tech   ", Kind: KindSelect,
			Options: []string{
				"Kafka", "NATS", "RabbitMQ", "Redis Streams",
				"AWS SQS/SNS", "Google Pub/Sub", "Azure Service Bus", "Pulsar",
			},
			Value: "Kafka",
		},
		{
			Key: "deployment", Label: "deployment    ", Kind: KindSelect,
			Options: []string{"Managed (cloud)", "Self-hosted", "Embedded"},
			Value:   "Managed (cloud)",
		},
		{
			Key: "serialization", Label: "serialization ", Kind: KindSelect,
			Options: []string{"JSON", "Protobuf", "Avro", "MessagePack", "CloudEvents"},
			Value:   "JSON",
		},
		{
			Key: "delivery", Label: "delivery      ", Kind: KindSelect,
			Options: []string{"At-most-once", "At-least-once", "Exactly-once"},
			Value:   "At-least-once",
		},
	}
}

func defaultEventFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "domain", Label: "domain        ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

func defaultAPIGWFields() []Field {
	return []Field{
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: []string{
				"Kong", "Traefik", "NGINX", "Envoy",
				"AWS API Gateway", "Cloudflare Workers", "Custom (specify)", "None",
			},
			Value: "Kong",
		},
		{
			Key: "routing", Label: "routing       ", Kind: KindSelect,
			Options: []string{"Path-based", "Header-based", "Domain-based"},
			Value:   "Path-based",
		},
		{
			Key: "features", Label: "features      ", Kind: KindMultiSelect,
			Options: []string{
				"Rate limiting", "JWT validation", "SSL termination",
				"Load balancing", "Request caching", "Logging & tracing",
				"Request transformation", "CORS handling",
				"IP allowlist/blocklist", "Circuit breaking", "Health checks",
			},
		},
	}
}

func defaultAuthFields() []Field {
	return []Field{
		{
			Key: "strategy", Label: "strategy      ", Kind: KindMultiSelect,
			Options: []string{
				"JWT (stateless)", "Session-based", "OAuth 2.0 / OIDC",
				"API Keys", "mTLS", "None",
			},
		},
		{
			Key: "provider", Label: "provider      ", Kind: KindSelect,
			Options: []string{
				"Self-managed", "Auth0", "Clerk", "Supabase Auth",
				"Firebase Auth", "Keycloak", "AWS Cognito", "Other",
			},
			Value: "Self-managed",
		},
		{
			Key: "authz_model", Label: "authz_model   ", Kind: KindSelect,
			Options: []string{"RBAC", "ABAC", "ACL", "ReBAC", "Policy-based (OPA/Cedar)", "Custom"},
			Value:   "RBAC",
		},
		{
			Key: "roles", Label: "roles         ", Kind: KindMultiSelect,
			Options: []string{
				"admin", "superadmin", "user", "moderator",
				"editor", "viewer", "manager", "auditor", "owner",
			},
		},
		{
			Key: "token_storage", Label: "token_storage ", Kind: KindMultiSelect,
			Options: []string{
				"HttpOnly cookie", "Authorization header (Bearer)",
				"WebSocket protocol header", "Other",
			},
		},
		{
			Key: "refresh_token", Label: "refresh_token ", Kind: KindSelect,
			Options: []string{"None", "Rotating", "Non-rotating", "Sliding window"},
			Value:   "None",
		},
		{
			Key: "mfa", Label: "mfa           ", Kind: KindSelect,
			Options: []string{"None", "TOTP", "SMS", "Email", "Passkeys/WebAuthn"},
			Value:   "None",
		},
	}
}

func defaultSecurityFields() []Field {
	return []Field{
		{
			Key: "waf_provider", Label: "waf_provider  ", Kind: KindSelect,
			Options: []string{"Cloudflare WAF", "AWS WAF", "ModSecurity", "NGINX ModSec", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "waf_ruleset", Label: "waf_ruleset   ", Kind: KindSelect,
			Options: []string{"OWASP Core Rule Set", "Managed rules", "Custom", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "captcha", Label: "captcha       ", Kind: KindSelect,
			Options: []string{"hCaptcha", "reCAPTCHA v2", "reCAPTCHA v3", "Cloudflare Turnstile", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "bot_protection", Label: "bot_protection", Kind: KindSelect,
			Options: []string{"Cloudflare Bot Management", "Imperva", "DataDome", "Custom", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "rate_limit_strategy", Label: "rate_limit    ", Kind: KindSelect,
			Options: []string{"Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "rate_limit_backend", Label: "rl_backend    ", Kind: KindSelect,
			Options: []string{"Redis", "Memcached", "In-memory", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "ddos_protection", Label: "ddos_protect  ", Kind: KindSelect,
			Options: []string{"CDN-level (Cloudflare)", "Provider-managed", "None"},
			Value:   "None", SelIdx: 2,
		},
	}
}

func defaultJobQueueFormFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: []string{"Temporal", "BullMQ", "Sidekiq", "Celery", "Faktory", "Asynq", "River", "Custom"},
			Value:   "BullMQ", SelIdx: 1,
		},
		{Key: "concurrency", Label: "concurrency   ", Kind: KindText, Value: "10"},
		{Key: "max_retries", Label: "max_retries   ", Kind: KindText, Value: "3"},
		{
			Key: "retry_policy", Label: "retry_policy  ", Kind: KindSelect,
			Options: []string{"Exponential backoff", "Fixed interval", "Linear backoff", "None"},
			Value:   "Exponential backoff",
		},
		{
			Key: "dlq", Label: "dead_letter_q ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
	}
}

// ── beSubView type ────────────────────────────────────────────────────────────

type beSubView int

const (
	beViewList beSubView = iota
	beViewForm
)

// ── list + form sub-editor for services and comm links ───────────────────────

type beListView int

const (
	beListViewList beListView = iota
	beListViewForm
)

type beListEditor struct {
	items    [][]Field        // each item is a slice of fields
	itemView beListView
	itemIdx  int
	formIdx  int
	form     []Field
}

func newBeListEditor() beListEditor {
	return beListEditor{itemView: beListViewList}
}

// ── BackendEditor ─────────────────────────────────────────────────────────────

// BackendEditor manages the BACKEND section.
type BackendEditor struct {
	// Arch selection
	ArchIdx       int
	ArchConfirmed bool
	dropdownOpen  bool
	dropdownIdx   int

	// Sub-tab state
	activeTabIdx int // index into subTabsForArch(arch)
	activeField  int

	// Field stores
	EnvFields       []Field
	envEnabled      bool
	MessagingFields []Field
	APIGWFields     []Field
	apiGWEnabled    bool
	AuthFields      []Field
	authEnabled     bool

	// Security/WAF tab
	securityFields []Field
	secFormIdx     int
	secEnabled     bool

	// Jobs tab
	jobQueues   []manifest.JobQueueDef
	jobsSubView beSubView
	jobsIdx     int
	jobsForm    []Field
	jobsFormIdx int

	// List editors
	serviceEditor beListEditor
	commEditor    beListEditor
	eventEditor   beListEditor // event catalog within messaging

	// Internal mode
	internalMode beMode
	formInput    textinput.Model
	width        int

	// Dropdown state (shared across all sub-contexts; only one can be open)
	ddOpen   bool
	ddOptIdx int

	// Services and comm links for manifest export
	Services  []manifest.ServiceDef
	CommLinks []manifest.CommLink
	Events    []manifest.EventDef

	// Cross-tab references (injected from model.go)
	DomainNames []string

	// Vim motion state
	countBuf string
	gBuf     bool
}

func newBackendEditor() BackendEditor {
	return BackendEditor{
		EnvFields:       defaultEnvFields(),
		MessagingFields: defaultMessagingFields(),
		APIGWFields:     defaultAPIGWFields(),
		AuthFields:      defaultAuthFields(),
		securityFields:  defaultSecurityFields(),
		serviceEditor:   newBeListEditor(),
		commEditor:      newBeListEditor(),
		eventEditor:     newBeListEditor(),
		formInput:       newFormInput(),
		dropdownOpen:    true,
	}
}

// SetDomainNames injects domain names from the data tab for event domain dropdowns.
func (be *BackendEditor) SetDomainNames(names []string) {
	be.DomainNames = names
}

// withServiceNames returns a copy of fields where from/to are upgraded to
// KindSelect dropdowns populated with the current service names.
func (be BackendEditor) withServiceNames(fields []Field) []Field {
	names := be.ServiceNames()
	if len(names) == 0 {
		return fields
	}
	out := copyFields(fields)
	for i := range out {
		if out[i].Key != "from" && out[i].Key != "to" {
			continue
		}
		out[i].Kind = KindSelect
		out[i].Options = names
		// Try to preserve the existing value by finding it in the options.
		// If not found, keep Value as-is but point SelIdx to first option.
		found := false
		for j, n := range names {
			if n == out[i].Value {
				out[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			out[i].SelIdx = 0
			if out[i].Value == "" {
				out[i].Value = names[0]
			}
		}
	}
	return out
}

// withDomainNames returns a copy of fields where the domain field is upgraded to
// a KindSelect dropdown populated with the available domain names.
func (be BackendEditor) withDomainNames(fields []Field) []Field {
	if len(be.DomainNames) == 0 {
		return fields
	}
	out := copyFields(fields)
	for i := range out {
		if out[i].Key == "domain" {
			out[i].Kind = KindSelect
			out[i].Options = be.DomainNames
			found := false
			for _, n := range be.DomainNames {
				if n == out[i].Value {
					found = true
					break
				}
			}
			if !found {
				out[i].Value = be.DomainNames[0]
				out[i].SelIdx = 0
			} else {
				for j, n := range be.DomainNames {
					if n == out[i].Value {
						out[i].SelIdx = j
						break
					}
				}
			}
		}
	}
	return out
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (be BackendEditor) currentArch() string {
	if be.ArchIdx >= 0 && be.ArchIdx < len(beArchOptions) {
		return beArchOptions[be.ArchIdx].value
	}
	return beArchOptions[0].value
}

func (be BackendEditor) activeTabs() []backendSubTab {
	return subTabsForArch(be.currentArch())
}

func (be BackendEditor) activeTab() backendSubTab {
	tabs := be.activeTabs()
	if be.activeTabIdx >= 0 && be.activeTabIdx < len(tabs) {
		return tabs[be.activeTabIdx]
	}
	return beTabEnv
}

func (be BackendEditor) tabLabels() []string {
	tabs := be.activeTabs()
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = subTabLabel(t)
	}
	return labels
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (be BackendEditor) ToManifest() manifest.BackendPillar {
	arch := be.currentArch()

	var env manifest.EnvConfig
	if be.envEnabled {
		env = manifest.EnvConfig{
			ComputeEnv:    fieldGet(be.EnvFields, "compute_env"),
			CloudProvider: fieldGet(be.EnvFields, "cloud_provider"),
			Orchestrator:  fieldGet(be.EnvFields, "orchestrator"),
			Regions:       fieldGetMulti(be.EnvFields, "regions"),
			Stages:        fieldGet(be.EnvFields, "stages"),
		}
	}

	var auth manifest.AuthConfig
	if be.authEnabled {
		auth = manifest.AuthConfig{
			Strategy:     fieldGetMulti(be.AuthFields, "strategy"),
			Provider:     fieldGet(be.AuthFields, "provider"),
			AuthzModel:   fieldGet(be.AuthFields, "authz_model"),
			Roles:        fieldGetMulti(be.AuthFields, "roles"),
			TokenStorage: fieldGetMulti(be.AuthFields, "token_storage"),
			RefreshToken: fieldGet(be.AuthFields, "refresh_token"),
			MFA:          fieldGet(be.AuthFields, "mfa"),
		}
	}

	bp := manifest.BackendPillar{
		ArchPattern: manifest.ArchPattern(arch),
		Env:         env,
		Services:    be.Services,
		CommLinks:   be.CommLinks,
		Auth:        auth,
		JobQueues:   be.jobQueues,
	}
	if be.secEnabled {
		bp.WAF = manifest.WAFConfig{
			Provider:          fieldGet(be.securityFields, "waf_provider"),
			Ruleset:           fieldGet(be.securityFields, "waf_ruleset"),
			CAPTCHA:           fieldGet(be.securityFields, "captcha"),
			BotProtection:     fieldGet(be.securityFields, "bot_protection"),
			RateLimitStrategy: fieldGet(be.securityFields, "rate_limit_strategy"),
			RateLimitBackend:  fieldGet(be.securityFields, "rate_limit_backend"),
			DDoSProtection:    fieldGet(be.securityFields, "ddos_protection"),
		}
	}
	if be.envEnabled {
		bp.CORSStrategy = fieldGet(be.EnvFields, "cors_strategy")
		bp.CORSOrigins = fieldGet(be.EnvFields, "cors_origins")
		bp.SessionMgmt = fieldGet(be.EnvFields, "session_mgmt")
		bp.BackendLinter = fieldGet(be.EnvFields, "be_linter")
	}

	tabs := subTabsForArch(arch)
	for _, t := range tabs {
		if t == beTabMessaging {
			mc := manifest.MessagingConfig{
				BrokerTech:    fieldGet(be.MessagingFields, "broker_tech"),
				Deployment:    fieldGet(be.MessagingFields, "deployment"),
				Serialization: fieldGet(be.MessagingFields, "serialization"),
				Delivery:      fieldGet(be.MessagingFields, "delivery"),
			}
			bp.Messaging = &mc
		}
		if t == beTabAPIGW && be.apiGWEnabled {
			gw := manifest.APIGatewayConfig{
				Technology: fieldGet(be.APIGWFields, "technology"),
				Routing:    fieldGet(be.APIGWFields, "routing"),
				Features:   fieldGet(be.APIGWFields, "features"),
			}
			bp.APIGateway = &gw
		}
	}

	// Legacy compat fields
	bp.ComputeEnv = manifest.ComputeEnv(env.ComputeEnv)
	bp.CloudProvider = env.CloudProvider
	if arch == "monolith" {
		bp.Language = fieldGet(be.EnvFields, "monolith_lang")
		bp.Framework = fieldGet(be.EnvFields, "monolith_fw")
	} else if len(be.Services) > 0 {
		bp.Language = be.Services[0].Language
		bp.Framework = be.Services[0].Framework
	}
	return bp
}

// FromBackendPillar populates the editor from a saved manifest BackendPillar,
// reversing the ToManifest() operation.
func (be BackendEditor) FromBackendPillar(bp manifest.BackendPillar) BackendEditor {
	// Restore arch selection.
	arch := string(bp.ArchPattern)
	for i, opt := range beArchOptions {
		if opt.value == arch {
			be.ArchIdx = i
			be.dropdownIdx = i
			break
		}
	}
	if arch != "" {
		be.ArchConfirmed = true
		be.dropdownOpen = false
	}

	// Env fields.
	if bp.Env.ComputeEnv != "" || bp.Env.CloudProvider != "" || bp.CORSStrategy != "" || bp.BackendLinter != "" {
		be.envEnabled = true
		be.EnvFields = setFieldValue(be.EnvFields, "compute_env", bp.Env.ComputeEnv)
		be.EnvFields = setFieldValue(be.EnvFields, "cloud_provider", bp.Env.CloudProvider)
		be.EnvFields = setFieldValue(be.EnvFields, "orchestrator", bp.Env.Orchestrator)
		be.EnvFields = restoreMultiSelectValue(be.EnvFields, "regions", bp.Env.Regions)
		be.EnvFields = setFieldValue(be.EnvFields, "stages", bp.Env.Stages)
		be.EnvFields = setFieldValue(be.EnvFields, "cors_strategy", bp.CORSStrategy)
		be.EnvFields = setFieldValue(be.EnvFields, "cors_origins", bp.CORSOrigins)
		be.EnvFields = setFieldValue(be.EnvFields, "session_mgmt", bp.SessionMgmt)
		be.EnvFields = setFieldValue(be.EnvFields, "be_linter", bp.BackendLinter)
		if arch == "monolith" {
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_lang", bp.Language)
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_fw", bp.Framework)
		}
	}

	// Auth fields.
	if bp.Auth.Strategy != "" || bp.Auth.Provider != "" {
		be.authEnabled = true
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "strategy", bp.Auth.Strategy)
		be.AuthFields = setFieldValue(be.AuthFields, "provider", bp.Auth.Provider)
		be.AuthFields = setFieldValue(be.AuthFields, "authz_model", bp.Auth.AuthzModel)
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "roles", bp.Auth.Roles)
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "token_storage", bp.Auth.TokenStorage)
		be.AuthFields = setFieldValue(be.AuthFields, "refresh_token", bp.Auth.RefreshToken)
		be.AuthFields = setFieldValue(be.AuthFields, "mfa", bp.Auth.MFA)
	}

	// Security / WAF fields.
	if bp.WAF.Provider != "" || bp.WAF.Ruleset != "" {
		be.secEnabled = true
		be.securityFields = setFieldValue(be.securityFields, "waf_provider", bp.WAF.Provider)
		be.securityFields = setFieldValue(be.securityFields, "waf_ruleset", bp.WAF.Ruleset)
		be.securityFields = setFieldValue(be.securityFields, "captcha", bp.WAF.CAPTCHA)
		be.securityFields = setFieldValue(be.securityFields, "bot_protection", bp.WAF.BotProtection)
		be.securityFields = setFieldValue(be.securityFields, "rate_limit_strategy", bp.WAF.RateLimitStrategy)
		be.securityFields = setFieldValue(be.securityFields, "rate_limit_backend", bp.WAF.RateLimitBackend)
		be.securityFields = setFieldValue(be.securityFields, "ddos_protection", bp.WAF.DDoSProtection)
	}

	// Messaging fields.
	if bp.Messaging != nil {
		be.MessagingFields = setFieldValue(be.MessagingFields, "broker_tech", bp.Messaging.BrokerTech)
		be.MessagingFields = setFieldValue(be.MessagingFields, "deployment", bp.Messaging.Deployment)
		be.MessagingFields = setFieldValue(be.MessagingFields, "serialization", bp.Messaging.Serialization)
		be.MessagingFields = setFieldValue(be.MessagingFields, "delivery", bp.Messaging.Delivery)
	}

	// API Gateway fields.
	if bp.APIGateway != nil {
		be.apiGWEnabled = true
		be.APIGWFields = setFieldValue(be.APIGWFields, "technology", bp.APIGateway.Technology)
		be.APIGWFields = setFieldValue(be.APIGWFields, "routing", bp.APIGateway.Routing)
		be.APIGWFields = setFieldValue(be.APIGWFields, "features", bp.APIGateway.Features)
	}

	// Collections — stored directly; per-item forms are rebuilt lazily on navigation.
	be.Services = bp.Services
	be.serviceEditor.items = make([][]Field, len(bp.Services))
	for i, svc := range bp.Services {
		be.serviceEditor.items[i] = serviceFieldsFromDef(svc)
	}

	be.CommLinks = bp.CommLinks
	be.commEditor.items = make([][]Field, len(bp.CommLinks))
	for i, link := range bp.CommLinks {
		be.commEditor.items[i] = commFieldsFromLink(link)
	}

	be.jobQueues = bp.JobQueues

	return be
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (be BackendEditor) Mode() Mode {
	if be.internalMode == beInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (be BackendEditor) HintLine() string {
	if !be.ArchConfirmed {
		if be.dropdownOpen {
			return hintBar("j/k", "navigate", "Enter/Space", "confirm", "Esc", "close")
		}
		return hintBar("Enter/Space", "open arch selector")
	}
	if be.ddOpen {
		return hintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
	}
	if be.internalMode == beInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}

	tab := be.activeTab()
	switch tab {
	case beTabServices:
		ed := be.serviceEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add service", "d", "delete", "Enter", "edit", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back", "Tab", "next field")
	case beTabComm:
		ed := be.commEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add link", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "Space", "cycle", "a", "add event", "d", "del event", "h/l", "sub-tab")
	case beTabJobs:
		if be.jobsSubView == beViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "a", "add job queue", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
	case beTabSecurity:
		if !be.secEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "gg/G", "top/bottom", "a/Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab")
	default:
		t := be.activeTab()
		configEnabled := true
		switch t {
		case beTabEnv:
			configEnabled = be.envEnabled
		case beTabAPIGW:
			configEnabled = be.apiGWEnabled
		case beTabAuth:
			configEnabled = be.authEnabled
		}
		if !configEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab", "b", "change arch")
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (be BackendEditor) Update(msg tea.Msg) (BackendEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		be.width = wsz.Width
		be.formInput.Width = wsz.Width - 22
		return be, nil
	}
	if !be.ArchConfirmed {
		return be.updateArchSelect(msg)
	}
	if be.internalMode == beInsert {
		return be.updateInsert(msg)
	}
	if be.ddOpen {
		key, ok := msg.(tea.KeyMsg)
		if ok {
			return be.updateDropdown(key)
		}
		return be, nil
	}
	return be.updateNormal(msg)
}

func (be BackendEditor) updateArchSelect(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}
	if be.dropdownOpen {
		switch key.String() {
		case "j", "down":
			if be.dropdownIdx < len(beArchOptions)-1 {
				be.dropdownIdx++
			}
		case "k", "up":
			if be.dropdownIdx > 0 {
				be.dropdownIdx--
			}
		case "g":
			be.dropdownIdx = 0
		case "G":
			be.dropdownIdx = len(beArchOptions) - 1
		case "enter", " ":
			be.ArchIdx = be.dropdownIdx
			be.dropdownOpen = false
			be.ArchConfirmed = true
			be.activeTabIdx = 0
			be.activeField = 0
		case "esc":
			be.dropdownOpen = false
		}
		return be, nil
	}
	switch key.String() {
	case "enter", " ":
		be.dropdownOpen = true
		be.dropdownIdx = be.ArchIdx
	}
	return be, nil
}

// updateDropdown handles navigation while a dropdown menu is open.
func (be BackendEditor) updateDropdown(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	opts := be.dropdownOptions()
	isMulti := be.isMultiSelectDropdown()
	switch key.String() {
	case "j", "down":
		if be.ddOptIdx < len(opts)-1 {
			be.ddOptIdx++
		}
	case "k", "up":
		if be.ddOptIdx > 0 {
			be.ddOptIdx--
		}
	case "g":
		be.ddOptIdx = 0
	case "G":
		if len(opts) > 0 {
			be.ddOptIdx = len(opts) - 1
		}
	case " ":
		if isMulti {
			// Toggle the current option
			be.toggleMultiSelectOption()
		} else {
			be.applyDropdown()
			be.ddOpen = false
		}
	case "enter":
		if isMulti {
			be.toggleMultiSelectOption()
		} else {
			be.applyDropdown()
			be.ddOpen = false
		}
	case "esc", "ctrl+c":
		if isMulti {
			be.saveMultiSelectCursor()
		}
		be.ddOpen = false
	}
	return be, nil
}

// isMultiSelectDropdown returns true when the active dropdown field is KindMultiSelect.
func (be BackendEditor) isMultiSelectDropdown() bool {
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) {
			return ed.form[ed.formIdx].Kind == KindMultiSelect
		}
	}
	if f := be.mutableFieldPtr(); f != nil {
		return f.Kind == KindMultiSelect
	}
	return false
}

// toggleMultiSelectOption toggles ddOptIdx in the active KindMultiSelect field.
func (be *BackendEditor) toggleMultiSelectOption() {
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].ToggleMultiSelect(be.ddOptIdx)
			ed.form[ed.formIdx].DDCursor = be.ddOptIdx
		}
		return
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindMultiSelect {
		f.ToggleMultiSelect(be.ddOptIdx)
		f.DDCursor = be.ddOptIdx
	}
}

// saveMultiSelectCursor saves the current dropdown cursor back to the field.
func (be *BackendEditor) saveMultiSelectCursor() {
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].DDCursor = be.ddOptIdx
		}
		return
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindMultiSelect {
		f.DDCursor = be.ddOptIdx
	}
}

// dropdownOptions returns the options of the currently active KindSelect or KindMultiSelect field.
func (be BackendEditor) dropdownOptions() []string {
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) && (be.jobsForm[be.jobsFormIdx].Kind == KindSelect || be.jobsForm[be.jobsFormIdx].Kind == KindMultiSelect) {
			return be.jobsForm[be.jobsFormIdx].Options
		}
	}
	if f := be.mutableFieldPtr(); f != nil {
		return f.Options
	}
	return nil
}

// applyDropdown writes ddOptIdx back to the active KindSelect field.
func (be *BackendEditor) applyDropdown() {
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			if f.Kind == KindSelect && be.ddOptIdx < len(f.Options) {
				f.SelIdx = be.ddOptIdx
				f.Value = f.Options[be.ddOptIdx]
				if f.Key == "language" {
					be.updateServiceFrameworkOptions(ed)
				}
			}
			// KindMultiSelect handled via toggleMultiSelectOption
		}
		return
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			if f.Kind == KindSelect && be.ddOptIdx < len(f.Options) {
				f.SelIdx = be.ddOptIdx
				f.Value = f.Options[be.ddOptIdx]
			}
		}
		return
	}
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			if f.Kind == KindSelect && be.ddOptIdx < len(f.Options) {
				f.SelIdx = be.ddOptIdx
				f.Value = f.Options[be.ddOptIdx]
			}
		}
		return
	}
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) {
			f := &be.jobsForm[be.jobsFormIdx]
			if f.Kind == KindSelect && be.ddOptIdx < len(f.Options) {
				f.SelIdx = be.ddOptIdx
				f.Value = f.Options[be.ddOptIdx]
			}
		}
		return
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect && be.ddOptIdx < len(f.Options) {
		f.SelIdx = be.ddOptIdx
		f.Value = f.Options[be.ddOptIdx]
	}
}

func (be BackendEditor) updateInsert(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			be.saveInput()
			be.internalMode = beNormal
			be.formInput.Blur()
			return be, nil
		case "tab":
			be.saveInput()
			if be.jobsSubView == beViewForm {
				n := len(be.jobsForm)
				if n > 0 {
					be.jobsFormIdx = (be.jobsFormIdx + 1) % n
					be.activeField = be.jobsFormIdx
				}
				return be.enterJobsFormInsert()
			}
			fields := be.currentEditableFields()
			if fields != nil {
				be.activeField = (be.activeField + 1) % len(*fields)
				return be.tryEnterInsert()
			}
		case "shift+tab":
			be.saveInput()
			if be.jobsSubView == beViewForm {
				n := len(be.jobsForm)
				if n > 0 {
					be.jobsFormIdx = (be.jobsFormIdx - 1 + n) % n
					be.activeField = be.jobsFormIdx
				}
				return be.enterJobsFormInsert()
			}
			fields := be.currentEditableFields()
			if fields != nil {
				n := len(*fields)
				be.activeField = (be.activeField - 1 + n) % n
				return be.tryEnterInsert()
			}
		}
	}
	var cmd tea.Cmd
	be.formInput, cmd = be.formInput.Update(msg)
	return be, cmd
}

func (be BackendEditor) updateNormal(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}

	tab := be.activeTab()

	// Delegate list editors
	switch tab {
	case beTabServices:
		if be.serviceEditor.itemView == beListViewList {
			return be.updateServiceList(key)
		}
		return be.updateServiceForm(key)
	case beTabComm:
		if be.commEditor.itemView == beListViewList {
			return be.updateCommList(key)
		}
		return be.updateCommForm(key)
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return be.updateEventForm(key)
		}
		return be.updateMessaging(key)
	case beTabJobs:
		if be.jobsSubView == beViewList {
			return be.updateJobsList(key)
		}
		return be.updateJobsForm(key)
	case beTabSecurity:
		return be.updateSecurity(key)
	}

	// Enabled guard for config-only tabs (ENV, API GW, AUTH)
	k := key.String()
	activeConfigEnabled := true
	switch tab {
	case beTabEnv:
		activeConfigEnabled = be.envEnabled
	case beTabAPIGW:
		activeConfigEnabled = be.apiGWEnabled
	case beTabAuth:
		activeConfigEnabled = be.authEnabled
	}
	if !activeConfigEnabled {
		if k == "a" {
			switch tab {
			case beTabEnv:
				be.envEnabled = true
			case beTabAPIGW:
				be.apiGWEnabled = true
			case beTabAuth:
				be.authEnabled = true
			}
			be.activeField = 0
		} else if k == "h" || k == "left" {
			be.countBuf = ""
			be.gBuf = false
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		} else if k == "l" || k == "right" {
			be.countBuf = ""
			be.gBuf = false
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		} else if k == "b" {
			be.countBuf = ""
			be.gBuf = false
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}

	// Vim count prefix (digits 1-9, or 0 when count already started)
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		be.countBuf += k
		be.gBuf = false
		return be, nil
	}
	if k == "0" && be.countBuf != "" {
		be.countBuf += "0"
		be.gBuf = false
		return be, nil
	}

	// Shift+D resets the active single-config tab (ENV, API GW, AUTH).
	if k == "D" {
		switch tab {
		case beTabEnv:
			be.envEnabled = false
			be.EnvFields = defaultEnvFields()
			be.activeField = 0
		case beTabAPIGW:
			be.apiGWEnabled = false
			be.APIGWFields = defaultAPIGWFields()
			be.activeField = 0
		case beTabAuth:
			be.authEnabled = false
			be.AuthFields = defaultAuthFields()
			be.activeField = 0
		}
		return be, nil
	}

	// Generic field navigation for ENV, API GW, AUTH
	switch k {
	case "b":
		be.countBuf = ""
		be.gBuf = false
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "h", "left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	case "j", "down":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		if be.activeTab() == beTabEnv {
			visible := be.visibleEnvFields()
			for i := 0; i < count; i++ {
				if be.activeField < len(visible)-1 {
					be.activeField++
				}
			}
		} else if fields := be.currentEditableFields(); fields != nil {
			for i := 0; i < count; i++ {
				if be.activeField < len(*fields)-1 {
					be.activeField++
				}
			}
		}
	case "k", "up":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField > 0 {
				be.activeField--
			}
		}
	case "g":
		if be.gBuf {
			// gg — go to top
			be.activeField = 0
			be.gBuf = false
		} else {
			be.gBuf = true
		}
		be.countBuf = ""
	case "G":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTab() == beTabEnv {
			visible := be.visibleEnvFields()
			if len(visible) > 0 {
				be.activeField = len(visible) - 1
			}
		} else if fields := be.currentEditableFields(); fields != nil {
			be.activeField = len(*fields) - 1
		}
	case "enter", " ":
		be.countBuf = ""
		be.gBuf = false
		if f := be.mutableFieldPtr(); f != nil && (f.Kind == KindSelect || f.Kind == KindMultiSelect) {
			be.ddOpen = true
			if f.Kind == KindSelect {
				be.ddOptIdx = f.SelIdx
			} else {
				be.ddOptIdx = f.DDCursor
			}
		} else {
			return be.tryEnterInsert()
		}
	case "H", "shift+left":
		be.countBuf = ""
		be.gBuf = false
		if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		be.countBuf = ""
		be.gBuf = false
		return be.tryEnterInsert()
	default:
		be.countBuf = ""
		be.gBuf = false
	}
	return be, nil
}

func (be BackendEditor) updateServiceList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		svc := manifest.ServiceDef{}
		be.Services = append(be.Services, svc)
		ed.items = append(ed.items, defaultServiceFields())
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.Services = append(be.Services[:ed.itemIdx], be.Services[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "b":
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	}
	return be, nil
}

// isServiceFieldHidden returns true when a service form field should be hidden for the current arch.
func (be BackendEditor) isServiceFieldHidden(key string) bool {
	arch := be.currentArch()
	if arch == "monolith" && (key == "language" || key == "framework") {
		return true
	}
	if arch != "hybrid" && key == "pattern_tag" {
		return true
	}
	return false
}

// nextServiceFormIdx advances formIdx skipping hidden fields.
func (be BackendEditor) nextServiceFormIdx(ed *beListEditor, delta int) int {
	n := len(ed.form)
	if n == 0 {
		return 0
	}
	idx := ed.formIdx
	for i := 0; i < n; i++ {
		idx = (idx + delta + n) % n
		if !be.isServiceFieldHidden(ed.form[idx].Key) {
			return idx
		}
	}
	return ed.formIdx
}

func (be BackendEditor) updateServiceForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = be.nextServiceFormIdx(ed, 1)
	case "k", "up":
		ed.formIdx = be.nextServiceFormIdx(ed, -1)
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			be.ddOpen = true
			if f.Kind == KindSelect {
				be.ddOptIdx = f.SelIdx
			} else {
				be.ddOptIdx = f.DDCursor
			}
		} else {
			return be.enterServiceFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
			if f.Key == "language" {
				be.updateServiceFrameworkOptions(ed)
			}
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterServiceFormInsert()
		}
	case "b", "esc":
		be.saveServiceForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be *BackendEditor) updateServiceFrameworkOptions(ed *beListEditor) {
	lang := fieldGet(ed.form, "language")
	opts, ok := backendFrameworksByLang[lang]
	if !ok {
		opts = []string{"Other"}
	}
	for i := range ed.form {
		if ed.form[i].Key == "framework" {
			ed.form[i].Options = opts
			ed.form[i].SelIdx = 0
			ed.form[i].Value = opts[0]
			break
		}
	}
}

func (be BackendEditor) enterServiceFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveServiceForm() {
	ed := &be.serviceEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	svc := serviceDefFromFields(ed.form)
	if ed.itemIdx < len(be.Services) {
		be.Services[ed.itemIdx] = svc
	}
}

func (be BackendEditor) updateCommList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		be.CommLinks = append(be.CommLinks, manifest.CommLink{})
		ed.items = append(ed.items, be.withServiceNames(defaultCommFields()))
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.CommLinks = append(be.CommLinks[:ed.itemIdx], be.CommLinks[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = be.withServiceNames(copyFields(ed.items[ed.itemIdx]))
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "b":
		be.ArchConfirmed = false
	}
	return be, nil
}

func (be BackendEditor) updateCommForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			be.ddOpen = true
			be.ddOptIdx = f.SelIdx
		} else {
			return be.enterCommFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterCommFormInsert()
		}
	case "b", "esc":
		be.saveCommForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterCommFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveCommForm() {
	ed := &be.commEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	link := commLinkFromFields(ed.form)
	if ed.itemIdx < len(be.CommLinks) {
		be.CommLinks[ed.itemIdx] = link
	}
}

func (be BackendEditor) updateMessaging(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	// Upper section: messaging broker config fields
	// Lower section: event catalog list
	// We use a split: first len(MessagingFields) positions are broker fields,
	// then event list items below.
	brokerCount := len(be.MessagingFields)
	eventCount := len(ed.items)
	total := brokerCount + eventCount

	switch key.String() {
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	case "b":
		be.ArchConfirmed = false
	case "j", "down":
		if be.activeField < total-1 {
			be.activeField++
		}
	case "k", "up":
		if be.activeField > 0 {
			be.activeField--
		}
	case "enter", " ":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				be.ddOpen = true
				be.ddOptIdx = f.SelIdx
			}
		} else {
			eventIdx := be.activeField - brokerCount
			if eventIdx < eventCount {
				ed.form = be.withDomainNames(copyFields(ed.items[eventIdx]))
				ed.formIdx = 0
				ed.itemIdx = eventIdx
				ed.itemView = beListViewForm
			}
		}
	case "H", "shift+left":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "a":
		be.Events = append(be.Events, manifest.EventDef{})
		ed.items = append(ed.items, be.withDomainNames(defaultEventFields()))
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = brokerCount + ed.itemIdx
	case "d":
		eventIdx := be.activeField - brokerCount
		if eventIdx >= 0 && eventIdx < eventCount {
			be.Events = append(be.Events[:eventIdx], be.Events[eventIdx+1:]...)
			ed.items = append(ed.items[:eventIdx], ed.items[eventIdx+1:]...)
			if be.activeField > brokerCount && be.activeField >= brokerCount+len(ed.items) {
				be.activeField = brokerCount + len(ed.items) - 1
			}
		}
	case "i":
		if be.activeField < brokerCount {
			return be.tryEnterInsert()
		}
	}
	return be, nil
}

func (be BackendEditor) updateEventForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			be.ddOpen = true
			be.ddOptIdx = f.SelIdx
		} else {
			return be.enterEventFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterEventFormInsert()
		}
	case "b", "esc":
		be.saveEventForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterEventFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveEventForm() {
	ed := &be.eventEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	evt := manifest.EventDef{
		Name:        fieldGet(ed.form, "name"),
		Domain:      fieldGet(ed.form, "domain"),
		Description: fieldGet(ed.form, "description"),
	}
	if ed.itemIdx < len(be.Events) {
		be.Events[ed.itemIdx] = evt
	}
}

// visibleEnvFields returns the ENV fields filtered by the current arch.
// For monolith, monolith_lang and monolith_fw are shown.
// For other archs, they are hidden.
// cors_origins is only shown when cors_strategy is "Strict allowlist".
func (be BackendEditor) visibleEnvFields() []Field {
	arch := be.currentArch()
	corsStrategy := fieldGet(be.EnvFields, "cors_strategy")
	var out []Field
	for _, f := range be.EnvFields {
		if (f.Key == "monolith_lang" || f.Key == "monolith_fw") && arch != "monolith" {
			continue
		}
		if f.Key == "cors_origins" && corsStrategy != "Strict allowlist" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// currentEditableFields returns a pointer to the current tab's field slice.
// For ENV, we return nil (use visibleEnvFields instead) but we keep it for
// generic field navigation — actual navigation uses visibleEnvFieldIdx.
func (be *BackendEditor) currentEditableFields() *[]Field {
	switch be.activeTab() {
	case beTabEnv:
		return &be.EnvFields
	case beTabMessaging:
		return &be.MessagingFields
	case beTabAPIGW:
		return &be.APIGWFields
	case beTabAuth:
		return &be.AuthFields
	case beTabSecurity:
		return &be.securityFields
	}
	return nil
}

// mutableFieldPtr returns a pointer to the active field for the current tab.
// For the ENV tab, it resolves through the visible fields to find the correct
// pointer in the underlying EnvFields slice.
func (be *BackendEditor) mutableFieldPtr() *Field {
	if be.activeTab() == beTabEnv {
		visible := be.visibleEnvFields()
		if be.activeField < 0 || be.activeField >= len(visible) {
			return nil
		}
		key := visible[be.activeField].Key
		for i := range be.EnvFields {
			if be.EnvFields[i].Key == key {
				return &be.EnvFields[i]
			}
		}
		return nil
	}
	fields := be.currentEditableFields()
	if fields == nil {
		return nil
	}
	if be.activeField >= 0 && be.activeField < len(*fields) {
		return &(*fields)[be.activeField]
	}
	return nil
}

func (be BackendEditor) tryEnterInsert() (BackendEditor, tea.Cmd) {
	fields := be.currentEditableFields()
	if fields == nil {
		return be, nil
	}
	n := len(*fields)
	for range n {
		if be.activeField >= n {
			break
		}
		f := (*fields)[be.activeField]
		if f.Kind == KindText || f.Kind == KindTextArea {
			be.internalMode = beInsert
			be.formInput.SetValue(f.Value)
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
		be.activeField = (be.activeField + 1) % n
	}
	return be, nil
}

func (be *BackendEditor) saveInput() {
	val := be.formInput.Value()

	// Check if we're in a service form
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Check if we're in a comm form
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Check if we're in an event form
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Check if we're in a jobs form
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) && be.jobsForm[be.jobsFormIdx].Kind == KindText {
			be.jobsForm[be.jobsFormIdx].Value = val
		}
		return
	}
	// Generic field stores
	fields := be.currentEditableFields()
	if fields == nil {
		return
	}
	if be.activeField < len(*fields) && (*fields)[be.activeField].Kind == KindText {
		(*fields)[be.activeField].Value = val
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (be BackendEditor) View(w, h int) string {
	be.width = w
	be.formInput.Width = w - 22
	if !be.ArchConfirmed {
		return be.viewArchSelect(w, h)
	}
	return be.viewSubTabs(w, h)
}

func (be BackendEditor) viewArchSelect(w, h int) string {
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend — Choose an architecture pattern"),
		"",
	)

	current := beArchOptions[be.ArchIdx]
	label := StyleFieldKey.Render("arch_pattern  ")
	val := StyleFieldValActive.Render(current.label) + StyleSelectArrow.Render(" ▾")
	row := "     " + label + StyleEquals.Render(" = ") + val
	raw := lipgloss.Width(row)
	if raw < w {
		row += strings.Repeat(" ", w-raw)
	}
	lines = append(lines, activeCurLineStyle().Render(row))

	if be.dropdownOpen {
		lines = append(lines, "")
		for i, opt := range beArchOptions {
			isCur := i == be.dropdownIdx
			var cursor string
			if isCur {
				cursor = StyleCurLineNum.Render("  ▶ ")
			} else {
				cursor = "      "
			}
			labelPart := fmt.Sprintf("%-20s", opt.label)
			var optRow string
			if isCur {
				optRow = cursor +
					StyleFieldValActive.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
				rw := lipgloss.Width(optRow)
				if rw < w {
					optRow += strings.Repeat(" ", w-rw)
				}
				optRow = activeCurLineStyle().Render(optRow)
			} else {
				optRow = cursor +
					StyleFieldKey.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
			}
			lines = append(lines, optRow)
		}
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewSubTabs(w, h int) string {
	var lines []string

	opt := beArchOptions[be.ArchIdx]
	archStr := StyleFieldValActive.Render(opt.label)
	hint := StyleSectionDesc.Render("  (b: change arch)")
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend · ")+archStr+hint,
		"",
		renderSubTabBar(be.tabLabels(), be.activeTabIdx, w),
		"",
	)

	tab := be.activeTab()
	switch tab {
	case beTabEnv:
		if be.envEnabled {
			envFields := be.visibleEnvFields()
			lines = append(lines, renderFormFields(w, envFields, be.activeField, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabServices:
		const beListHeaderH = 2
		svcLines := be.viewServiceEditor(w)
		if be.serviceEditor.itemView == beListViewList {
			svcLines = appendViewport(svcLines, beListHeaderH, be.serviceEditor.itemIdx, h-4)
		}
		lines = append(lines, svcLines...)
	case beTabComm:
		commLines := be.viewCommEditor(w)
		if be.commEditor.itemView == beListViewList {
			commLines = appendViewport(commLines, 2, be.commEditor.itemIdx, h-4)
		}
		lines = append(lines, commLines...)
	case beTabMessaging:
		msgLines := be.viewMessaging(w)
		if be.eventEditor.itemView == beListViewList {
			msgLines = appendViewport(msgLines, 2, be.eventEditor.itemIdx, h-4)
		}
		lines = append(lines, msgLines...)
	case beTabAPIGW:
		if be.apiGWEnabled {
			lines = append(lines, renderFormFields(w, be.APIGWFields, be.activeField, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabJobs:
		jobLines := be.viewJobs(w)
		if be.jobsSubView == beViewList {
			jobLines = appendViewport(jobLines, 2, be.jobsIdx, h-4)
		}
		lines = append(lines, jobLines...)
	case beTabSecurity:
		if be.secEnabled {
			lines = append(lines, renderFormFields(w, be.securityFields, be.activeField, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabAuth:
		if be.authEnabled {
			lines = append(lines, renderFormFields(w, be.AuthFields, be.activeField, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewServiceEditor(w int) []string {
	ed := be.serviceEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Service Units — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no services yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				name := fieldGet(item, "name")
				if name == "" {
					name = fmt.Sprintf("(service #%d)", i+1)
				}
				lang := fieldGet(item, "language")
				fw := fieldGet(item, "framework")
				extra := lang
				if fw != "" {
					extra += " / " + fw
				}
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}
	// Form view
	idx := ed.itemIdx
	name := "(new service)"
	if idx < len(ed.items) {
		n := fieldGet(ed.form, "name")
		if n != "" {
			name = n
		}
	}

	arch := be.currentArch()
	var fields []Field
	filteredActiveIdx := ed.formIdx
	skippedBefore := 0
	for i, f := range ed.form {
		// For monolith: language and framework are defined at top level (ENV tab)
		if arch == "monolith" && (f.Key == "language" || f.Key == "framework") {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		// Hide pattern_tag for non-hybrid arches
		if arch != "hybrid" && f.Key == "pattern_tag" {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		fields = append(fields, f)
	}
	filteredActiveIdx = ed.formIdx - skippedBefore
	if filteredActiveIdx < 0 {
		filteredActiveIdx = 0
	}

	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, fields, filteredActiveIdx, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
	return lines
}

func (be BackendEditor) viewCommEditor(w int) []string {
	ed := be.commEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Communication Links — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no links yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				from := fieldGet(item, "from")
				to := fieldGet(item, "to")
				if from == "" {
					from = "?"
				}
				if to == "" {
					to = "?"
				}
				proto := fieldGet(item, "protocol")
				name := from + " → " + to
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, proto))
			}
		}
		return lines
	}
	from := fieldGet(be.commEditor.form, "from")
	to := fieldGet(be.commEditor.form, "to")
	title := from + " → " + to
	if from == "" && to == "" {
		title = "(new link)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(title), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
	return lines
}

func (be BackendEditor) viewMessaging(w int) []string {
	ed := be.eventEditor
	if ed.itemView == beListViewForm {
		name := fieldGet(ed.form, "name")
		if name == "" {
			name = "(new event)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
		return lines
	}

	// Combined view: broker config fields + event catalog list
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # Messaging Broker + Event Catalog"), "")

	brokerCount := len(be.MessagingFields)
	// Render broker fields in upper section
	const msgDDIndent = 21 // lineNumW(4) + labelW(14) + eqW(3)
	for i, f := range be.MessagingFields {
		isCur := i == be.activeField
		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}
		eq := StyleEquals.Render(" = ")
		val := f.DisplayValue()
		var valStr string
		if isCur && be.ddOpen {
			valStr = StyleFieldValActive.Render(val) + StyleSelectArrow.Render(" ▴")
		} else if isCur {
			valStr = StyleFieldValActive.Render(val) + StyleSelectArrow.Render(" ▾")
		} else {
			valStr = StyleFieldVal.Render(val) + StyleSelectArrow.Render(" ▾")
		}
		row := lineNo + keyStr + eq + valStr
		if isCur {
			rw := lipgloss.Width(row)
			if rw < w {
				row += strings.Repeat(" ", w-rw)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
		// Inline dropdown for active broker field
		if isCur && be.ddOpen {
			indent := strings.Repeat(" ", msgDDIndent)
			for j, opt := range f.Options {
				isHL := j == be.ddOptIdx
				var optRow string
				if isHL {
					optRow = indent + StyleFieldValActive.Render("▶ "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = activeCurLineStyle().Render(optRow)
				} else {
					optRow = indent + StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	// Divider + event catalog
	lines = append(lines, "", StyleSectionDesc.Render("  ── Event Catalog (a: add  d: delete  Enter: edit) ──"), "")

	if len(ed.items) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no events yet — press 'a' to add)"))
	} else {
		for i, item := range ed.items {
			globalIdx := brokerCount + i
			isCur := globalIdx == be.activeField
			name := fieldGet(item, "name")
			if name == "" {
				name = fmt.Sprintf("(event #%d)", i+1)
			}
			domain := fieldGet(item, "domain")
			lines = append(lines, renderListItem(w, isCur, "  ▶ ", name, domain))
		}
	}
	return lines
}

// ── Jobs updates ──────────────────────────────────────────────────────────────

func (be BackendEditor) updateJobsList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.jobQueues)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.jobsIdx < n-1 {
			be.jobsIdx++
		}
	case "k", "up":
		if be.jobsIdx > 0 {
			be.jobsIdx--
		}
	case "a":
		be.jobQueues = append(be.jobQueues, manifest.JobQueueDef{})
		be.jobsIdx = len(be.jobQueues) - 1
		be.jobsForm = defaultJobQueueFormFields()
		be.jobsFormIdx = 0
		be.jobsSubView = beViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.jobQueues = append(be.jobQueues[:be.jobsIdx], be.jobQueues[be.jobsIdx+1:]...)
			if be.jobsIdx > 0 && be.jobsIdx >= len(be.jobQueues) {
				be.jobsIdx = len(be.jobQueues) - 1
			}
		}
	case "enter":
		if n > 0 {
			jq := be.jobQueues[be.jobsIdx]
			be.jobsForm = defaultJobQueueFormFields()
			be.jobsForm = setFieldValue(be.jobsForm, "name", jq.Name)
			be.jobsForm = setFieldValue(be.jobsForm, "technology", jq.Technology)
			be.jobsForm = setFieldValue(be.jobsForm, "concurrency", jq.Concurrency)
			be.jobsForm = setFieldValue(be.jobsForm, "max_retries", jq.MaxRetries)
			be.jobsForm = setFieldValue(be.jobsForm, "retry_policy", jq.RetryPolicy)
			be.jobsForm = setFieldValue(be.jobsForm, "dlq", jq.DLQ)
			be.jobsFormIdx = 0
			be.jobsSubView = beViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "b":
		be.ArchConfirmed = false
	}
	return be, nil
}

func (be BackendEditor) updateJobsForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.jobsForm)
	switch key.String() {
	case "j", "down":
		if be.jobsFormIdx < n-1 {
			be.jobsFormIdx++
		}
		be.activeField = be.jobsFormIdx
	case "k", "up":
		if be.jobsFormIdx > 0 {
			be.jobsFormIdx--
		}
		be.activeField = be.jobsFormIdx
	case "enter", " ":
		if be.jobsFormIdx < n {
			f := &be.jobsForm[be.jobsFormIdx]
			if f.Kind == KindSelect {
				be.ddOpen = true
				be.ddOptIdx = f.SelIdx
			} else {
				return be.enterJobsFormInsert()
			}
		}
	case "H", "shift+left":
		if be.jobsFormIdx < n {
			f := &be.jobsForm[be.jobsFormIdx]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "i", "a":
		if be.jobsFormIdx < n && be.jobsForm[be.jobsFormIdx].Kind == KindText {
			return be.enterJobsFormInsert()
		}
	case "b", "esc":
		be.saveJobsForm()
		be.jobsSubView = beViewList
	}
	return be, nil
}

func (be BackendEditor) enterJobsFormInsert() (BackendEditor, tea.Cmd) {
	if be.jobsFormIdx >= len(be.jobsForm) {
		return be, nil
	}
	f := be.jobsForm[be.jobsFormIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveJobsForm() {
	if be.jobsIdx >= len(be.jobQueues) {
		return
	}
	jq := &be.jobQueues[be.jobsIdx]
	jq.Name = fieldGet(be.jobsForm, "name")
	jq.Technology = fieldGet(be.jobsForm, "technology")
	jq.Concurrency = fieldGet(be.jobsForm, "concurrency")
	jq.MaxRetries = fieldGet(be.jobsForm, "max_retries")
	jq.RetryPolicy = fieldGet(be.jobsForm, "retry_policy")
	jq.DLQ = fieldGet(be.jobsForm, "dlq")
}

// ── Security updates ──────────────────────────────────────────────────────────

func (be BackendEditor) updateSecurity(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	if !be.secEnabled {
		switch k {
		case "a":
			be.secEnabled = true
			be.activeField = 0
		case "h", "left":
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		case "l", "right":
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		case "b":
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}
	// Security uses generic field navigation via currentEditableFields / mutableFieldPtr
	// which already handles beTabSecurity. Just fall through to normal key handling.
	n := len(be.securityFields)

	// Vim count prefix
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		be.countBuf += k
		be.gBuf = false
		return be, nil
	}
	if k == "0" && be.countBuf != "" {
		be.countBuf += "0"
		be.gBuf = false
		return be, nil
	}

	switch k {
	case "j", "down":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField < n-1 {
				be.activeField++
			}
		}
	case "k", "up":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField > 0 {
				be.activeField--
			}
		}
	case "g":
		if be.gBuf {
			be.activeField = 0
			be.gBuf = false
		} else {
			be.gBuf = true
		}
		be.countBuf = ""
	case "G":
		be.countBuf = ""
		be.gBuf = false
		be.activeField = n - 1
	case "h", "left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	case "b":
		be.countBuf = ""
		be.gBuf = false
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "enter", " ":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == KindSelect {
				be.ddOpen = true
				be.ddOptIdx = f.SelIdx
			}
		}
	case "H", "shift+left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "D":
		be.countBuf = ""
		be.gBuf = false
		be.secEnabled = false
		be.securityFields = defaultSecurityFields()
		be.activeField = 0
	default:
		be.countBuf = ""
		be.gBuf = false
	}
	return be, nil
}

func (be BackendEditor) viewJobs(w int) []string {
	if be.jobsSubView == beViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Job Queues — a: add  d: delete  Enter: edit"), "")
		if len(be.jobQueues) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no job queues yet — press 'a' to add)"))
		} else {
			for i, jq := range be.jobQueues {
				name := jq.Name
				if name == "" {
					name = fmt.Sprintf("(queue #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == be.jobsIdx, "  ▶ ", name, jq.Technology))
			}
		}
		return lines
	}
	// Form view
	name := fieldGet(be.jobsForm, "name")
	if name == "" {
		name = "(new job queue)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, be.jobsForm, be.jobsFormIdx, be.internalMode == beInsert, be.formInput, be.ddOpen, be.ddOptIdx)...)
	return lines
}

// AuthRoleOptions returns a list of common auth roles for use in frontend page forms.
// Based on the authz_model configured in auth settings.
func (be BackendEditor) AuthRoleOptions() []string {
	model := fieldGet(be.AuthFields, "authz_model")
	switch model {
	case "RBAC":
		return []string{"admin", "user", "moderator", "editor", "viewer", "superadmin"}
	case "ABAC":
		return []string{"admin", "user", "owner", "manager", "auditor"}
	case "ACL":
		return []string{"admin", "read", "write", "execute", "owner"}
	default:
		return []string{"admin", "user", "moderator", "editor", "viewer"}
	}
}

// ServiceNames returns the names of all created service units for cross-reference.
func (be BackendEditor) ServiceNames() []string {
	var names []string
	for _, item := range be.serviceEditor.items {
		name := fieldGet(item, "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ServiceDefs returns full service definitions for technology-based protocol filtering.
func (be BackendEditor) ServiceDefs() []manifest.ServiceDef {
	defs := make([]manifest.ServiceDef, 0, len(be.serviceEditor.items))
	for _, item := range be.serviceEditor.items {
		defs = append(defs, serviceDefFromFields(item))
	}
	return defs
}

// copyFields makes a deep copy of a field slice.
func copyFields(src []Field) []Field {
	dst := make([]Field, len(src))
	for i, f := range src {
		dst[i] = f
		if f.Options != nil {
			dst[i].Options = make([]string, len(f.Options))
			copy(dst[i].Options, f.Options)
		}
	}
	return dst
}
