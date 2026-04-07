package backend

import "github.com/vibe-menu/internal/ui/core"

// ── Env fields ───────────────────────────────────────────────────────────────

func defaultEnvFields() []core.Field {
	return []core.Field{
		// Monolith-only: language and framework defined once at top level
		{
			Key: "monolith_lang", Label: "language      ", Kind: core.KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "monolith_lang_ver", Label: "lang version  ", Kind: core.KindSelect,
			Options: core.LangVersions["Go"],
			Value:   core.LangVersions["Go"][0],
		},
		{
			Key: "monolith_fw", Label: "framework     ", Kind: core.KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "monolith_fw_ver", Label: "fw version    ", Kind: core.KindSelect,
			Options: core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber"),
			Value:   core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber")[0],
		},
		// Monolith-only: shared environment for all services.
		{
			Key:     "environment",
			Label:   "environment   ",
			Kind:    core.KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
		// Monolith-only: global health dependencies for the entire application.
		{Key: "health_deps", Label: "Health Deps   ", Kind: core.KindMultiSelect},
	}
}

// ── Messaging fields ─────────────────────────────────────────────────────────

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
	brokerTech := core.FieldGet(be.MessagingFields, "broker_tech")
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

func defaultMessagingFields() []core.Field {
	return []core.Field{
		{
			Key: "broker_tech", Label: "broker_tech   ", Kind: core.KindSelect,
			Options: []string{
				"Kafka", "NATS", "RabbitMQ", "Redis Streams",
				"AWS SQS/SNS", "Google Pub/Sub", "Azure Service Bus", "Pulsar",
			},
			Value: "Kafka",
		},
		{
			Key: "deployment", Label: "deployment    ", Kind: core.KindSelect,
			Options: []string{"Managed (cloud)", "Self-hosted", "Embedded"},
			Value:   "Managed (cloud)",
		},
		{
			Key: "serialization", Label: "serialization ", Kind: core.KindSelect,
			Options: []string{"JSON", "Protobuf", "Avro", "MessagePack", "CloudEvents"},
			Value:   "JSON",
		},
		{
			Key: "delivery", Label: "delivery      ", Kind: core.KindSelect,
			Options: []string{"At-most-once", "At-least-once", "Exactly-once"},
			Value:   "At-least-once",
		},
		{
			Key:     "environment",
			Label:   "environment   ",
			Kind:    core.KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
	}
}

func defaultEventFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "publisher_service", Label: "publisher_svc ", Kind: core.KindText},
		{Key: "consumer_service", Label: "consumer_svc  ", Kind: core.KindText},
		{Key: "dto", Label: "dto           ", Kind: core.KindSelect},
		{Key: "description", Label: "description   ", Kind: core.KindText},
	}
}

