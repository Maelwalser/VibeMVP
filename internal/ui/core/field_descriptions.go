package core

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// fieldDescriptions maps fieldKey → optionValue → human-readable description.
var fieldDescriptions = map[string]map[string]string{

	// ── DATABASE ──────────────────────────────────────────────────────────────

	"type": {
		"PostgreSQL":    "Relational database with ACID transactions, JSON support, and rich indexing. Mature, open-source, widely hosted. Best all-round choice for most applications.",
		"MySQL":         "Relational database optimized for read-heavy workloads. Wide cloud support. Generates MySQL-compatible schema migrations and connection pooling.",
		"SQLite":        "Embedded file-based relational database. Zero server overhead. Ideal for local dev, testing, CLI tools, and edge deployments.",
		"MongoDB":       "Document-oriented NoSQL database with flexible schemas. Native JSON storage. Best for rapidly changing data shapes or content-heavy apps.",
		"DynamoDB":      "AWS-managed key-value and document store. Single-digit millisecond latency at any scale. Requires careful access-pattern design upfront.",
		"Cassandra":     "Wide-column distributed database designed for high-volume time-series and write-heavy workloads. Tunable consistency per query.",
		"Redis":         "In-memory key-value store with pub/sub, sorted sets, and streams. Used for caching, sessions, leaderboards, and real-time features.",
		"Memcached":     "Simple in-memory key-value cache. Fastest raw throughput for plain cache use cases. No persistence or complex data structures.",
		"ClickHouse":    "Columnar analytical database for real-time analytics queries over billions of rows. Best for dashboards and log aggregation.",
		"Elasticsearch": "Distributed full-text search and analytics engine. Inverted index for fast text search, aggregations, and geo queries.",
		"other":         "Custom or unlisted database. Generates a generic data-source scaffold for you to configure.",
	},

	"is_cache": {
		"no":  "Primary data store. Data persisted durably. Generates read/write connection helpers.",
		"yes": "Cache-only store. Data may be evicted. Generates cache client with TTL and eviction helpers. Listed as a cache alias in service configs.",
	},

	"ssl_mode": {
		"require":     "TLS required; server certificate not verified. Encrypts data in transit. Suitable for trusted private networks.",
		"disable":     "No TLS. Plaintext connection. Only use in local development or fully isolated networks.",
		"verify-ca":   "TLS required; certificate verified against a trusted CA. Protects against MITM from untrusted certificate authorities.",
		"verify-full": "TLS required; CA and hostname both verified. Strictest setting. Use in production.",
	},

	"consistency": {
		"strong":       "All reads reflect the latest write. Synchronous replication. Higher latency; no stale reads.",
		"eventual":     "Reads may briefly return stale data after a write. Lower latency, higher availability. Suitable for non-critical reads.",
		"LOCAL_QUORUM": "Cassandra: majority of local-DC replicas must acknowledge. Balances consistency and latency within a datacenter.",
		"ONE":          "Cassandra: single replica acknowledges. Fastest reads; lowest consistency.",
		"QUORUM":       "Cassandra: majority across all datacenters must acknowledge. Strong consistency at higher latency.",
		"ALL":          "Cassandra: all replicas must acknowledge. Strongest consistency; not fault-tolerant if any replica is down.",
		"LOCAL_ONE":    "Cassandra: single local-DC replica acknowledges. Fast local reads with minimal consistency guarantee.",
	},

	"replication": {
		"single-node":     "No replication. One database instance. Good for development and non-critical services.",
		"primary-replica": "One writer, one or more read replicas. Scales reads; async replication. Generates read/write connection routing.",
		"multi-region":    "Replicas in multiple geographic regions. Low-latency global reads; complex conflict resolution. Generates region-aware routing.",
	},

	// ── BACKEND: Architecture & Language ─────────────────────────────────────

	"arch": {
		"monolith":         "Single deployable unit. All features bundled in one codebase. Easiest to develop, test, and deploy. Best for MVPs and small teams.",
		"modular-monolith": "Bounded modules within a single deployment. Modules share a process but communicate through defined interfaces. A stepping-stone toward microservices.",
		"microservices":    "Independently deployable services communicating over APIs. Enables team autonomy and polyglot technology. Generates separate service directories with API contracts.",
		"event-driven":     "Services communicate asynchronously via a message broker. Decouples producers from consumers. Generates event publishers, consumers, and message schemas.",
		"hybrid":           "Mix of synchronous APIs and async event streams. Flexible for incremental migration or mixed workloads.",
	},

	"monolith_lang": {
		"Go":              "Compiled, statically typed, fast startup. Excellent concurrency via goroutines. Generates Go modules with idiomatic error handling.",
		"TypeScript/Node": "JavaScript superset with type safety. Large npm ecosystem. Generates Node services with TypeScript strict mode.",
		"Python":          "Readable, dynamically typed. Rich ML and data ecosystem. Generates Python services using FastAPI, Django, or Flask.",
		"Java":            "Mature, strongly typed JVM language. Enterprise ecosystem with Spring. Generates Spring Boot services.",
		"Kotlin":          "Modern JVM language with null safety and coroutines. Generates Ktor or Spring Boot Kotlin services.",
		"C#/.NET":         "Microsoft's type-safe language on .NET. Generates ASP.NET Core services with dependency injection.",
		"Rust":            "Memory-safe systems language. Zero-cost abstractions, no GC. Generates Actix-web or Axum services.",
		"Ruby":            "Developer-friendly, convention-over-configuration. Generates Rails or Sinatra applications.",
		"PHP":             "Widely deployed web language. Generates Laravel or Symfony applications.",
		"Elixir":          "Functional, fault-tolerant on the BEAM VM. Generates Phoenix applications with excellent real-time support.",
		"Other":           "Custom language choice. Generates a generic scaffold for you to extend.",
	},

	"language": {
		"Go":              "Compiled, statically typed, fast startup. Excellent concurrency via goroutines. Generates Go modules with idiomatic error handling.",
		"TypeScript/Node": "JavaScript superset with type safety. Large npm ecosystem. Generates Node services with TypeScript strict mode.",
		"Python":          "Readable, dynamically typed. Rich ML and data ecosystem. Generates Python services using FastAPI, Django, or Flask.",
		"Java":            "Mature, strongly typed JVM language. Enterprise ecosystem with Spring. Generates Spring Boot services.",
		"Kotlin":          "Modern JVM language with null safety and coroutines. Generates Ktor or Spring Boot Kotlin services.",
		"C#/.NET":         "Microsoft's type-safe language on .NET. Generates ASP.NET Core services with dependency injection.",
		"Rust":            "Memory-safe systems language. Zero-cost abstractions, no GC. Generates Actix-web or Axum services.",
		"Ruby":            "Developer-friendly, convention-over-configuration. Generates Rails or Sinatra applications.",
		"PHP":             "Widely deployed web language. Generates Laravel or Symfony applications.",
		"Elixir":          "Functional, fault-tolerant on the BEAM VM. Generates Phoenix applications with excellent real-time support.",
		"Other":           "Custom language choice. Generates a generic scaffold for you to extend.",
	},

	"pattern_tag": {
		"REST API":       "Standard HTTP resource-based API. Generates route handlers, middleware, and OpenAPI spec stubs.",
		"Worker":         "Background processing service. Consumes queue messages or runs scheduled jobs. No HTTP server.",
		"Event consumer": "Subscribes to message broker topics. Processes events with idempotent handlers and dead-letter support.",
		"Gateway":        "API gateway service. Routes, transforms, and authenticates requests before forwarding upstream.",
		"BFF":            "Backend-for-Frontend. Tailored API layer for a specific client. Aggregates multiple upstream calls.",
		"GraphQL API":    "GraphQL schema-first API. Generates resolvers, schema definitions, and DataLoader patterns.",
		"gRPC service":   "Protocol Buffer-based RPC service. Generates .proto files and gRPC server stubs.",
		"Hybrid":         "Multi-protocol service exposing both REST and gRPC or WebSocket endpoints.",
	},

	"error_format": {
		"RFC 7807 (Problem Details)": "Standard HTTP error body: type, title, status, detail, instance. Interoperable with any client. Generates Problem Details middleware and typed error constructors.",
		"Custom JSON envelope":       "App-specific error wrapper with code, message, and data fields. Generates shared error package with typed codes.",
		"Platform default":           "Uses the framework's built-in error handling. No additional wrapper generated.",
	},

	// ── BACKEND: Messaging ────────────────────────────────────────────────────

	"broker_tech": {
		"Kafka":             "High-throughput distributed event streaming. Durable log with consumer groups and topic partitioning. Best for high-volume event pipelines and stream processing.",
		"NATS":              "Lightweight, cloud-native messaging with pub/sub, queues, and JetStream persistence. Ultra-low latency. Best for microservice communication.",
		"RabbitMQ":          "Feature-rich message broker with routing, exchanges, and dead-letter queues. AMQP protocol. Best for complex routing and task queues.",
		"Redis Streams":     "Redis-native event streaming. Combines cache and message queue in one service. Best when Redis is already in the stack.",
		"AWS SQS/SNS":       "AWS-managed queues (SQS) and fanout topics (SNS). Zero infrastructure management. Generates AWS SDK producers and consumers.",
		"Google Pub/Sub":    "GCP-managed publish-subscribe messaging. Exactly-once or at-least-once delivery. Generates Pub/Sub client code.",
		"Azure Service Bus": "Azure-managed enterprise messaging with queues, topics, and sessions. Generates Azure SDK integration.",
		"Pulsar":            "Multi-tenant distributed messaging with built-in geo-replication. Supports both streaming and queuing semantics.",
	},

	// ── BACKEND: API Gateway ──────────────────────────────────────────────────

	"routing": {
		"Path-based":   "Routes requests by URL path prefix. Simple and predictable. Most common pattern for REST APIs.",
		"Header-based": "Routes requests based on HTTP header values (e.g. X-Service, Accept-Version). Useful for versioning and canary routing.",
		"Domain-based": "Routes requests based on hostname or subdomain. Generates virtual-host-style routing rules.",
	},

	// ── BACKEND: Auth ─────────────────────────────────────────────────────────

	"provider": {
		"Self-managed":  "Your own auth service. Full control over token format and session lifecycle. Generates JWT issuance, validation middleware, and refresh token storage.",
		"Auth0":         "Cloud-hosted identity platform. Generates Auth0 SDK integration, callback routes, and user profile sync.",
		"Clerk":         "Developer-focused auth-as-a-service with pre-built UI components. Generates Clerk SDK wrappers.",
		"Supabase Auth": "Open-source BaaS auth with row-level security. Generates Supabase client setup and auth helpers.",
		"Firebase Auth": "Google's mobile-first auth. Generates Firebase Admin SDK integration for token verification.",
		"Keycloak":      "Open-source identity and access management. Self-hosted or cloud. Generates OIDC/SAML configuration.",
		"AWS Cognito":   "AWS managed user pools. Generates Cognito SDK integration and JWT verification middleware.",
		"Other":         "Custom identity provider. Generates a generic OIDC/OAuth2 integration scaffold.",
	},

	"authz_model": {
		"RBAC":                     "Role-Based Access Control. Users get roles; roles have permissions. Simple and auditable. Generates role middleware and permissions table.",
		"ABAC":                     "Attribute-Based Access Control. Policies evaluate user, resource, and environment attributes. Flexible. Generates policy evaluation engine.",
		"ACL":                      "Access Control Lists. Each resource has an explicit list of allowed principals. Fine-grained. Generates ACL storage and check helpers.",
		"ReBAC":                    "Relationship-Based Access Control. Access determined by entity relationships. Google Zanzibar-style. Generates relationship tuples and check service.",
		"Policy-based (OPA/Cedar)": "Declarative policy engine. Policies are versioned code, testable independently. Generates OPA/Cedar policy files and evaluation middleware.",
		"Custom":                   "Hand-rolled authorization logic. Generates a placeholder authorization middleware for you to implement.",
	},

	"session_mgmt": {
		"Stateless (JWT only)":         "No server-side session state. All claims in the JWT. Scales horizontally without shared storage.",
		"Server-side sessions (Redis)": "Session data stored in Redis. Supports revocable tokens and centralized logout. Generates Redis session store.",
		"Database sessions":            "Session records in the primary database. Audit-friendly. Generates sessions table and DB session handler.",
		"None":                         "No session management. Service delegates auth entirely to an upstream gateway.",
	},

	"refresh_token": {
		"None":           "No refresh tokens. Short-lived access tokens only; users re-authenticate when they expire.",
		"Rotating":       "Each refresh token use issues a new token and invalidates the old one. Detects token theft via reuse detection.",
		"Non-rotating":   "A single long-lived refresh token reused until expiry. Simpler to implement.",
		"Sliding window": "Refresh token TTL resets on each use, expiring only after a period of inactivity.",
	},

	"mfa": {
		"None":              "No multi-factor authentication.",
		"TOTP":              "Time-based One-Time Password (Google Authenticator, Authy). Generates TOTP enrollment and QR code generation.",
		"SMS":               "One-time code via SMS. Generates SMS gateway integration (Twilio/Vonage).",
		"Email":             "One-time code via email. Generates email OTP flow.",
		"Passkeys/WebAuthn": "Passwordless auth using device biometrics or hardware keys. Generates WebAuthn ceremony endpoints.",
	},

	// ── BACKEND: Security ─────────────────────────────────────────────────────

	"waf_provider": {
		"Cloudflare WAF": "Cloudflare edge WAF with managed rules and custom policies. Blocks attacks before reaching your origin.",
		"AWS WAF":        "AWS-native WAF for ALB, API Gateway, and CloudFront. Generates WAF ACL rules and managed rule group associations.",
		"Cloud Armor":    "GCP DDoS protection and WAF for HTTP(S) load balancers. Generates security policy and pre-configured rules.",
		"Azure WAF":      "Azure WAF integrated with Application Gateway or Front Door. Generates OWASP ruleset configuration.",
		"ModSecurity":    "Open-source WAF module for Apache/Nginx. OWASP Core Rule Set support. Generates ModSecurity configuration files.",
		"NGINX ModSec":   "ModSecurity as an NGINX dynamic module. Generates NGINX config with ModSec integration and CRS rules.",
		"None":           "No WAF configured. Suitable for internal services or early-stage projects.",
	},

	"waf_ruleset": {
		"OWASP Core Rule Set": "Comprehensive community-maintained ruleset covering OWASP Top 10 vulnerabilities. Industry standard.",
		"Managed rules":       "Provider-managed rulesets automatically updated by the WAF vendor. Lower maintenance overhead.",
		"Custom":              "Hand-crafted rules for specific application requirements. Full control over what is blocked.",
		"None":                "No WAF ruleset. WAF is disabled or using provider defaults only.",
	},

	"captcha": {
		"hCaptcha":             "Privacy-focused CAPTCHA alternative to reCAPTCHA. Generates hCaptcha widget integration and server-side verification.",
		"reCAPTCHA v2":         "Google's checkbox or image-challenge CAPTCHA. Visible challenge. Generates reCAPTCHA v2 widget and verification.",
		"reCAPTCHA v3":         "Google's invisible CAPTCHA scoring users by behavior. No user challenge. Generates score-based risk assessment middleware.",
		"Cloudflare Turnstile": "Cloudflare's privacy-respecting CAPTCHA replacement. No user friction for most users. Generates Turnstile widget and token validation.",
		"None":                 "No CAPTCHA configured.",
	},

	"bot_protection": {
		"Cloudflare Bot Management": "Cloudflare's ML-based bot scoring at the edge. Blocks scrapers and credential stuffers before they reach the origin.",
		"Imperva":                   "Imperva's bot management platform with behavioral analysis and fingerprinting.",
		"DataDome":                  "Real-time bot and fraud protection with global threat intelligence.",
		"Custom":                    "Custom bot detection logic. Generates a middleware placeholder for implementing fingerprinting or behavioral analysis.",
		"None":                      "No bot protection configured.",
	},

	"rate_limit_strategy": {
		"Token bucket (Redis)": "Tokens replenish at a fixed rate; burst capacity allowed. Redis-backed for distributed enforcement.",
		"Sliding window":       "Counts requests in a rolling time window. Smoother than fixed window; prevents boundary bursts.",
		"Fixed window":         "Counts requests in discrete time windows. Simple; allows burst at window edges.",
		"Leaky bucket":         "Requests queued and processed at a fixed rate. Smooths traffic spikes.",
		"API Gateway":          "Delegates rate limiting to the upstream API gateway. No application-level code generated.",
		"None":                 "No rate limiting. Suitable when rate limiting is handled at the infrastructure layer.",
	},

	"rate_limit_backend": {
		"Redis":     "Redis-backed distributed rate limit counter. Consistent across multiple app instances.",
		"Memcached": "Memcached-backed counter. Fast but no persistence; counters reset on restart.",
		"In-memory": "Per-process in-memory counter. No coordination between instances. Only suitable for single-instance deployments.",
		"None":      "No rate limit storage. Rate limiting is disabled.",
	},

	"ddos_protection": {
		"CDN-level (Cloudflare)": "Cloudflare absorbs and filters DDoS traffic at the edge before it reaches your infrastructure.",
		"Provider-managed":       "Cloud provider's built-in DDoS protection (AWS Shield, Azure DDoS Protection, GCP Cloud Armor). Automatic mitigation.",
		"None":                   "No explicit DDoS protection beyond default network-level filtering.",
	},

	"internal_mtls": {
		"Enabled":  "Mutual TLS enforced between all internal services. Both client and server present certificates. Strong machine-to-machine auth. Generates TLS config and certificate provisioning.",
		"Disabled": "No internal mTLS. Services communicate over plain HTTP or one-way TLS internally.",
	},

	// ── DATA: Caching ─────────────────────────────────────────────────────────

	"layer": {
		"Application-level": "Cache managed in application memory or a shared in-process store. Low latency, no network hop. Best for single-instance or sticky-session deployments.",
		"Dedicated cache":   "Separate caching infrastructure (Redis, Memcached). Shared across service instances. Generates cache client setup and key namespacing.",
		"CDN":               "Content Delivery Network caches at edge nodes. Reduces origin load for static or semi-static content. Generates cache-control header helpers.",
		"None":              "No caching layer. All reads go directly to the primary data store.",
	},

	"invalidation": {
		"TTL-based":    "Cache entries expire after a fixed time-to-live. Simple and predictable. Stale data possible near expiry.",
		"Event-driven": "Cache invalidated by domain events (entity updated). Strongly consistent. Generates event handlers that delete or refresh cache keys on write.",
		"Manual":       "Application code explicitly evicts cache entries on mutation. Full control. Generates cache manager with typed delete methods.",
		"Hybrid":       "Combines TTL with event-driven invalidation. TTL as safety net; events for immediate consistency.",
	},

	// ── DATA: Governance ──────────────────────────────────────────────────────

	"delete_strategy": {
		"Soft-delete":           "Records marked deleted but kept in the database. Supports audit trails. Generates deleted_at column, soft-delete scope, and restore endpoint.",
		"Hard-delete":           "Records permanently removed. Simplest approach; no cleanup needed. Generates standard DELETE queries.",
		"Archival":              "Records moved to an archive table before deletion. Balances auditability with hot-table performance. Generates archival job and archive schema.",
		"Soft + periodic purge": "Records soft-deleted first, then hard-deleted after a retention period by a scheduled job. Compliance-friendly.",
	},

	"pii_encryption": {
		"Field-level AES-256":      "Sensitive columns encrypted individually using AES-256. Only targeted fields protected; others remain queryable. Generates encryption helpers and migration.",
		"Full database encryption": "Entire database volume encrypted at rest (TDE). Managed by the database engine or cloud provider. No application-level changes.",
		"Application-level":        "Data encrypted by the application before being written. Full control over key management. Generates encryption service with pluggable key provider.",
		"None":                     "No additional PII encryption beyond storage-at-rest defaults.",
	},

	"data_residency": {
		"US":      "All data stored and processed in US regions. Required for US government and many domestic compliance frameworks.",
		"EU":      "All data stored and processed in EU regions. Required for GDPR data-residency obligations.",
		"APAC":    "All data stored and processed in Asia-Pacific regions. Meets local data-sovereignty requirements.",
		"US + EU": "Data mirrored across US and EU regions with region-specific access controls.",
		"Global":  "No geographic restriction. Data distributed globally for latency optimization.",
		"Custom":  "Custom residency requirements. Generates placeholders for you to specify region constraints.",
	},
}

