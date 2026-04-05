package ui

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
)

// ── language / framework / linter tables ─────────────────────────────────────

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

var backendLintersByLang = map[string][]string{
	"Go":              {"golangci-lint", "staticcheck", "go vet", "None"},
	"TypeScript/Node": {"ESLint", "Biome", "TSLint (legacy)", "None"},
	"Python":          {"Ruff", "Flake8", "Pylint", "mypy", "None"},
	"Java":            {"Checkstyle", "SpotBugs", "PMD", "SonarLint", "None"},
	"Kotlin":          {"ktlint", "detekt", "SonarLint", "None"},
	"C#/.NET":         {"Roslyn Analyzers", "StyleCop", "SonarLint", "None"},
	"Rust":            {"Clippy", "cargo-audit", "None"},
	"Ruby":            {"RuboCop", "StandardRB", "None"},
	"PHP":             {"PHP-CS-Fixer", "PHPStan", "Psalm", "None"},
	"Elixir":          {"Credo", "Dialyxir", "None"},
	"Other":           {"Custom", "None"},
}

// errorFormatByProtocol maps a technology/protocol name to the error formats
// that are idiomatic for it. Used to narrow the error_format dropdown based on
// the technologies selected for a service.
var errorFormatByProtocol = map[string][]string{
	"REST":      {"RFC 7807 (Problem Details)", "Custom JSON envelope"},
	"GraphQL":   {"GraphQL spec errors", "Custom extensions"},
	"gRPC":      {"gRPC status codes", "google.rpc.Status"},
	"WebSocket": {"Custom JSON envelope"},
	"SSE":       {"Custom JSON envelope"},
	"tRPC":      {"tRPC error format"},
}

// errorFormatOptsForTechs derives the appropriate error_format option list from
// the selected technologies. Options from each matching protocol are unioned in
// encounter order; "Platform default" is always appended as the last option.
// If no protocol matches, the static default set is returned.
func errorFormatOptsForTechs(techs []string) []string {
	seen := map[string]bool{"Platform default": true}
	var opts []string
	for _, tech := range techs {
		if formats, ok := errorFormatByProtocol[tech]; ok {
			for _, f := range formats {
				if !seen[f] {
					seen[f] = true
					opts = append(opts, f)
				}
			}
		}
	}
	if len(opts) == 0 {
		return []string{"RFC 7807 (Problem Details)", "Custom JSON envelope", "Platform default"}
	}
	return append(opts, "Platform default")
}

// ── field definitions ─────────────────────────────────────────────────────────
// Default field slices and manifest serialization helpers for BackendEditor.

func defaultEnvFields() []Field {
	return []Field{
		// Monolith-only: language and framework defined once at top level
		{
			Key: "monolith_lang", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "monolith_lang_ver", Label: "lang version  ", Kind: KindSelect,
			Options: langVersions["Go"],
			Value:   langVersions["Go"][0],
		},
		{
			Key: "monolith_fw", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "monolith_fw_ver", Label: "fw version    ", Kind: KindSelect,
			Options: compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber"),
			Value:   compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber")[0],
		},
		// Monolith-only: shared environment for all services.
		{
			Key:     "environment",
			Label:   "environment   ",
			Kind:    KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
		// Monolith-only: global health dependencies for the entire application.
		{Key: "health_deps", Label: "Health Deps   ", Kind: KindMultiSelect},
	}
}

func defaultStackConfigFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "language", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages, Value: "Go",
		},
		{
			Key: "language_version", Label: "lang version  ", Kind: KindSelect,
			Options: langVersions["Go"], Value: langVersions["Go"][0],
		},
		{
			Key: "framework", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"], Value: "Fiber",
		},
		{
			Key: "framework_version", Label: "fw version    ", Kind: KindSelect,
			Options: compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber"),
			Value:   compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber")[0],
		},
	}
}

func defaultServiceFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "responsibility", Label: "responsibility", Kind: KindText},
		// config_ref: picks a StackConfig from the CONFIG tab (non-monolith arches only).
		{
			Key:     "config_ref",
			Label:   "stack config  ",
			Kind:    KindSelect,
			Options: []string{"(no configs defined)"},
			Value:   "(no configs defined)",
		},
		{
			Key: "language", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "language_version", Label: "lang version  ", Kind: KindSelect,
			Options: langVersions["Go"],
			Value:   langVersions["Go"][0],
		},
		{
			Key: "framework", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "framework_version", Label: "fw version    ", Kind: KindSelect,
			Options: compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber"),
			Value:   compatibleFrameworkVersions("Go", langVersions["Go"][0], "Fiber")[0],
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
		{Key: "health_deps", Label: "Health Deps   ", Kind: KindMultiSelect},
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
		// environment is a KindSelect populated dynamically from InfraPillar.Environments.
		{
			Key: "environment", Label: "environment   ", Kind: KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
	}
}

func serviceFieldsFromDef(s manifest.ServiceDef) []Field {
	f := defaultServiceFields()
	f = setFieldValue(f, "name", s.Name)
	f = setFieldValue(f, "responsibility", s.Responsibility)
	if s.ConfigRef != "" {
		f = setFieldValue(f, "config_ref", s.ConfigRef)
	}
	if s.Language != "" {
		f = setFieldValue(f, "language", s.Language)
		// Update language_version options for this language.
		if vers, ok := langVersions[s.Language]; ok {
			for i := range f {
				if f[i].Key == "language_version" {
					f[i].Options = vers
					f[i].SelIdx = 0
					f[i].Value = vers[0]
					if s.LanguageVersion != "" {
						f = setFieldValue(f, "language_version", s.LanguageVersion)
					}
					break
				}
			}
		}
		// Update framework options based on language.
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
		// Update framework_version options based on language + language_version + framework.
		lang := s.Language
		langVer := s.LanguageVersion
		fw := s.Framework
		if fw == "" {
			if opts, ok := backendFrameworksByLang[lang]; ok && len(opts) > 0 {
				fw = opts[0]
			}
		}
		fwVers := compatibleFrameworkVersions(lang, langVer, fw)
		for i := range f {
			if f[i].Key == "framework_version" {
				f[i].Options = fwVers
				f[i].SelIdx = 0
				f[i].Value = fwVers[0]
				if s.FrameworkVersion != "" {
					f = setFieldValue(f, "framework_version", s.FrameworkVersion)
				}
				break
			}
		}
	}
	if s.PatternTag != "" {
		f = setFieldValue(f, "pattern_tag", s.PatternTag)
	}
	if s.Environment != "" {
		f = setFieldValue(f, "environment", s.Environment)
	}
	// Restore technologies selections and narrow error_format options accordingly.
	if len(s.Technologies) > 0 {
		for i := range f {
			if f[i].Key != "technologies" {
				continue
			}
			f[i].SelectedIdxs = nil
			for _, tech := range s.Technologies {
				for j, opt := range f[i].Options {
					if opt == tech {
						f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						break
					}
				}
			}
			break
		}
		errOpts := errorFormatOptsForTechs(s.Technologies)
		for i := range f {
			if f[i].Key == "error_format" {
				f[i].Options = errOpts
				break
			}
		}
	}
	// Restore error_format value (must happen after options are set).
	if s.ErrorFormat != "" {
		f = setFieldValue(f, "error_format", s.ErrorFormat)
	}
	// Restore health_deps selections — options are populated dynamically later
	// via SetDBSourceAliases; store names in Value for lazy restoration.
	if len(s.HealthDeps) > 0 {
		for i := range f {
			if f[i].Key == "health_deps" {
				f[i].Value = strings.Join(s.HealthDeps, ", ")
				break
			}
		}
	}
	return f
}

