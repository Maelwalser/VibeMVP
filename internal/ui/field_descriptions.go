package ui

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

	// ── CONTRACTS: DTOs ───────────────────────────────────────────────────────

	"category": {
		"Request":       "Inbound payload DTO. Represents data the client sends to the API. Generates validation annotations and request binding.",
		"Response":      "Outbound payload DTO. Represents data the API returns to clients. Generates serialization code.",
		"Event Payload": "Message payload for async event systems. Generates schema definitions compatible with the selected serialization format.",
		"Shared/Common": "Reusable DTO referenced by multiple requests or responses. Generates a shared types module.",
	},

	"protocol": {
		"REST/JSON":         "JSON over HTTP. Default for web APIs. Generates JSON struct tags and OpenAPI schema definitions.",
		"Protobuf":          "Binary Protocol Buffer encoding. Compact and fast. Generates .proto message definitions and compiled stubs.",
		"Avro":              "Binary Avro encoding. Schema registered in a schema registry. Generates Avro schema files and registry-aware serializer.",
		"MessagePack":       "Binary MessagePack encoding. JSON-compatible but more compact. Generates MessagePack codec wrappers.",
		"Thrift":            "Apache Thrift binary encoding. Multi-language RPC and serialization. Generates .thrift IDL files and language stubs.",
		"FlatBuffers":       "Zero-copy binary encoding. Extremely fast deserialization. Generates FlatBuffers schema and accessor code.",
		"Cap'n Proto":       "Zero-copy, schema-based binary encoding. No parse step needed. Generates Cap'n Proto schema and bindings.",
		"REST":              "Synchronous HTTP endpoint. Generates HTTP handler with method, path, and request/response types.",
		"GraphQL":           "Query language for APIs. Clients specify exact data needs. Generates resolver stubs and GraphQL schema types.",
		"gRPC":              "High-performance RPC over HTTP/2. Strong typing via Protobuf. Generates .proto service definitions and gRPC stubs.",
		"WebSocket message": "Bidirectional persistent connection. Real-time push and receive. Generates WebSocket handler with message dispatch.",
		"Event":             "Async message via a broker (Kafka, RabbitMQ). Decoupled delivery. Generates producer/consumer stubs and message schemas.",
		"WebSocket":         "Persistent bidirectional connection. Real-time server-push. Generates WebSocket upgrade handler.",
		"Webhook":           "Server pushes events to a registered client URL. Generates webhook dispatcher and HMAC signature validation.",
		"SOAP":              "XML-based web service protocol. Generates WSDL definition and SOAP envelope binding code.",
	},

	"http_method": {
		"GET":    "Retrieve a resource. Idempotent and safe. Should not modify state.",
		"POST":   "Create a new resource or submit data. Not idempotent. Generates handler that persists a new record.",
		"PUT":    "Replace an entire resource. Idempotent. Generates handler that overwrites all fields of an existing record.",
		"PATCH":  "Partially update a resource. Generates handler that merges provided fields.",
		"DELETE": "Remove a resource. Idempotent. Generates handler that deletes or soft-deletes a record by ID.",
	},

	"graphql_op_type": {
		"Query":        "Read-only data fetch. No side effects. Generates resolver that reads data.",
		"Mutation":     "Data modification operation. Creates, updates, or deletes. Generates resolver with input validation and persistence.",
		"Subscription": "Real-time data stream over WebSocket. Generates subscription resolver with event source wiring.",
	},

	"grpc_stream_type": {
		"Unary":            "Single request, single response. Standard function call semantics. Default for most gRPC methods.",
		"Server stream":    "Single request, stream of responses from the server. Good for live feeds and large datasets.",
		"Client stream":    "Stream of requests from the client, single aggregated response. Good for batch uploads.",
		"Bidirectional":    "Both client and server stream simultaneously. Full-duplex. Used for chat and live collaboration.",
		"Server streaming": "Single request, stream of responses. Good for progress notifications.",
		"Client streaming": "Stream of requests, single response. Good for batch uploads or sensor data ingestion.",
	},

	"ws_direction": {
		"Client→Server": "Messages flow from client to server only. Server processes incoming events.",
		"Server→Client": "Messages pushed from server to client only. Server-side broadcast or live updates.",
		"Bidirectional": "Both client and server send messages freely. Used for chat, collaborative editing, and real-time games.",
		"Send":          "This endpoint sends messages to connected clients.",
		"Receive":       "This endpoint receives messages from clients.",
	},

	"pagination": {
		"Cursor-based": "Opaque cursor pointing to a result-set position. Consistent results during mutations. Best for real-time data.",
		"Offset/limit": "Standard page number and size. Simple to implement. Results may shift if data changes between pages.",
		"Keyset":       "Paginate by the last seen primary key. Efficient for large tables; no offset scanning.",
		"Page number":  "Client specifies page number and page size. Familiar UX. Same caveats as offset/limit.",
		"None":         "No pagination. Returns all results. Only suitable for small bounded result sets.",
	},

	"rate_limit": {
		"Default (global)": "Uses the API gateway's global rate limit policy. No per-endpoint override.",
		"Strict":           "Lower limit than default. Extra protection for sensitive or expensive endpoints.",
		"Relaxed":          "Higher limit than default. For trusted clients or batch endpoints.",
		"None":             "No rate limiting on this endpoint.",
	},

	"deprecation": {
		"None":                     "No deprecation notice. This API version is current.",
		"Sunset header":            "HTTP Sunset header added to responses announcing the removal date. RFC 8594 standard.",
		"Versioned removal notice": "Deprecation documented in API changelog with a specific removal version.",
		"Changelog entry":          "Deprecation noted in the project CHANGELOG only. No runtime header.",
		"Custom":                   "Custom deprecation strategy. Generates a placeholder for your deprecation logic.",
	},

	"tls_mode": {
		"TLS":      "One-way TLS. Server presents certificate; client verifies it. Encrypts traffic.",
		"mTLS":     "Mutual TLS. Both client and server present certificates. Strong machine-to-machine authentication.",
		"Insecure": "No TLS. Plaintext connection. Only use in local development or fully trusted internal networks.",
	},

	"soap_version": {
		"1.1": "SOAP 1.1 (HTTP+XML). Older standard; widely supported by legacy systems.",
		"1.2": "SOAP 1.2. Stricter, better defined semantics. Recommended for new SOAP integrations.",
	},

	"auth_mechanism": {
		"API Key": "Static secret key sent in a header or query parameter. Generates API key validation middleware.",
		"OAuth2":  "OAuth 2.0 token-based auth. Generates OAuth2 client with token refresh logic.",
		"Bearer":  "JWT or opaque bearer token in the Authorization header. Generates token validation middleware.",
		"Basic":   "Base64-encoded username:password. Only use over HTTPS. Generates basic auth decoder.",
		"mTLS":    "Mutual TLS certificate authentication. Generates TLS config with client cert verification.",
		"None":    "No authentication on this external API. Suitable for public APIs.",
	},

	"failure_strategy": {
		"Retry 3x":           "Retry failed requests up to three times with exponential backoff.",
		"Retry 5x":           "Retry failed requests up to five times. More aggressive recovery.",
		"Immediate fail":     "No retries. Return error on first failure. Application handles fallback.",
		"None":               "No explicit failure strategy. Framework defaults apply.",
		"Circuit breaker":    "Opens circuit after failure threshold, preventing further calls until it resets.",
		"Fallback":           "On failure, return a cached response or default value.",
		"Retry with backoff": "Retry with increasing delay between attempts. Reduces pressure on a struggling service.",
		"Timeout":            "Fail after a configured timeout. Prevents slow external calls from blocking the application.",
		"Timeout + fail":     "Apply a timeout; on expiry fail immediately without retries.",
	},

	// ── FRONTEND: Tech ────────────────────────────────────────────────────────

	"platform": {
		// Frontend platforms
		"Web":     "Browser-based application delivered over HTTP. Generates a web project with HTML/CSS/JS output.",
		"Mobile":  "Native or cross-platform mobile application for iOS and/or Android.",
		"Desktop": "Native desktop application for macOS, Windows, or Linux.",
		"Hybrid":  "Targets multiple platforms from a single codebase.",
		// CI/CD platforms
		"GitHub Actions": "GitHub's native CI/CD. Tight repository integration. Generates workflow YAML files.",
		"GitLab CI":      "GitLab's built-in CI/CD with pipeline YAML. Generates .gitlab-ci.yml.",
		"Jenkins":        "Self-hosted CI/CD. Highly extensible via plugins. Generates Jenkinsfile.",
		"CircleCI":       "Cloud CI/CD with fast caching. Generates .circleci/config.yml.",
		"ArgoCD":         "GitOps continuous delivery for Kubernetes. Generates ArgoCD Application manifests.",
		"Tekton":         "Kubernetes-native CI/CD pipelines. Generates Tekton Pipeline and Task manifests.",
	},

	"meta_framework": {
		"Next.js":           "React meta-framework. SSR, SSG, ISR, and file-based routing. Generates app router, API routes, and deployment config.",
		"Nuxt":              "Vue meta-framework. SSR, SSG, and file-based routing. Generates Nuxt config, composables, and server routes.",
		"SvelteKit":         "Svelte meta-framework. SSR and file-based routing with minimal JS. Generates SvelteKit routes and server-load functions.",
		"Remix":             "React meta-framework focused on web fundamentals. Nested routing and form actions. Generates loaders and actions.",
		"Astro":             "Islands architecture for content-heavy sites. Minimal client JS. Generates Astro pages with component islands.",
		"TanStack Start":    "React meta-framework from the TanStack team. Type-safe routing and server functions.",
		"Angular Universal": "Server-side rendering for Angular. Generates SSR server and TransferState setup.",
		"None":              "No meta-framework. Pure client-side rendering. Generates a SPA.",
	},

	"pkg_manager": {
		"npm":  "Node's built-in package manager. Widest compatibility; default for most projects.",
		"yarn": "Facebook's package manager. Faster installs with lockfile determinism. Workspaces support.",
		"pnpm": "Disk-efficient package manager using a content-addressable store. Strictest dependency resolution.",
		"bun":  "All-in-one JS runtime and package manager. Fastest installs. Generates bun.lockb and bun-compatible scripts.",
	},

	"styling": {
		"Tailwind CSS":      "Utility-first CSS framework. No custom CSS needed for most UIs. Generates Tailwind config and purge settings.",
		"CSS Modules":       "Locally scoped CSS. Each component has its own CSS file; classes are hashed. Generates module CSS files.",
		"Styled Components": "CSS-in-JS with tagged template literals. Dynamic styles based on props. Generates StyledComponent definitions.",
		"Sass/SCSS":         "CSS superset with variables, nesting, and mixins. Generates SCSS files with shared variables and partials.",
		"Vanilla CSS":       "Plain CSS. No framework or preprocessor. Full control; no added abstraction.",
		"UnoCSS":            "Atomic CSS engine. Faster than Tailwind with more flexibility. Generates UnoCSS preset config.",
	},

	"component_lib": {
		"shadcn/ui":   "Unstyled, copy-paste components built on Radix UI. Full ownership of component code. Tailwind-based styling.",
		"Radix":       "Unstyled accessible primitives. Composable headless components for building custom design systems.",
		"Material UI": "Google Material Design React components. Comprehensive, opinionated. Generates MUI theme and component imports.",
		"Ant Design":  "Enterprise-grade React component library. Rich data table, form, and layout components.",
		"Headless UI": "Tailwind Labs headless components. Integrates with Tailwind CSS. Accessible by default.",
		"DaisyUI":     "Tailwind CSS component library with semantic class names. Generates DaisyUI theme config.",
		"None":        "No component library. Build UI components from scratch.",
		"Custom":      "Custom component library. Generates a component folder scaffold.",
	},

	"state_mgmt": {
		"Redux Toolkit": "Opinionated Redux with reducers, actions, and thunks. Best for large apps with complex shared state.",
		"Zustand":       "Lightweight state management with hooks. Minimal boilerplate. Generates typed stores.",
		"Pinia":         "Vue's official state management. Composition API-based. Generates typed stores with actions and getters.",
		"MobX":          "Observable-based reactive state. Automatically tracks dependencies. Generates observable stores.",
		"Jotai":         "Atomic state management for React. Each atom is a unit of state. Generates typed atoms.",
		"Valtio":        "Proxy-based state with automatic re-renders. Minimal API. Generates state objects with snapshot reads.",
		"Context API":   "React's built-in context. Simple; no extra dependencies. Suitable for low-frequency state updates.",
		"XState":        "Finite state machine library. Explicit states and transitions. Generates machine definitions.",
		"NgRx":          "Redux-inspired state management for Angular. Actions, reducers, effects, and selectors.",
		"None":          "No dedicated state management library. Component-local state only.",
	},

	"data_fetching": {
		"TanStack Query": "Async state management for server data. Caching, refetching, pagination, and optimistic updates.",
		"SWR":            "React hook for data fetching with stale-while-revalidate. Lightweight, automatic revalidation.",
		"Apollo Client":  "Feature-rich GraphQL client with normalized cache.",
		"Urql":           "Lightweight GraphQL client. Composable and extensible.",
		"RTK Query":      "Data-fetching layer built into Redux Toolkit. Cache invalidation tied to Redux state.",
		"Fetch API":      "Native browser fetch. No extra dependencies. Generates typed fetch wrappers.",
		"Axios":          "Promise-based HTTP client with interceptors. Generates Axios instance with base config.",
		"Vue Query":      "TanStack Query for Vue. Generates useQuery and useMutation composables.",
		"None":           "No data-fetching library. Raw fetch or XHR used directly.",
	},

	"form_handling": {
		"React Hook Form": "Performant, flexible form library using uncontrolled inputs. Minimal re-renders. Generates form hooks and validation integration.",
		"Formik":          "Form state management with Yup validation integration. Higher-level API than React Hook Form.",
		"Zod + native":    "Form state managed by hand; validation via Zod schemas. No form library dependency.",
		"Vee-Validate":    "Vue form validation library with composition API support. Generates typed form validation composables.",
		"None":            "No form handling library. Forms managed with plain state.",
	},

	"validation": {
		"Zod":             "TypeScript-first schema validation with type inference. Generates Zod schemas used for both runtime validation and TypeScript types.",
		"Yup":             "Schema-based object validation. Widely used with Formik. Generates Yup schema definitions.",
		"Valibot":         "Modular, tree-shakeable validation library. Smaller bundle than Zod. Generates Valibot schemas.",
		"Joi":             "Powerful validation for JavaScript objects. Battle-tested. Generates Joi schema definitions.",
		"Class-validator": "TypeScript decorator-based validation for classes. Best with NestJS or Angular. Generates decorated DTO classes.",
		"None":            "No validation library. Validation implemented manually.",
	},

	"realtime": {
		"WebSocket": "Persistent bidirectional connection. Low latency, true push. Generates WebSocket client with reconnect logic.",
		"SSE":       "Server-Sent Events. Server pushes; client reads. Simpler than WebSocket for one-way streams.",
		"Polling":   "Periodic HTTP requests to check for updates. Simple; no persistent connection.",
		"None":      "No real-time data channel. Data refreshed on explicit user action only.",
	},

	"auth_flow": {
		"Redirect (OAuth/OIDC)": "Redirect user to identity provider login page. Standard OAuth2/OIDC PKCE flow. Generates auth redirect handler, callback route, and token storage.",
		"Modal login":           "Login form shown in an overlay without leaving the current page.",
		"Magic link":            "Passwordless login via a one-time link emailed to the user.",
		"Passwordless":          "Login via OTP (email or SMS) without a password.",
		"Social only":           "Login exclusively via social providers (Google, GitHub). No username/password.",
	},

	"pwa_support": {
		"None":                              "No PWA features. Standard web application.",
		"Basic (manifest + service worker)": "Web app manifest and basic service worker for install prompt and offline shell. Generates manifest.json and SW registration.",
		"Full offline":                      "Comprehensive offline-first PWA. Generates service worker with cache strategies (Workbox) for full offline functionality.",
		"Push notifications":                "Service worker with Web Push API. Generates push subscription management and notification display logic.",
	},

	"image_opt": {
		"Next/Image (built-in)": "Next.js Image component with automatic resizing, lazy loading, and WebP conversion. Zero extra cost for Next.js projects.",
		"Cloudinary":            "Cloud-based image and video management. Generates Cloudinary SDK integration with transformation URL helpers.",
		"Imgix":                 "Real-time image processing CDN. URL-based transformations. Generates Imgix URL builder helpers.",
		"Sharp (self-hosted)":   "Node.js image processing library for server-side resizing and format conversion. Generates Sharp transform pipeline.",
		"CDN transform":         "Use CDN-native image transformation (Cloudflare Images, BunnyCDN). No application-level code needed.",
		"None":                  "No image optimization. Images served as-is.",
	},

	"error_boundary": {
		"React Error Boundary": "React class component or react-error-boundary library. Catches rendering errors and shows fallback UI.",
		"Global try-catch":     "Top-level error handlers (window.onerror, unhandledrejection). Catches async errors outside the React tree.",
		"Framework default":    "Use the meta-framework's built-in error handling (Next.js error.tsx, SvelteKit +error.svelte).",
		"Custom":               "Custom error boundary implementation. Generates a typed ErrorBoundary component scaffold.",
	},

	"bundle_opt": {
		"Code splitting (route-based)": "JavaScript split into per-route chunks loaded lazily. Fastest initial load. Generates route-based dynamic import configuration.",
		"Dynamic imports":              "On-demand loading of modules at runtime. Generates import() call patterns for heavy components.",
		"Tree shaking only":            "Dead code eliminated at build time. No lazy loading. Generates build config with side-effect annotations.",
		"None":                         "No bundle optimization. Single bundle output. Suitable for internal tools.",
	},

	// ── FRONTEND: Theme ───────────────────────────────────────────────────────

	"dark_mode": {
		"None":                     "Light-only interface. No dark mode support.",
		"Toggle (user preference)": "User can switch between light and dark mode. Preference persisted in localStorage. Generates toggle component and CSS variable switching.",
		"System preference":        "Respects the OS dark mode setting via prefers-color-scheme. No manual toggle.",
		"Dark only":                "Dark-only interface. No light mode fallback.",
	},

	"border_radius": {
		"Sharp (0)":     "Zero border radius. Hard-edged UI. Technical, developer-tool aesthetic.",
		"Subtle (4px)":  "Very slight rounding. Softens edges without looking rounded. Neutral; works across most design systems.",
		"Rounded (8px)": "Moderate rounding. Friendly and approachable. Popular in consumer SaaS.",
		"Pill (999px)":  "Fully rounded buttons and badges. Playful, modern aesthetic.",
		"Custom":        "Custom border-radius value. Specify your own design token.",
	},

	"spacing": {
		"Compact":     "Tighter spacing scale. Fits more content on screen. Common in data-dense dashboards.",
		"Comfortable": "Balanced spacing. Default for most products.",
		"Spacious":    "Generous whitespace. Content breathes. Common in marketing and editorial layouts.",
	},

	"elevation": {
		"Flat":      "No shadows or depth. Borders separate elements. Clean, minimal aesthetic.",
		"Subtle":    "Soft shadows for depth cues. Subtle layer separation without heavy drop shadows.",
		"Prominent": "Strong shadows and depth. Cards and modals clearly float above the background.",
	},

	"motion": {
		"None":                   "No animations or transitions. Fastest perceived performance. Best for reduced-motion accessibility.",
		"Subtle transitions":     "Gentle opacity and translate transitions. Polished without distraction. 150-200ms ease transitions.",
		"Animated (spring/ease)": "Rich spring-physics or easing animations. Expressive interactions. Generates animation library config.",
	},

	"vibe": {
		"Professional": "Clean, corporate aesthetic. Neutral palette. High information density. Suited for enterprise SaaS.",
		"Friendly":     "Warm colors, rounded elements, approachable typography. Suited for consumer apps.",
		"Playful":      "Bold colors, expressive animations, personality-forward. Suited for consumer products.",
		"Minimal":      "Extensive whitespace, limited color, restrained typography. Suited for content sites.",
		"Technical":    "Dark background, monospace accents, data-dense layout. Developer-facing tools.",
		"Custom":       "Custom vibe. Describe your design intent in the description field.",
	},

	"font": {
		"Inter":   "Highly legible geometric sans-serif. Excellent screen rendering. Default for many design systems.",
		"Geist":   "Vercel's clean sans-serif. Optimized for developer tools and dashboards.",
		"DM Sans": "Rounded, friendly geometric sans-serif. Warm and modern.",
		"System":  "Platform system font stack. Zero download. Fastest loading.",
		"Custom":  "Custom font selection. Specify font-family, weights, and load strategy.",
	},

	// ── FRONTEND: Analytics ───────────────────────────────────────────────────

	"analytics": {
		"PostHog":            "Open-source product analytics with feature flags, session recording, and funnel analysis.",
		"Google Analytics 4": "Google's event-based analytics platform. Deep integration with Google Ads. Generates GA4 gtag setup.",
		"Plausible":          "Privacy-first, lightweight analytics. GDPR-compliant. No cookies. Generates script snippet.",
		"Mixpanel":           "Event-based user analytics with cohort analysis. Generates Mixpanel SDK init and event helpers.",
		"Segment":            "Customer data platform. Routes events to multiple analytics destinations. Generates Segment analytics.js setup.",
		"Custom":             "Custom analytics integration. Generates typed event tracking helpers.",
		"None":               "No analytics configured.",
	},

	"telemetry": {
		"Sentry":            "Error tracking and performance monitoring. Captures exceptions with full stack traces and context. Generates Sentry SDK init.",
		"Datadog RUM":       "Real User Monitoring from Datadog. Tracks page load, errors, and user interactions. Generates Datadog RUM init.",
		"LogRocket":         "Session replay with error tracking. Shows exactly what users experienced when errors occurred.",
		"New Relic Browser": "Full-stack observability with browser agent. Tracks page performance and JS errors.",
		"Custom":            "Custom frontend error tracking. Generates typed error reporter scaffold.",
		"None":              "No frontend RUM or error tracking.",
	},

	// ── FRONTEND: Navigation ──────────────────────────────────────────────────

	"nav_type": {
		"Top bar":          "Horizontal navigation at the top of the viewport. Works well for apps with 5-10 top-level sections.",
		"Sidebar":          "Vertical navigation panel on the left. Scales to many sections. Common in dashboards.",
		"Bottom tabs":      "Tab bar at the bottom. Mobile-native pattern. Thumb-friendly on phones.",
		"Hamburger menu":   "Collapsed navigation behind a hamburger icon. Saves space on small screens.",
		"Breadcrumbs only": "Navigation expressed as a breadcrumb trail. No global nav. For deeply hierarchical content.",
		"None":             "No navigation component generated.",
	},

	"auth_aware": {
		"true":  "Navigation dynamically shows or hides items based on authentication state. Generates auth-aware nav guards.",
		"false": "Navigation is static. Auth state not reflected in nav items.",
	},

	"breadcrumbs": {
		"true":  "Breadcrumb trail generated above page content. Shows hierarchical path. Generates breadcrumb component.",
		"false": "No breadcrumbs.",
	},

	// ── FRONTEND: A11y/SEO ────────────────────────────────────────────────────

	"wcag_level": {
		"A":    "Minimum WCAG compliance. Basic keyboard navigation and alt-text. Required for most public-sector sites.",
		"AA":   "Standard WCAG level. Color contrast, resize, and focus indicators. Recommended baseline for all products.",
		"AAA":  "Highest WCAG compliance. Extended audio description, sign language. Difficult to achieve fully.",
		"None": "No WCAG compliance target. Accessibility handled manually.",
	},

	"seo_render_strategy": {
		"SSR":       "Server-Side Rendering. HTML generated on each request. Fresh content, crawlable. Generates SSR entry points.",
		"SSG":       "Static Site Generation. HTML pre-built at deploy time. Fastest load; no server needed. Generates static export config.",
		"ISR":       "Incremental Static Regeneration. Static pages rebuilt in background after revalidation period. Next.js-specific.",
		"Prerender": "Pre-renders specific routes to static HTML via a headless browser. Generates prerender configuration.",
		"None":      "No server-side or pre-rendering. Client-side SPA only.",
	},

	"sitemap": {
		"true":  "Generates a sitemap.xml for search engine crawling. Configures sitemap route and robots.txt.",
		"false": "No sitemap generated.",
	},

	"meta_tag_injection": {
		"true":  "Dynamic meta tags (title, description, og:image) injected per page. Generates meta tag management helpers.",
		"false": "Static meta tags only. No dynamic per-page meta injection.",
	},

	// ── FRONTEND: i18n ────────────────────────────────────────────────────────

	"translation_strategy": {
		"i18n library": "Dedicated i18n library (next-intl, vue-i18n, i18next). Generates library config, translation files, and locale switching.",
		"Static files": "JSON translation files loaded statically. Simple; no runtime dependency on i18n library.",
		"CDN":          "Translation strings fetched from a CDN or localization platform (Lokalise, Phrase). Generates fetch-on-load logic.",
		"Custom":       "Custom translation mechanism. Generates a typed translation function scaffold.",
	},

	"timezone_handling": {
		"Server-side":     "Timestamps stored and formatted on the server. Consistent across all clients.",
		"Client-side":     "Timestamps formatted in the user's local timezone by the browser.",
		"Both":            "Server normalizes to UTC; client formats to local timezone. Best of both worlds.",
		"UTC always":      "All dates displayed in UTC. No timezone conversion. Suitable for developer tools.",
		"User preference": "User sets their preferred timezone in profile settings.",
	},

	// ── INFRASTRUCTURE: Networking ────────────────────────────────────────────

	"dns_provider": {
		"Cloudflare": "DNS with DDoS protection and edge caching. Generates Terraform Cloudflare DNS resources.",
		"Route53":    "AWS Route 53. Tight integration with AWS services. Generates Terraform aws_route53_zone resources.",
		"Cloud DNS":  "Google Cloud DNS. Generates Terraform google_dns_managed_zone resources.",
		"Azure DNS":  "Azure DNS. Generates Terraform azurerm_dns_zone resources.",
		"Other":      "Custom DNS provider. Generates placeholder DNS configuration.",
	},

	"tls_ssl": {
		"Let's Encrypt": "Free automated TLS certificates via ACME. Generates cert-manager or Caddy configuration.",
		"Cloudflare":    "Cloudflare-managed TLS. Generates Cloudflare SSL configuration and origin certificate.",
		"ACM":           "AWS Certificate Manager. Free TLS for AWS resources. Generates Terraform aws_acm_certificate resources.",
		"Manual":        "Manually managed certificates. Generates TLS secret mounts and renewal reminder configuration.",
		"None (dev)":    "No TLS. HTTP only. For local development only.",
	},

	"cdn": {
		"Cloudflare CDN": "Cloudflare's global edge network. Caches static assets. Generates Cloudflare page rules and cache config.",
		"AWS CloudFront": "AWS CDN. Tight S3 and ALB integration. Generates Terraform CloudFront distribution resources.",
		"GCP Cloud CDN":  "GCP CDN for Load Balancer backends. Generates Terraform backend service with CDN policy.",
		"Azure CDN":      "Azure CDN profiles. Generates Terraform azurerm_cdn_profile resources.",
		"BunnyCDN":       "Cost-effective CDN with image optimization and edge scripting.",
		"None":           "No CDN. Assets served directly from origin.",
	},

	"domain_strategy": {
		"Subdomain per service": "Each service gets its own subdomain (api.example.com, auth.example.com). Clear separation; requires wildcard TLS.",
		"Path-based routing":    "All services under one domain, differentiated by path prefix (/api/*, /auth/*). Single certificate.",
		"Single domain":         "Everything served from one domain and path. Simple; limited separation.",
		"Custom":                "Custom domain strategy. Generates placeholder routing rules.",
	},

	"cors_infra": {
		"Reverse proxy (Nginx/Caddy)": "CORS headers added by the reverse proxy layer. Centralized; no application code needed.",
		"Application-level":           "Each service handles its own CORS middleware. More granular control per endpoint.",
		"CDN/WAF":                     "CORS handled at the CDN or WAF layer. Edge-level enforcement.",
		"Both":                        "Both application-level and proxy/CDN CORS headers. Belt-and-suspenders approach.",
	},

	"cors_strategy": {
		"Permissive":       "Allow all origins (*). Suitable for public APIs with no sensitive data.",
		"Strict allowlist": "Only explicitly listed origins are allowed. Generates allowed-origins environment variable.",
		"Same-origin":      "Requests only allowed from the same origin. Strictest; no cross-origin access.",
	},

	"ssl_cert": {
		"cert-manager (k8s)": "Kubernetes cert-manager operator. Automatically provisions and renews Let's Encrypt certificates.",
		"Caddy (auto)":       "Caddy web server with automatic HTTPS. Handles certificate provisioning and renewal.",
		"AWS ACM":            "AWS Certificate Manager. Generates ACM certificate request and DNS validation records.",
		"Manual rotation":    "Certificates managed and rotated manually or by a custom script.",
		"Cloudflare (edge)":  "Cloudflare manages TLS at the edge. Origin certificate optional.",
		"None":               "No SSL certificate management configured.",
	},

	// ── INFRASTRUCTURE: CI/CD ─────────────────────────────────────────────────

	"deploy_strategy": {
		"Rolling":    "Gradually replaces old instances with new ones. Zero downtime. Some versions run simultaneously briefly.",
		"Blue-green": "Two identical environments; traffic switches instantly. Zero downtime. Easy rollback.",
		"Canary":     "Small percentage of traffic routed to new version first. Monitors for errors before full rollout.",
		"Recreate":   "Old version stopped before new version starts. Brief downtime. Simplest strategy.",
	},

	"iac_tool": {
		"Terraform": "HashiCorp's declarative IaC. Provider-agnostic. Generates .tf files for all infrastructure with state management.",
		"Pulumi":    "IaC using TypeScript, Python, or Go. Generates Pulumi stacks with typed infrastructure components.",
		"CDK":       "AWS Cloud Development Kit. Define infrastructure in TypeScript/Python against AWS constructs.",
		"Ansible":   "Agentless configuration management via YAML playbooks. Generates Ansible roles and inventory.",
		"Helm":      "Kubernetes package manager. Generates Helm chart templates and values.yaml.",
		"None":      "No IaC tooling. Infrastructure managed manually.",
	},

	"registry": {
		"ECR":          "AWS Elastic Container Registry. Tight IAM integration. Generates ECR repository and push permissions.",
		"GCR/Artifact": "Google Container Registry or Artifact Registry. Generates GCP service account and push config.",
		"ACR":          "Azure Container Registry. Generates Azure managed identity and ACR pull config.",
		"Docker Hub":   "Public/private Docker Hub registry. Generates docker login step in CI pipeline.",
		"GHCR":         "GitHub Container Registry. Free for public images; integrated with GitHub Actions.",
		"Self-hosted":  "Self-hosted registry (Harbor, Nexus). Generates registry credentials and trust config.",
	},

	"secrets_mgmt": {
		"AWS Secrets Manager":        "AWS-managed secret storage with automatic rotation. Generates IAM policies and SDK calls.",
		"HashiCorp Vault":            "Open-source secrets platform. Dynamic secrets and encryption-as-a-service. Generates Vault policy files.",
		"GCP Secret Manager":         "Google Cloud-managed secrets. Generates IAM bindings and Secret Manager client calls.",
		"Azure Key Vault":            "Azure-managed secrets and keys. Generates Managed Identity access and Key Vault SDK integration.",
		"Kubernetes Secrets":         "Native Kubernetes Secrets as env vars or files. Generates Secret manifests and volume mounts.",
		"Doppler":                    "Developer-friendly secrets platform with ENV injection. Generates Doppler CLI setup.",
		"Environment variables only": "Plain environment variables. Generates .env.example files.",
	},

	"backup_dr": {
		"Cross-region replication": "Database replicated to another region. Survive regional outage with low RTO/RPO.",
		"Daily snapshots":          "Automated daily database snapshots stored in durable object storage. Simple and cost-effective.",
		"Managed provider DR":      "Cloud provider's managed disaster recovery (AWS RDS Multi-AZ, GCP Cloud SQL HA). Zero-effort DR.",
		"None":                     "No backup or disaster recovery configured.",
	},

	// ── INFRASTRUCTURE: Compute ───────────────────────────────────────────────

	"compute_env": {
		"Bare Metal":          "Physical servers without virtualization. Maximum performance and I/O throughput. Generates server provisioning scripts.",
		"VM":                  "Virtual machines on cloud or on-premises. Familiar deployment model. Generates Terraform VM resources.",
		"Containers (Docker)": "Docker containers on a single host or cluster. Portable and reproducible. Generates Dockerfiles.",
		"Kubernetes":          "Container orchestration at scale. Self-healing, autoscaling. Generates Kubernetes manifests.",
		"Serverless (FaaS)":   "Functions invoked on demand. No server management. Generates Lambda/Cloud Functions handlers.",
		"PaaS":                "Platform as a Service. Push to deploy; infrastructure abstracted. Generates platform config files.",
	},

	"cloud_provider": {
		"AWS":             "Amazon Web Services. Widest service catalog. Generates Terraform AWS resources and IAM configuration.",
		"GCP":             "Google Cloud Platform. Strong in data analytics and ML. Generates Terraform GCP resources.",
		"Azure":           "Microsoft Azure. Strong enterprise and Active Directory integration. Generates Terraform Azure resources.",
		"Cloudflare":      "Cloudflare's edge platform (Workers, R2, D1). Generates Wrangler config and edge function scaffolding.",
		"Hetzner":         "Cost-effective European cloud. VMs, dedicated servers. Generates Terraform Hetzner resources.",
		"Self-hosted":     "On-premises or co-located servers. Full control. Generates Ansible playbooks.",
		"Other (specify)": "Custom cloud provider. Generates generic infrastructure configuration placeholders.",
	},

	"orchestrator": {
		"Kubernetes":           "Full container orchestration. Deployments, Services, Ingress, HPA. Generates complete K8s manifest set.",
		"Docker Swarm":         "Docker's native clustering. Simpler than Kubernetes. Generates docker-compose.yml with Swarm deploy constraints.",
		"ECS (Fargate)":        "AWS fully-managed container runtime. No node management. Generates ECS task definitions.",
		"Cloud Run":            "GCP serverless containers. Scales to zero. Generates Cloud Run service YAML.",
		"Azure Container Apps": "Azure serverless container platform. Generates Container Apps manifests and scaling rules.",
		"Fly.io":               "Global application platform. Generates fly.toml configuration.",
		"Nomad":                "HashiCorp's flexible orchestrator. Supports containers, VMs, and raw executables.",
		"None":                 "No container orchestrator. Manual deployment or systemd services.",
	},

	// ── INFRASTRUCTURE: Observability ─────────────────────────────────────────

	"logging": {
		"CloudWatch":         "AWS CloudWatch Logs. Generates CloudWatch log group Terraform and structured logging config.",
		"Datadog":            "SaaS observability with unified logs, metrics, and traces. Generates Datadog agent config.",
		"ELK Stack":          "Elasticsearch, Logstash, Kibana. Generates Logstash pipeline and Kibana dashboards.",
		"Loki + Grafana":     "Lightweight Grafana-native log aggregation. Cost-effective for Kubernetes. Generates Loki Helm values.",
		"GCP Cloud Logging":  "Google Cloud managed logging. Generates structured logging config and log sink Terraform.",
		"Azure Monitor Logs": "Azure Log Analytics. Generates Diagnostic Settings and KQL query examples.",
		"Fluentd/Fluent Bit": "Open-source log collectors with flexible routing. Generates Fluent Bit DaemonSet.",
		"None":               "No centralized logging. Application writes to stdout only.",
	},

	"metrics": {
		"Prometheus + Grafana": "Open-source metrics stack. Pull-based scraping. Generates Prometheus scrape configs and Grafana dashboards.",
		"Datadog":              "SaaS metrics with APM, dashboards, and anomaly detection. Generates Datadog agent and custom metrics.",
		"CloudWatch":           "AWS CloudWatch Metrics. Generates CloudWatch alarms and dashboard JSON.",
		"New Relic":            "Full-stack observability SaaS. Generates New Relic agent config.",
		"OpenTelemetry":        "Vendor-neutral telemetry standard. Generates OTEL SDK setup, exporters, and collector config.",
		"GCP Cloud Monitoring": "Google Cloud managed metrics. Generates metric descriptors and Terraform dashboard resources.",
		"Azure Monitor":        "Azure unified monitoring. Generates Azure Monitor workspace and alert rules.",
		"None":                 "No metrics collection.",
	},

	"tracing": {
		"Jaeger":             "Open-source distributed tracing. Generates Jaeger agent config and OTEL instrumentation.",
		"Zipkin":             "Open-source distributed tracing. Generates Zipkin reporter config.",
		"Datadog APM":        "Datadog application performance monitoring. Generates Datadog tracer init.",
		"AWS X-Ray":          "AWS-native distributed tracing. Generates X-Ray daemon config and SDK instrumentation.",
		"Google Cloud Trace": "GCP-native distributed tracing. Generates Cloud Trace exporter.",
		"OpenTelemetry":      "Vendor-neutral trace instrumentation. Generates OTEL trace SDK and exporter config.",
		"None":               "No distributed tracing.",
	},

	"error_tracking": {
		"Sentry":   "Error tracking with full stack traces, breadcrumbs, and context. Generates Sentry SDK init.",
		"Datadog":  "Datadog error tracking integrated with APM and logs.",
		"Rollbar":  "Real-time error monitoring with deployment tracking. Generates Rollbar SDK init.",
		"Built-in": "Platform or framework built-in error logging (CloudWatch, GCP Error Reporting). No extra dependency.",
		"None":     "No error tracking configured.",
	},

	"health_checks": {
		"true":  "Health check endpoints generated for each service. Generates /health or /readyz + /livez handlers for Kubernetes probes.",
		"false": "No health check endpoints generated.",
	},

	"alerting": {
		"PagerDuty":         "On-call incident management. Generates PagerDuty integration key and escalation policy.",
		"OpsGenie":          "Atlassian alert management and on-call scheduling. Generates OpsGenie API integration.",
		"Alertmanager":      "Prometheus Alertmanager. Routes alerts to receivers (Slack, email, PagerDuty). Generates alertmanager.yml.",
		"Datadog Monitors":  "Datadog's built-in alerting on metrics and logs. Generates monitor Terraform resources.",
		"CloudWatch Alarms": "AWS CloudWatch threshold-based alarms. Generates Terraform CloudWatch alarm resources.",
		"Slack webhooks":    "Simple Slack notifications for alerts. No on-call management. Generates Slack webhook integration.",
		"None":              "No alerting configured.",
	},

	"log_retention": {
		"7 days":     "Logs retained for 7 days. Low cost; suitable for development and staging.",
		"30 days":    "Logs retained for 30 days. Standard for most production workloads.",
		"90 days":    "Logs retained for 90 days. Required for many compliance frameworks.",
		"1 year":     "Logs retained for 1 year. Meets most audit and compliance requirements.",
		"Indefinite": "Logs never automatically deleted. Highest cost. Use only when indefinite audit trails are required.",
	},

	// ── CROSSCUT: Testing ─────────────────────────────────────────────────────

	"unit": {
		"Go testing": "Go's built-in testing package. Table-driven tests. Generates _test.go files with TestXxx functions.",
		"Testify":    "Go test assertions and mocking. Generates test files using assert, require, and mock packages.",
		"Jest":       "JavaScript/TypeScript test runner with built-in mocking and coverage. Generates jest.config.ts and test files.",
		"Vitest":     "Vite-native unit testing. Faster than Jest for Vite projects. Generates vitest.config.ts.",
		"pytest":     "Python's most popular test framework. Fixtures and parametrize. Generates conftest.py and test_*.py files.",
		"unittest":   "Python's built-in test framework. Generates unittest.TestCase subclasses.",
		"JUnit":      "Java standard unit testing. Generates JUnit 5 test classes with @Test annotations.",
		"Kotest":     "Kotlin-native testing DSL. Generates Kotest Spec classes.",
		"TestNG":     "Java testing with data providers. Generates TestNG annotated test classes.",
		"xUnit":      ".NET community test framework. Generates xUnit test classes with [Fact] and [Theory].",
		"NUnit":      ".NET testing with [Test] attributes. Generates NUnit test fixtures.",
		"MSTest":     "Microsoft's .NET test framework. Generates MSTest test classes.",
		"cargo test": "Rust's built-in test runner. Generates #[test] and #[cfg(test)] annotated functions.",
		"RSpec":      "Ruby BDD-style testing. Generates RSpec spec files with describe/it blocks.",
		"minitest":   "Ruby standard library test framework. Generates Minitest::Test subclasses.",
		"PHPUnit":    "PHP unit testing framework. Generates PHPUnit TestCase subclasses.",
		"Pest":       "PHP elegant testing with expressive syntax. Generates Pest test files.",
		"Other":      "Custom test framework. Generates placeholder test structure.",
	},

	"integration": {
		"Testcontainers":  "Real Docker containers for databases and brokers in tests. Language-specific SDK. Generates container setup helpers.",
		"Docker Compose":  "Multi-container test environments. Generates compose file for integration test dependencies.",
		"In-memory fakes": "In-process implementations of external dependencies. Fast; no Docker. Generates fake adapters.",
		"None":            "No integration testing framework.",
	},

	"e2e": {
		"Playwright":       "Cross-browser E2E testing (Chromium, Firefox, WebKit). Generates Playwright tests with page objects.",
		"Cypress":          "JavaScript E2E testing with time-travel debugging. Generates Cypress spec files.",
		"Selenium":         "Classic browser automation. Language-agnostic. Generates Selenium test classes.",
		"Flutter Driver":   "Flutter's native E2E test framework. Generates flutter_driver test files.",
		"Integration Test": "Flutter official integration test package. Generates integration_test/ directory.",
		"Espresso":         "Android UI testing via Instrumentation. Generates Espresso test classes.",
		"UI Automator":     "Android cross-app UI testing. Generates UIAutomator test classes.",
		"XCUITest":         "Apple's official iOS/macOS UI testing. Generates XCUITest target and classes.",
		"EarlGrey":         "Google's iOS UI testing with synchronization. Generates EarlGrey test files.",
		"None":             "No E2E test framework.",
	},

	"fe_testing": {
		"Vitest":          "Vite-native component testing. Fast HMR-based test runner. Generates Vitest component tests.",
		"Jest":            "Component testing with jsdom. Generates Jest component tests.",
		"Testing Library": "User-centric component testing utilities. Works with Jest or Vitest. Generates RTL render helpers.",
		"Storybook":       "Component development and visual regression testing. Generates Storybook stories and test configurations.",
		"None":            "No frontend component testing.",
	},

	"load": {
		"k6":        "Go-based load testing with JavaScript scripting. Generates k6 test scripts with virtual user scenarios.",
		"Artillery": "Node.js load testing toolkit. YAML-based scenarios. Generates Artillery config and scenario files.",
		"JMeter":    "Java-based load testing with GUI. Generates JMeter JMX test plan.",
		"Locust":    "Python-based distributed load testing. Generates locustfile.py with user tasks.",
		"None":      "No load testing configured.",
	},

	"contract": {
		"Pact":                  "Consumer-driven contract testing. Generates Pact consumer and provider test files.",
		"Schemathesis":          "API schema-based fuzzing and contract validation. Generates Schemathesis test config.",
		"Dredd":                 "HTTP API testing against OpenAPI specs. Generates Dredd hooks and configuration.",
		"AsyncAPI validator":    "Validates async event messages against AsyncAPI schemas. Generates validator setup.",
		"Spring Cloud Contract": "JVM contract testing framework. Generates contract Groovy files and base test classes.",
		"None":                  "No contract testing.",
	},

	// ── CROSSCUT: Standards ────────────────────────────────────────────────────

	"dep_updates": {
		"Dependabot": "GitHub automated dependency update PRs. Generates .github/dependabot.yml with update schedules.",
		"Renovate":   "More configurable automated updates. Monorepo support and custom grouping. Generates renovate.json.",
		"Manual":     "Dependencies updated manually. No automated PR generation.",
		"None":       "No dependency update automation.",
	},

	"feature_flags": {
		"LaunchDarkly":      "Feature flag SaaS with targeting, gradual rollouts, and A/B testing. Generates LaunchDarkly SDK init.",
		"Unleash":           "Open-source feature flags. Self-hosted or cloud. Generates Unleash client setup.",
		"Flagsmith":         "Open-source feature flags and remote config. Generates Flagsmith SDK integration.",
		"Custom (env vars)": "Feature flags as environment variables. No external service. Generates typed accessors.",
		"None":              "No feature flag system.",
	},

	"changelog": {
		"Conventional Commits": "Automated changelog from commit messages following the Conventional Commits spec. Generates .commitlintrc and CHANGELOG.md template.",
		"Manual":               "Changelog maintained by hand. Generates CHANGELOG.md template.",
		"None":                 "No changelog strategy.",
	},

	"auto_generate": {
		"true":  "API documentation auto-generated from code annotations or schemas. Generates tooling configuration.",
		"false": "API documentation written manually.",
	},

	// ── REALIZE ───────────────────────────────────────────────────────────────

	"concurrency": {
		"1": "Sequential task execution. One task at a time. Easiest to debug; slowest for large manifests.",
		"2": "Two parallel tasks. Moderate speed increase with low risk of resource contention.",
		"4": "Four concurrent tasks. Good balance of speed and stability. Recommended for most manifests.",
		"8": "Eight concurrent tasks. Maximum throughput. Best for large manifests on capable machines.",
	},

	"verify": {
		"true":  "Verify generated code after each task. Compiles and lints before moving on. Failures trigger retries with escalating model tiers.",
		"false": "Skip verification. Faster generation but generated code may have compilation errors.",
	},

	"dry_run": {
		"false": "Execute agent calls and generate code. Full pipeline runs.",
		"true":  "Print the task execution plan without calling AI agents. Review generation order and task count before committing.",
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