// sectionPanels holds overview text for each main section tab shown when no
// specific field description is active.
var sectionPanels = []string{
	// 0 — Description
	"Describe your project in natural language.\n\nThis free-text overview is passed to every code generation agent as context, helping them understand the purpose, audience, and scope of what is being built.\n\nFill this in before the structured pillars for best results.",

	// 1 — Backend
	"Define the server-side architecture and services.\n\nChoose an architecture pattern, add service units with languages and frameworks, configure communication links, set up auth strategy and roles, define security policies, and configure background jobs.\n\nBackend roles defined here are available for endpoint and page access control in later pillars.",

	// 2 — Data
	"Model your data layer.\n\nAdd databases with their type, hosting, and replication strategy. Define domain entities with attributes and relationships. Configure caching layers and file storage. Set governance policies for retention, deletion, encryption, and compliance.",

	// 3 — Contracts
	"Define the API surface between services and clients.\n\nCreate DTOs (data transfer objects) with protocol-specific schemas. Define endpoints with auth, pagination, and rate limits. Configure API versioning and deprecation strategy. Add external API integrations.",

	// 4 — Frontend
	"Specify the client-side application.\n\nChoose language, platform, framework, and meta-framework. Configure the design system (theme, spacing, motion, vibe). Define pages with routes, auth requirements, and components. Set up navigation, i18n, accessibility, and SEO.",

	// 5 — Infrastructure
	"Describe the deployment and operations setup.\n\nConfigure networking (DNS, TLS, CDN, CORS). Define CI/CD pipelines, container registry, IaC tools, and deployment strategy. Set up observability (logging, metrics, tracing, alerting). Define named server environments.",

	// 6 — Cross-Cutting
	"Configure quality, testing, and development standards.\n\nSelect testing frameworks for unit, integration, E2E, API, load, and contract tests filtered to your technology choices. Configure documentation formats and auto-generation. Set dependency update automation and feature flag tooling.",

	// 7 — Realize
	"Configure the code generation pipeline.\n\nSet the application name and output directory. Choose the global LLM model and provider for generation. Configure concurrency, verification, and dry-run mode. Override the model per pillar for fine-grained control.",
}