// withEventNames returns a copy of fields where publisher_service and
// consumer_service are upgraded to core.KindSelect dropdowns populated with service
// names, and dto is upgraded to a core.KindSelect populated with available DTO names.
func (be BackendEditor) withEventNames(fields []core.Field) []core.Field {
	svcNames := be.ServiceNames()
	dtoOpts, defaultDTO := core.NoneOrPlaceholder(be.availableDTOs, "(no DTOs configured)")
	out := core.CopyFields(fields)
	for i := range out {
		switch out[i].Key {
		case "publisher_service", "consumer_service":
			out[i].Kind = core.KindSelect
			out[i].Options = svcNames
			prev := out[i].Value
			out[i].Value = core.PlaceholderFor(svcNames, "(no services configured)")
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
			out[i].Kind = core.KindSelect
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

// ── API Gateway fields ───────────────────────────────────────────────────────

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

func defaultAPIGWFields() []core.Field {
	return []core.Field{
		{
			Key: "environment", Label: "environment   ", Kind: core.KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
		{
			Key: "technology", Label: "technology    ", Kind: core.KindSelect,
			Options: []string{
				"Kong", "Traefik", "NGINX", "Envoy",
				"AWS API Gateway", "Cloudflare Workers", "Custom (specify)", "None",
			},
			Value: "Kong",
		},
		{
			Key: "routing", Label: "routing       ", Kind: core.KindSelect,
			Options: []string{"Path-based", "Header-based", "Domain-based"},
			Value:   "Path-based",
		},
		{
			Key: "features", Label: "features      ", Kind: core.KindMultiSelect,
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
			Key: "endpoints", Label: "endpoints     ", Kind: core.KindMultiSelect,
		},
	}
}

// ── Auth & Security fields ───────────────────────────────────────────────────

func defaultAuthFields() []core.Field {
	return []core.Field{
		{
			Key: "strategy", Label: "strategy      ", Kind: core.KindMultiSelect,
			Options: []string{
				"JWT (stateless)", "Session-based", "OAuth 2.0 / OIDC",
				"API Keys", "mTLS", "None",
			},
		},
		{
			Key: "provider", Label: "provider      ", Kind: core.KindSelect,
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
			Kind:    core.KindSelect,
			Options: []string{"None (external)"},
			Value:   "None (external)",
		},
		{
			Key: "authz_model", Label: "authz_model   ", Kind: core.KindSelect,
			Options: []string{"RBAC", "ABAC", "ACL", "ReBAC", "Policy-based (OPA/Cedar)", "Custom"},
			Value:   "RBAC",
		},
		{
			Key: "token_storage", Label: "token_storage ", Kind: core.KindMultiSelect,
			Options: []string{
				"HttpOnly cookie", "Authorization header (Bearer)",
				"WebSocket protocol header", "Other",
			},
		},
		{
			Key: "session_mgmt", Label: "Session Mgmt  ", Kind: core.KindSelect,
			Options: []string{"Stateless (JWT only)", "Server-side sessions (Redis)", "Database sessions", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "refresh_token", Label: "refresh_token ", Kind: core.KindSelect,
			Options: []string{"None", "Rotating", "Non-rotating", "Sliding window"},
			Value:   "None",
		},
		{
			Key: "mfa", Label: "mfa           ", Kind: core.KindSelect,
			Options: []string{"None", "TOTP", "SMS", "Email", "Passkeys/WebAuthn"},
			Value:   "None",
		},
	}
}

func defaultSecurityFields() []core.Field {
	return []core.Field{
		{
			Key: "waf_provider", Label: "waf_provider  ", Kind: core.KindSelect,
			Options: []string{"Cloudflare WAF", "AWS WAF", "Cloud Armor", "Azure WAF", "ModSecurity", "NGINX ModSec", "None"},
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "waf_ruleset", Label: "waf_ruleset   ", Kind: core.KindSelect,
			Options: []string{"OWASP Core Rule Set", "Managed rules", "Custom", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "captcha", Label: "captcha       ", Kind: core.KindSelect,
			Options: []string{"hCaptcha", "reCAPTCHA v2", "reCAPTCHA v3", "Cloudflare Turnstile", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "bot_protection", Label: "bot_protection", Kind: core.KindSelect,
			Options: []string{"Cloudflare Bot Management", "Imperva", "DataDome", "Custom", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "rate_limit_strategy", Label: "rate_limit    ", Kind: core.KindSelect,
			Options: []string{"Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "API Gateway", "None"},
			Value:   "None", SelIdx: 5,
		},
		{
			Key: "rate_limit_backend", Label: "rl_backend    ", Kind: core.KindSelect,
			Options: []string{"Redis", "Memcached", "In-memory", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "ddos_protection", Label: "ddos_protect  ", Kind: core.KindSelect,
			Options: []string{"CDN-level (Cloudflare)", "Provider-managed", "None"},
			Value:   "None", SelIdx: 2,
		},
		{
			Key: "internal_mtls", Label: "internal_mtls ", Kind: core.KindSelect,
			Options: []string{"Enabled", "Disabled"},
			Value:   "Disabled", SelIdx: 1,
		},
	}
}

// defaultRoleFormFields returns form fields for a role, wiring permissions and
// inheritable role names as core.KindMultiSelect dropdowns.
func defaultRoleFormFields(permNames, roleNames []string) []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "description", Label: "description   ", Kind: core.KindText},
		{Key: "permissions", Label: "permissions   ", Kind: core.KindMultiSelect,
			Options: permNames,
			Value:   core.PlaceholderFor(permNames, "(no permissions configured)"),
		},
		{Key: "inherits", Label: "inherits      ", Kind: core.KindMultiSelect,
			Options: roleNames,
			Value:   core.PlaceholderFor(roleNames, "(no roles configured)"),
		},
	}
}

func defaultPermFormFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "description", Label: "description   ", Kind: core.KindText},
	}
}

// rateBackendOptions returns rate-limit backend options combining cache aliases
// with the built-in "In-memory" and "None" fallbacks.
func (be BackendEditor) rateBackendOptions() []string {
	opts := make([]string, 0, len(be.cacheAliases)+2)
	opts = append(opts, be.cacheAliases...)
	opts = append(opts, "In-memory", "None")
	return opts
}