func serviceDefFromFields(fields []Field) manifest.ServiceDef {
	// Read technologies multiselect
	var techs []string
	var healthDeps []string
	for _, f := range fields {
		switch f.Key {
		case "technologies":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					techs = append(techs, f.Options[idx])
				}
			}
		case "health_deps":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					healthDeps = append(healthDeps, f.Options[idx])
				}
			}
		}
	}
	env := fieldGet(fields, "environment")
	if env == "(no environments configured)" {
		env = ""
	}
	cfgRef := fieldGet(fields, "config_ref")
	if cfgRef == "(no configs defined)" {
		cfgRef = ""
	}
	return manifest.ServiceDef{
		Name:             fieldGet(fields, "name"),
		Responsibility:   fieldGet(fields, "responsibility"),
		ConfigRef:        cfgRef,
		Language:         fieldGet(fields, "language"),
		LanguageVersion:  fieldGet(fields, "language_version"),
		Framework:        fieldGet(fields, "framework"),
		FrameworkVersion: fieldGet(fields, "framework_version"),
		PatternTag:       fieldGet(fields, "pattern_tag"),
		Technologies:     techs,
		HealthDeps:       healthDeps,
		ErrorFormat:      fieldGet(fields, "error_format"),
		ServiceDiscovery: fieldGet(fields, "service_discovery"),
		Environment:      env,
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
		// Options populated dynamically via withDTONames(); Value stores
		// comma-sep names for lazy restoration before options are injected.
		{Key: "payload_dto", Label: "payload_dto   ", Kind: KindMultiSelect},
		// Only shown when direction is "Bidirectional (↔)".
		{Key: "response_dto", Label: "response_dto  ", Kind: KindMultiSelect},
	}
}

func commLinkFromFields(fields []Field) manifest.CommLink {
	var resilience []string
	var payloadDTOs []string
	var responseDTOs []string
	for _, f := range fields {
		switch f.Key {
		case "resilience":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					resilience = append(resilience, f.Options[idx])
				}
			}
		case "payload_dto":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					payloadDTOs = append(payloadDTOs, f.Options[idx])
				}
			}
		case "response_dto":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					responseDTOs = append(responseDTOs, f.Options[idx])
				}
			}
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
		DTOs:               payloadDTOs,
		ResponseDTOs:       responseDTOs,
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
	// Store selected DTO names in Value for lazy restoration via withDTONames().
	if len(l.DTOs) > 0 {
		for i := range f {
			if f[i].Key == "payload_dto" {
				f[i].Value = strings.Join(l.DTOs, ", ")
				break
			}
		}
	}
	if len(l.ResponseDTOs) > 0 {
		for i := range f {
			if f[i].Key == "response_dto" {
				f[i].Value = strings.Join(l.ResponseDTOs, ", ")
				break
			}
		}
	}
	return f
}

// brokerDeploymentOptions returns deployment options specific to the broker
// technology and the configured cloud provider.
func brokerDeploymentOptions(brokerTech, cloudProvider string) []string {
	switch cloudProvider {
	case "AWS":
		switch brokerTech {
		case "Kafka":
			return []string{"AWS MSK (managed)", "Self-hosted (EC2/K8s)"}
		case "RabbitMQ":
			return []string{"Amazon MQ", "Self-hosted"}
		case "Redis Streams":
			return []string{"ElastiCache", "Self-hosted"}
		case "NATS":
			return []string{"Synadia Cloud", "Self-hosted"}
		case "AWS SQS/SNS":
			return []string{"AWS SQS/SNS (managed)"}
		default:
			return []string{"Managed (cloud)", "Self-hosted"}
		}
	case "GCP":
		switch brokerTech {
		case "Kafka":
			return []string{"Confluent Cloud", "Self-hosted"}
		case "NATS":
			return []string{"Synadia Cloud", "Self-hosted"}
		case "Google Pub/Sub":
			return []string{"Google Pub/Sub (managed)"}
		default:
			return []string{"Managed (cloud)", "Self-hosted"}
		}
	case "Azure":
		switch brokerTech {
		case "Kafka":
			return []string{"Confluent Cloud", "Self-hosted"}
		case "NATS":
			return []string{"Synadia Cloud", "Self-hosted"}
		case "RabbitMQ":
			return []string{"Azure Service Bus (managed)", "Self-hosted"}
		case "Azure Service Bus":
			return []string{"Azure Service Bus (managed)"}
		default:
			return []string{"Managed (cloud)", "Self-hosted"}
		}
	case "":
		// Cloud provider not yet configured — keep generic options.
		return []string{"Managed (cloud)", "Self-hosted", "Embedded"}
	default:
		// Non-major cloud providers (Hetzner, Cloudflare, bare-metal, etc.)
		switch brokerTech {
		case "NATS":
			return []string{"Synadia Cloud", "Self-hosted", "Embedded"}
		default:
			return []string{"Self-hosted", "Embedded"}
		}
	}
}