// GetOptionDescription returns the description for a field key + option value pair.
// Returns "" when no description is registered for this combination.
func GetOptionDescription(fieldKey, value string) string {
	opts, ok := fieldDescriptions[fieldKey]
	if !ok {
		return ""
	}
	// Exact match first.
	if d, ok := opts[value]; ok {
		return d
	}
	// Case-insensitive fallback.
	lower := strings.ToLower(value)
	for k, d := range opts {
		if strings.ToLower(k) == lower {
			return d
		}
	}
	return ""
}

// FormatDescriptionPanel formats a field label, current value, and description
// into a []string panel for withDescriptionPanel(). Lines contain ANSI styles.
func FormatDescriptionPanel(label, value, desc string, w, h int) []string {
	if w < 10 {
		return nil
	}
	inner := w - 2 // 1-char left margin + 1-char right margin

	styleTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrYellow)).
		Bold(true)
	styleValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)
	styleDim := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFgDim))
	styleBody := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))

	var lines []string
	add := func(s string) { lines = append(lines, s) }
	blank := func() { add("") }

	blank()
	blank()

	// Field label as title
	titleText := strings.ToUpper(label)
	if len([]rune(titleText)) > inner {
		titleText = string([]rune(titleText)[:inner])
	}
	add(" " + styleTitle.Render(titleText))
	blank()

	// Currently selected value
	valLine := "  " + value
	if len([]rune(valLine)) > inner {
		runes := []rune(valLine)
		valLine = string(runes[:inner-1]) + "…"
	}
	add(" " + styleValue.Render(valLine))
	blank()

	// Separator
	sep := strings.Repeat("─", inner-1)
	add(" " + styleDim.Render(sep))
	blank()

	// Description — word-wrapped to inner width
	wrapped := wordWrap(desc, inner-2)
	for _, wl := range wrapped {
		add(" " + styleBody.Render(wl))
	}

	blank()

	if h > 0 && len(lines) > h {
		lines = lines[:h]
	}

	return lines
}