// refreshMessagingDeploymentOptions re-derives the deployment field's Options
// from the current broker_tech value and the cached cloud provider.
func (be *BackendEditor) refreshMessagingDeploymentOptions() {
	brokerTech := fieldGet(be.MessagingFields, "broker_tech")
	opts := brokerDeploymentOptions(brokerTech, be.cloudProvider)
	for i := range be.MessagingFields {
		if be.MessagingFields[i].Key != "deployment" {
			continue
		}
		prev := be.MessagingFields[i].Value
		be.MessagingFields[i].Options = opts
		// Keep current value when still valid; otherwise reset to first option.
		found := false
		for j, o := range opts {
			if o == prev {
				be.MessagingFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			be.MessagingFields[i].SelIdx = 0
			be.MessagingFields[i].Value = opts[0]
		}
		break
	}
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
		{
			Key:     "environment",
			Label:   "environment   ",
			Kind:    KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
	}
}

func defaultEventFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "publisher_service", Label: "publisher_svc ", Kind: KindText},
		{Key: "consumer_service", Label: "consumer_svc  ", Kind: KindText},
		{Key: "dto", Label: "dto           ", Kind: KindSelect},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

// withEventNames returns a copy of fields where publisher_service and
// consumer_service are upgraded to KindSelect dropdowns populated with service
// names, and dto is upgraded to a KindSelect populated with available DTO names.
func (be BackendEditor) withEventNames(fields []Field) []Field {
	svcNames := be.ServiceNames()
	dtoOpts, defaultDTO := noneOrPlaceholder(be.availableDTOs, "(no DTOs configured)")
	out := copyFields(fields)
	for i := range out {
		switch out[i].Key {
		case "publisher_service", "consumer_service":
			out[i].Kind = KindSelect
			out[i].Options = svcNames
			prev := out[i].Value
			out[i].Value = placeholderFor(svcNames, "(no services configured)")
			out[i].SelIdx = 0
			for j, n := range svcNames {
				if n == prev {
					out[i].SelIdx = j
					out[i].Value = n
					break
				}
			}
			if len(svcNames) > 0 && out[i].Value == "" {
				out[i].Value = svcNames[0]
			}
		case "dto":
			out[i].Kind = KindSelect
			out[i].Options = dtoOpts
			prev := out[i].Value
			out[i].Value = defaultDTO
			out[i].SelIdx = 0
			for j, opt := range dtoOpts {
				if opt == prev {
					out[i].SelIdx = j
					out[i].Value = opt
					break
				}
			}
		}
	}
	return out
}

// apiGWTechOptionsForEnv returns the API gateway technology options appropriate
// for the given orchestrator and cloud provider combination.
func apiGWTechOptionsForEnv(orchestrator, cloudProvider string) []string {
	switch orchestrator {
	case "Kubernetes", "K3s":
		return []string{"Kong", "Traefik", "NGINX Ingress", "Envoy", "Custom (specify)", "None"}
	case "Docker Compose":
		return []string{"Traefik", "NGINX", "Custom (specify)", "None"}
	}
	switch cloudProvider {
	case "AWS":
		return []string{"AWS API Gateway", "Kong", "Custom (specify)", "None"}
	case "GCP":
		return []string{"Cloudflare Workers", "Custom (specify)", "None"}
	}
	// Default / unknown: full list.
	return []string{
		"Kong", "Traefik", "NGINX", "Envoy",
		"AWS API Gateway", "Cloudflare Workers", "Custom (specify)", "None",
	}
}