// FormatSectionPanel returns a panel shown when no specific field description
// is active, giving an overview of the current section.
func FormatSectionPanel(sectionIdx, w, h int) []string {
	if w < 10 || sectionIdx < 0 || sectionIdx >= len(sectionPanels) {
		return nil
	}
	inner := w - 2

	styleTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrMagenta)).
		Bold(true)
	styleDim := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFgDim))
	styleBody := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFgDim))

	sectionNames := []string{
		"DESCRIPTION", "BACKEND", "DATA", "CONTRACTS",
		"FRONTEND", "INFRASTRUCTURE", "CROSS-CUTTING", "REALIZE",
	}

	var lines []string
	add := func(s string) { lines = append(lines, s) }
	blank := func() { add("") }

	blank()
	blank()

	name := ""
	if sectionIdx < len(sectionNames) {
		name = sectionNames[sectionIdx]
	}
	add(" " + styleTitle.Render(name))
	blank()

	sep := strings.Repeat("─", inner-1)
	add(" " + styleDim.Render(sep))
	blank()

	text := sectionPanels[sectionIdx]
	paragraphs := strings.Split(text, "\n\n")
	for pi, para := range paragraphs {
		wrapped := wordWrap(strings.ReplaceAll(para, "\n", " "), inner-2)
		for _, wl := range wrapped {
			add(" " + styleBody.Render(wl))
		}
		if pi < len(paragraphs)-1 {
			blank()
		}
	}

	blank()

	if h > 0 && len(lines) > h {
		lines = lines[:h]
	}

	return lines
}

// wordWrap splits text into lines of at most maxW runes, breaking at word boundaries.
func wordWrap(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, word := range words {
		cleaned := strings.Map(func(r rune) rune {
			if unicode.IsPrint(r) {
				return r
			}
			return -1
		}, word)
		if cleaned == "" {
			continue
		}
		if current == "" {
			current = cleaned
			continue
		}
		candidate := current + " " + cleaned
		if len([]rune(candidate)) <= maxW {
			current = candidate
		} else {
			lines = append(lines, current)
			current = cleaned
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