func defaultAPIGWFields() []Field {
	return []Field{
		{
			Key: "environment", Label: "environment   ", Kind: KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
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
		{
			// Options populated dynamically via SetEndpointNames(); Value stores
			// comma-sep names for lazy restoration before options are injected.
			Key: "endpoints", Label: "endpoints     ", Kind: KindMultiSelect,
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
			// Options populated lazily from configured services when provider is
			// Self-managed or Keycloak; otherwise stays as "None (external)".
			Key:     "service_unit",
			Label:   "service_unit  ",
			Kind:    KindSelect,
			Options: []string{"None (external)"},
			Value:   "None (external)",
		},
		{
			Key: "authz_model", Label: "authz_model   ", Kind: KindSelect,
			Options: []string{"RBAC", "ABAC", "ACL", "ReBAC", "Policy-based (OPA/Cedar)", "Custom"},
			Value:   "RBAC",
		},
		{
			Key: "token_storage", Label: "token_storage ", Kind: KindMultiSelect,
			Options: []string{
				"HttpOnly cookie", "Authorization header (Bearer)",
				"WebSocket protocol header", "Other",
			},
		},
		{
			Key: "session_mgmt", Label: "Session Mgmt  ", Kind: KindSelect,
			Options: []string{"Stateless (JWT only)", "Server-side sessions (Redis)", "Database sessions", "None"},
			Value:   "None", SelIdx: 3,
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
			Options: []string{"Cloudflare WAF", "AWS WAF", "Cloud Armor", "Azure WAF", "ModSecurity", "NGINX ModSec", "None"},
			Value:   "None", SelIdx: 6,
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
			Options: []string{"Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "API Gateway", "None"},
			Value:   "None", SelIdx: 5,
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
		{
			Key: "internal_mtls", Label: "internal_mtls ", Kind: KindSelect,
			Options: []string{"Enabled", "Disabled"},
			Value:   "Disabled", SelIdx: 1,
		},
	}
}

var jobQueueTechByLang = map[string][]string{
	"Go":              {"Asynq", "River", "Temporal", "Faktory", "Custom"},
	"TypeScript/Node": {"BullMQ", "Temporal", "Custom"},
	"Python":          {"Celery", "Temporal", "Custom"},
	"Ruby":            {"Sidekiq", "Temporal", "Custom"},
	"Java":            {"Temporal", "Custom"},
	"Kotlin":          {"Temporal", "Custom"},
	"C#/.NET":         {"Hangfire", "Temporal", "Custom"},
	"Rust":            {"Temporal", "Custom"},
	"PHP":             {"Laravel Queues", "Temporal", "Custom"},
	"Elixir":          {"Oban", "Temporal", "Custom"},
	"Other":           {"Temporal", "Custom"},
}

// jobQueueTechOptions returns filtered technology options based on languages.
// When no languages are configured, returns the full set.
func jobQueueTechOptions(langs []string) ([]string, string) {
	if len(langs) == 0 {
		return []string{"Temporal", "BullMQ", "Sidekiq", "Celery", "Faktory", "Asynq", "River", "Custom"}, "Temporal"
	}
	seen := make(map[string]bool)
	var opts []string
	for _, lang := range langs {
		for _, tech := range jobQueueTechByLang[lang] {
			if !seen[tech] {
				seen[tech] = true
				opts = append(opts, tech)
			}
		}
	}
	if len(opts) == 0 {
		return []string{"Temporal", "Custom"}, "Temporal"
	}
	return opts, opts[0]
}

func defaultJobQueueFormFields(services, dtos, langs, configNames []string) []Field {
	workerOpts, workerVal := noneOrPlaceholder(services, "(no services configured)")
	payloadOpts, payloadVal := noneOrPlaceholder(dtos, "(no DTOs configured)")
	techOpts, techVal := jobQueueTechOptions(langs)

	var cfgOpts []string
	var cfgVal string
	if len(configNames) > 0 {
		cfgOpts = append([]string{"(any)"}, configNames...)
		cfgVal = "(any)"
	} else {
		cfgOpts = []string{"(no configs defined)"}
		cfgVal = "(no configs defined)"
	}

	fields := []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
		{
			Key:     "config_ref",
			Label:   "stack config  ",
			Kind:    KindSelect,
			Options: cfgOpts,
			Value:   cfgVal,
		},
	}

	fields = append(fields, []Field{
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: techOpts,
			Value:   techVal,
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
			Options: OptionsOffOn, Value: "false",
		},
		{
			Key: "worker_service", Label: "worker_service", Kind: KindSelect,
			Options: workerOpts, Value: workerVal,
		},
		{
			Key: "payload_dto", Label: "payload_dto   ", Kind: KindSelect,
			Options: payloadOpts, Value: payloadVal,
		},
	}...)
	return fields
}

// defaultRoleFormFields returns form fields for a role, wiring permissions and
// inheritable role names as KindMultiSelect dropdowns.
func defaultRoleFormFields(permNames, roleNames []string) []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
		{Key: "permissions", Label: "permissions   ", Kind: KindMultiSelect,
			Options: permNames,
			Value:   placeholderFor(permNames, "(no permissions configured)"),
		},
		{Key: "inherits", Label: "inherits      ", Kind: KindMultiSelect,
			Options: roleNames,
			Value:   placeholderFor(roleNames, "(no roles configured)"),
		},
	}
}

func defaultPermFormFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

// ── Runtime field population ──────────────────────────────────────────────────

// rateBackendOptions returns rate-limit backend options combining cache aliases
// with the built-in "In-memory" and "None" fallbacks.
func (be BackendEditor) rateBackendOptions() []string {
	opts := make([]string, 0, len(be.cacheAliases)+2)
	opts = append(opts, be.cacheAliases...)
	opts = append(opts, "In-memory", "None")
	return opts
}

// withServiceNames returns a copy of fields where from/to are upgraded to
// KindSelect dropdowns populated with the current service names.
func (be BackendEditor) withServiceNames(fields []Field) []Field {
	names := be.ServiceNames()
	out := copyFields(fields)
	for i := range out {
		if out[i].Key != "from" && out[i].Key != "to" {
			continue
		}
		out[i].Kind = KindSelect
		out[i].Options = names
		out[i].Value = placeholderFor(names, "(no services configured)")
		out[i].SelIdx = 0
		for j, n := range names {
			if n == out[i].Value {
				out[i].SelIdx = j
				break
			}
		}
		if len(names) > 0 && out[i].Value == "" {
			out[i].Value = names[0]
		}
	}
	return out
}

// withDTONames returns a copy of fields where payload_dto and response_dto
// options are populated with the currently available DTO names, restoring any
// prior selections by matching option names.
func (be BackendEditor) withDTONames(fields []Field) []Field {
	out := copyFields(fields)
	for i := range out {
		if out[i].Key != "payload_dto" && out[i].Key != "response_dto" {
			continue
		}
		// Collect previously selected DTO names before replacing options.
		var selectedNames []string
		if len(out[i].Options) > 0 {
			for _, idx := range out[i].SelectedIdxs {
				if idx < len(out[i].Options) {
					selectedNames = append(selectedNames, out[i].Options[idx])
				}
			}
		} else if out[i].Value != "" {
			// Options not yet set — Value stores comma-sep names from commFieldsFromLink.
			selectedNames = strings.Split(out[i].Value, ", ")
		}
		out[i].Options = be.availableDTOs
		out[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range out[i].Options {
				if opt == name {
					out[i].SelectedIdxs = append(out[i].SelectedIdxs, j)
					break
				}
			}
		}
	}
	return out
}
