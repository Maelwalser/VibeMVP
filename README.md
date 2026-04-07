# VibeMenu

What vibe is on the menu today?

<img width="1900" height="1140" alt="Pasted image" src="https://github.com/user-attachments/assets/4c636af3-6a08-4acb-b7f1-07e6e4965c7e" />

VibeMenu is a Terminal User Interface (TUI) for declaratively specifying a complete software system architecture. Instead of writing boilerplate config files or lengthy architecture documents, you fill in a structured, interactive menu across 8 sections — covering everything from database schemas and API contracts to frontend pages and CI/CD pipelines. The result is a single `manifest.json` that fully describes your system, ready for downstream code generation via the built-in `realize` engine.

**Why VibeMenu?**

- **One manifest, entire stack** — define backend services, data models, API contracts, frontend pages, infrastructure, and testing strategy in a single place.
- **Smart field filtering** — options are dynamically narrowed based on your choices (e.g., selecting Go as a language filters frameworks to Fiber, Gin, Echo, etc.).
- **Non-linear editing** — fill sections in any order. Empty cross-references show as "unlinked" placeholders that resolve when you fill in the missing piece.
- **Multi-provider code generation** — the `realize` engine reads your manifest and dispatches parallel, agentic code generation across Claude, ChatGPT, Gemini, Mistral, and Llama.

> Still in development — not yet production-stable.

## Table of Contents

- [Installation](#installation)
- [The TUI Editor](#the-tui-editor)
  - [Section 0 — Description](#section-0--description)
  - [Section 1 — Backend](#section-1--backend)
  - [Section 2 — Data](#section-2--data)
  - [Section 3 — Contracts](#section-3--contracts)
  - [Section 4 — Frontend](#section-4--frontend)
  - [Section 5 — Infrastructure](#section-5--infrastructure)
  - [Section 6 — Cross-Cutting](#section-6--cross-cutting)
  - [Section 7 — Realize](#section-7--realize)
- [Architecture Diagram Overview](#architecture-diagram-overview)
- [manifest.json Reference](#manifestjson-reference)
- [Provider Configuration](#provider-configuration)
- [Code Generation (`realize`)](#code-generation-realize)
  - [Model Tiering](#model-tiering)
  - [Verification](#verification)
- [Skills System](#skills-system)

---

## Installation

### macOS / Linux — install script

```bash
curl -fsSL https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.sh | bash
```

Installs `vibemenu` and `realize` to `/usr/local/bin` (override with `INSTALL_DIR`).

### Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\Programs\vibemenu` and adds it to your user `PATH` (override with `$env:INSTALL_DIR`).

### Specific version

```bash
# macOS / Linux
VIBEMENU_VERSION=v1.0.0 bash install.sh

# Windows
$env:VIBEMENU_VERSION = "v1.0.0"; irm https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.ps1 | iex
```

### Manual download

Pre-built binaries for every platform are attached to each [GitHub Release](https://github.com/Maelwalser/vibemenu/releases):

| Platform | Archive |
|----------|---------|
| Linux x86-64 | `vibemenu-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `vibemenu-<version>-linux-arm64.tar.gz` |
| macOS x86-64 | `vibemenu-<version>-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `vibemenu-<version>-darwin-arm64.tar.gz` |
| Windows x86-64 | `vibemenu-<version>-windows-amd64.zip` |

Each archive contains two binaries: `vibemenu` (TUI editor) and `realize` (code generation). A `checksums.txt` file is included in each release for SHA-256 verification — the install scripts verify it automatically.

### Build from source

```bash
git clone https://github.com/Maelwalser/vibemenu
cd vibemenu
go build -o vibemenu ./cmd/agent
go build -o realize  ./cmd/realize
```

Requires Go 1.26+.

### Skills (bundled — no extra setup needed)

Skill files are **embedded in the `realize` binary**. On first run, they are automatically extracted to `.vibemenu/skills/` in the current directory:

```
realize --manifest manifest.json
# realize: extracting bundled skills to .vibemenu/skills
```

Existing files are never overwritten, so you can safely customise the skills directory. Point `realize` at a different location with `--skills`:

```bash
realize --skills /path/to/custom/skills --manifest manifest.json
```

---

## The TUI Editor

VibeMenu uses a vim-modal editing system with three modes: **Normal** (navigate sections, lists, tabs), **Insert** (type into text fields), and **Command** (`:w` to save, `:q` to quit, `:wq` to save and quit, `:1`-`:8` to jump to a section).

The editor is organized into 8 sections. Each section contains sub-tabs that group related configuration. Sections can be filled in any order — the dependency graph below shows the intended resolution order for code generation:

```
Description → Data → Backend → Contracts → Frontend → Infrastructure → Cross-Cutting → Realize
```

---

### Section 0 — Description

Free-text textarea for describing the project in natural language. This is injected into the code-generation prompt as high-level context, giving the AI agents an understanding of what the system is supposed to do before they generate code for each pillar.

---

### Section 1 — Backend

Sub-tabs: **Env** · **Services** · **Stack Config** · **Communication** · **Messaging** · **API Gateway** · **Jobs** · **Security** · **Auth**

The Backend section starts with an **architecture pattern** selector that controls which sub-tabs are visible:

| Architecture Pattern | Visible Sub-tabs |
|---------------------|-----------------|
| Monolith | Env · Services · Jobs · Security · Auth |
| Modular Monolith | Env · Services · Stack Config · Communication · Jobs · Security · Auth |
| Microservices | Env · Services · Stack Config · Communication · API Gateway · Jobs · Security · Auth |
| Event-Driven | Env · Services · Stack Config · Communication · Messaging · Jobs · Security · Auth |
| Hybrid | All sub-tabs |

**Env** — For monoliths, configure the single shared language, framework, and environment. For multi-service architectures, this shows global health dependencies. Languages supported: Go, TypeScript/Node, Python, Java, Kotlin, C#/.NET, Rust, Ruby, PHP, Elixir. Frameworks are dynamically filtered by language (e.g., Go offers Fiber, Gin, Echo, Chi, net/http, Connect; TypeScript/Node offers Express, Fastify, NestJS, Hono, tRPC, Elysia).

**Services** — Define your backend service units. Each service has a name, responsibility, language, framework (filtered by language), technologies (WebSocket, gRPC, REST, GraphQL, SSE, tRPC, MQTT, Kafka consumer), error format (filtered by technology), service discovery strategy, and environment assignment. Hybrid architectures add a pattern tag (Monolith part, Modular module, Microservice, Event processor, Serverless function).

**Stack Config** — Reusable language/framework combinations that services can reference instead of defining their stack inline. Useful for multi-language architectures where several services share the same tech stack.

**Communication** — Service-to-service links. Each link defines: from/to service, direction (unidirectional, bidirectional, pub/sub), protocol (REST, gRPC, GraphQL, WebSocket, Message Queue, Event Bus, Internal), trigger description, sync/async mode, resilience patterns (circuit breaker, retry with backoff, timeout, bulkhead), and request/response DTOs.

**Messaging** — Broker configuration for event-driven architectures. Supports Kafka, NATS, RabbitMQ, Redis Streams, AWS SQS/SNS, Google Pub/Sub, Azure Service Bus, and Pulsar. Deployment options are filtered by broker technology and cloud provider. Includes an event catalog where you define events (name, publisher, consumer, DTO, description) and configure serialization format (JSON, Protobuf, Avro, MessagePack, CloudEvents) and delivery guarantees (at-most-once, at-least-once, exactly-once).

**API Gateway** — Technology selection filtered by orchestrator and cloud provider (Kong, Traefik, NGINX Ingress, Envoy, AWS API Gateway, Cloudflare Workers). Configure routing strategy (path-based, header-based, domain-based) and features like rate limiting, JWT validation, SSL termination, load balancing, request caching, CORS handling, circuit breaking, and health checks. Link specific endpoints from the Contracts tab.

**Jobs** — Background job queues and cron jobs. Queue technology is filtered by backend language (e.g., Go offers Asynq, River, Temporal, Faktory). Configure concurrency, max retries, retry policy (exponential backoff, fixed interval, linear backoff), dead letter queue, worker service, and payload DTO. Cron jobs define a name, cron expression, handler, and timeout.

**Security** — WAF configuration (provider filtered by cloud provider), CAPTCHA (hCaptcha, reCAPTCHA, Cloudflare Turnstile), bot protection, rate limit strategy and backend, DDoS protection, and internal mTLS.

**Auth** — Authentication strategy (JWT, session-based, OAuth 2.0/OIDC, API keys, mTLS), identity provider (Self-managed, Auth0, Clerk, Supabase Auth, Firebase Auth, Keycloak, AWS Cognito), authorization model (RBAC, ABAC, ACL, ReBAC, Policy-based), token storage, session management, refresh token strategy, and MFA support. Includes a full role editor with permissions, role inheritance, and RBAC — roles defined here are referenced by endpoints and frontend pages for access control.

---

### Section 2 — Data

Sub-tabs: **Databases** · **Domains** · **Caching** · **File Storage**

**Databases** — Define database sources with alias, type (PostgreSQL, MySQL, SQLite, MongoDB, DynamoDB, Cassandra, Redis, Memcached, ClickHouse, Elasticsearch), version, namespace, and cache flag. Type-conditional fields appear based on your selection: SSL mode (PostgreSQL/MySQL), consistency level (Cassandra/MongoDB/DynamoDB), replication strategy, and connection pool sizing.

**Domains** — The source of truth for your system's data model. Each domain is a bounded context (e.g., User, Order, Product) with repeatable attributes. Attributes have a name, type (String, Int, Float, Boolean, DateTime, UUID, Enum, JSON/Map, Binary, Array, Ref), constraints (required, unique, not_null, min/max, length limits, email, url, regex, positive, future/past), default value, sensitive flag (for encryption/masking/audit), and validation rules. Relationships between domains support One-to-One, One-to-Many, and Many-to-Many with cascade behavior (CASCADE, SET NULL, RESTRICT, NO ACTION, SET DEFAULT). Domains are referenced throughout Contracts (DTOs), Backend (services), and Frontend (pages).

**Caching** — Named caching configurations with layer type (application-level, dedicated cache, CDN), strategy (cache-aside, read-through, write-through, write-behind, CDN purge), invalidation policy (TTL-based, event-driven, manual, hybrid), TTL, and entity selection from your domains.

**File Storage** — Object/file storage buckets. Supports S3, GCS, Azure Blob, MinIO, Cloudflare R2, and local disk. Configure access mode (public CDN-fronted, private signed URLs, internal only), max upload size, allowed MIME types, signed URL TTL, and which domains store files here.

**Governance** — Data governance policies applied to databases. Configure migration tool (filtered by backend language — e.g., golang-migrate, Atlas, goose for Go; Prisma Migrate, TypeORM for TypeScript), backup strategy, search technology (Elasticsearch, Meilisearch, Algolia, Typesense), retention policy, delete strategy (soft-delete, hard-delete, archival), PII encryption, compliance standards (GDPR, HIPAA, SOC2, PCI-DSS, ISO-27001, CCPA, PIPEDA), data residency, and archival storage.

---

### Section 3 — Contracts

Sub-tabs: **DTOs** · **Endpoints** · **API Versioning** · **External APIs**

**DTOs** — Data Transfer Objects with name, category (Request, Response, Event Payload, Shared/Common), source domain(s), and serialization protocol (REST/JSON, Protobuf, Avro, MessagePack, Thrift, FlatBuffers, Cap'n Proto). Protocol-specific fields appear automatically — Protobuf adds package, syntax, and options; Avro adds namespace and schema registry; Thrift/FlatBuffers/Cap'n Proto add namespace. Each DTO has repeatable fields with name, type, required/nullable flags, validation rules, and protocol-specific metadata (field numbers for Protobuf, field IDs for Thrift/Cap'n Proto).

**Endpoints** — API operations exposed by your service units. Each endpoint defines a service unit, name/path, protocol (REST, GraphQL, gRPC, WebSocket, Event), auth requirement, auth roles (multi-select from Backend roles), request/response DTOs, pagination strategy, rate limit level, and description. Protocol-conditional fields: HTTP method (REST), operation type (GraphQL), stream type (gRPC), direction (WebSocket).

**API Versioning** — Per-protocol versioning strategies. REST supports URL path, header, or query param versioning. GraphQL uses schema versioning or field deprecation. gRPC uses package versioning. WebSocket and Event protocols have their own strategies. Also configures current version, deprecation policy (sunset header, versioned removal notice, changelog entry), and global pagination strategy.

**External APIs** — Third-party integrations (e.g., Stripe, SendGrid, Twilio). Each defines a provider, protocol (REST, GraphQL, gRPC, WebSocket, Webhook, SOAP), auth mechanism (API Key, OAuth2, Bearer, Basic, mTLS), and failure strategy (circuit breaker, retry with backoff, fallback value, timeout). Protocol-specific fields include base URL, rate limits, webhook paths, TLS mode, subprotocol, HMAC headers, and SOAP version. Each external API has repeatable interactions (individual API calls) with name, path, request/response DTOs, and protocol-specific operation details.

---

### Section 4 — Frontend

Sub-tabs: **Tech** · **Theme** · **Pages** · **Navigation** · **i18n** · **A11y/SEO** · **Assets**

**Tech** — Comprehensive frontend technology selection. Language (TypeScript, JavaScript, Dart, Kotlin, Swift), platform (Web SPA, Web SSR/SSG, Mobile cross-platform, Mobile native, Desktop), framework (filtered by language — e.g., React, Vue, Svelte, Angular, Solid, Qwik, HTMX for TypeScript; Flutter for Dart; SwiftUI/UIKit for Swift), meta-framework (filtered by framework — Next.js, Remix, Astro for React; Nuxt for Vue; SvelteKit for Svelte), package manager, styling (Tailwind CSS, CSS Modules, Styled Components, Sass/SCSS, UnoCSS), component library (filtered by framework — shadcn/ui, Radix, Material UI, Ant Design, Headless UI, DaisyUI for React), state management (filtered — Zustand, Redux Toolkit, Jotai for React; Pinia for Vue; Svelte stores for Svelte), data fetching (TanStack Query, SWR, Apollo Client, tRPC client for React), form handling (React Hook Form, Formik for React; Vee-Validate for Vue), validation library, PWA support, realtime strategy (WebSocket, SSE, Polling), image optimization, auth flow type, error boundary, bundle optimization, frontend testing, and linter.

**Theme** — Visual design configuration. Dark mode strategy (toggle, system preference, dark only), border radius (sharp, subtle, rounded, pill), spacing scale, elevation style (shadows, borders, flat), motion (none, subtle transitions, animated), vibe (Professional, Playful, Minimal, Bold, Elegant, Technical, Creative, Friendly, Serious, Modern), font family, colors, and prose description of the visual feel.

**Pages** — Define application pages/routes. Each page has a name, route path, auth requirement, layout (default, sidebar, full-width, blank, custom), description, core actions, loading strategy (skeleton, spinner, progressive, instant SSR/SSG), error handling (inline, toast, error boundary, retry), auth roles (multi-select from Backend roles for role-based page access), and linked pages. Pages can define component actions with 12+ action types: Fetch, Submit, Download, Upload, Delete, Refresh, Export, Navigate, Toast, State, and Custom.

**Navigation** — Nav type (top bar, sidebar, bottom tabs, hamburger menu, combined), breadcrumbs toggle, and auth-aware navigation (show/hide items based on auth state).

**i18n** — Internationalization settings. Enable/disable, default locale (40+ locale options), supported locales, i18n library (i18next, next-intl, react-i18next, LinguiJS, vue-i18n), and timezone handling (UTC always, user preference, auto-detect, manual).

**A11y/SEO** — Accessibility: WCAG level (A, AA, AAA). SEO: rendering strategy (SSR, SSG, ISR, Prerender), sitemap generation, meta tag management (manual, react-helmet, framework-native), analytics (PostHog, Google Analytics 4, Plausible, Mixpanel, Segment), and frontend RUM (Sentry, Datadog RUM, LogRocket, New Relic Browser).

**Assets** — Attach design assets, mockups, or inspiration references. Each asset has a name, path/URL, type (image, icon, font, video, mockup, moodboard), format (png, jpg, svg, gif, mp4, pdf, figma, sketch), usage classification (project or inspiration), and description.

---

### Section 5 — Infrastructure

Sub-tabs: **Environments** · **Networking** · **CI/CD** · **Observability**

**Environments** — Named deployment environments (e.g., dev, staging, prod) referenced by all other pillars. Each environment defines compute type (Bare Metal, VM, Containers, Kubernetes, Serverless, PaaS), cloud provider (AWS, GCP, Azure, Cloudflare, Hetzner, Self-hosted), orchestrator (filtered by compute — Docker Compose, K3s, K8s managed, Nomad, ECS, Cloud Run), and regions.

**Networking** — DNS provider (filtered by cloud — Route53 for AWS, Cloud DNS for GCP, Cloudflare), TLS/SSL (Let's Encrypt, Cloudflare, ACM), reverse proxy (filtered by orchestrator — NGINX Ingress/Traefik/Envoy for Kubernetes; Nginx/Caddy/Traefik for Docker Compose), CDN (CloudFront for AWS, Cloud CDN for GCP, Cloudflare), primary domain, domain strategy (subdomain per service, path-based routing, single domain), CORS configuration, and SSL certificate management.

**CI/CD** — Pipeline configuration. Platform (GitHub Actions, GitLab CI, Jenkins, CircleCI, ArgoCD, Tekton), container registry (filtered by cloud — ECR for AWS, GCR/Artifact Registry for GCP, ACR for Azure), deploy strategy (filtered by orchestrator — rolling, blue-green, canary, recreate), IaC tool (filtered by cloud — Terraform, Pulumi, CloudFormation/CDK for AWS, Bicep for Azure, Wrangler for Cloudflare), secrets management (filtered by cloud — AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, HashiCorp Vault), container runtime (filtered by language — scratch/distroless for Go/Rust, node:alpine for TypeScript, python:slim for Python), and backup/DR strategy.

**Observability** — Logging (filtered by cloud — Loki+Grafana, ELK Stack, CloudWatch, Datadog), metrics (Prometheus+Grafana, Datadog, CloudWatch, New Relic), tracing (filtered by metrics backend — OpenTelemetry+Jaeger, Datadog APM, AWS X-Ray, Cloud Trace), error tracking (Sentry, Datadog, Rollbar), health check generation, alerting (filtered by metrics — Grafana Alerting, PagerDuty, OpsGenie, CloudWatch Alarms), and log retention policy.

---

### Section 6 — Cross-Cutting

Sub-tabs: **Testing** · **Docs** · **Standards**

**Testing** — Framework selections dynamically filtered by your backend languages and frontend tech:

| Test Type | How it's filtered |
|-----------|------------------|
| Unit | By backend language (Go testing/Testify for Go, Jest/Vitest for TypeScript, pytest for Python, JUnit for Java, etc.) |
| Integration | By architecture pattern (Testcontainers/Docker Compose for microservices; in-memory fakes for monoliths) |
| E2E | By frontend platform (Playwright/Cypress/Selenium for web; Flutter Driver for Dart; Espresso for Android; XCUITest for iOS) |
| API | By communication protocols (Bruno/Hurl for REST; GraphQL Playground for GraphQL; grpcurl/BloomRPC for gRPC) |
| Load | k6, Artillery, JMeter (plus Locust when Python is a backend language) |
| Contract | By architecture pattern (Pact/Schemathesis for microservices; AsyncAPI validator for event-driven) |

**Docs** — Per-protocol documentation format. REST gets OpenAPI/Swagger, GraphQL gets GraphQL Playground/SDL, gRPC gets reflection/buf.build, WebSocket and Events get AsyncAPI/CloudEvents spec. Also configures auto-generation from code annotations and changelog strategy (Conventional Commits, Manual).

**Standards** — Dependency update strategy (Dependabot, Renovate), feature flags (LaunchDarkly, Unleash, Flagsmith), backend linter (filtered by language), and frontend linter (ESLint+Prettier, Biome, oxlint).

---

### Section 7 — Realize

Code generation configuration for the `realize` engine:

| Field | Description |
|-------|-------------|
| App Name | Application name used in generated code |
| Output Dir | Destination directory for generated files |
| Model | Global LLM model for code generation |
| Concurrency | Max parallel tasks (1, 2, 4, 8) |
| Verify | Run language verifiers after generation |
| Dry Run | Print task plan without calling agents |

**Per-section model overrides** let you assign different LLM providers and tiers to each pillar (backend, data, contracts, frontend, infra, crosscut) using the format `"Provider · Tier"` (e.g., `"Claude · Sonnet"`, `"Gemini · Flash"`). Sections without an override inherit the global model.

---

## Architecture Diagram Overview

Press `P` to open the architecture diagram overview, which visualizes the dependency graph across all configured pillars.

<img width="1900" height="1138" alt="image" src="https://github.com/user-attachments/assets/333e5fac-cf88-4490-918a-ddba848df91d" />

---

## manifest.json Reference

Saved on `:w` or `Ctrl+S`. Unconfigured pillars are omitted automatically.

```json
{
  "created_at": "2026-01-01T00:00:00Z",
  "description": "Free-text project description",

  "backend": {
    "arch_pattern": "Microservices",
    "services": [],
    "stack_configs": [],
    "auth": { "strategy": "JWT", "roles": [] },
    "waf": {},
    "job_queues": [],
    "cron_jobs": []
  },

  "data": {
    "databases": [],
    "domains": [],
    "cachings": [],
    "file_storages": []
  },

  "contracts": {
    "dtos": [],
    "endpoints": [],
    "versioning": {},
    "external_apis": []
  },

  "frontend": {
    "tech": {},
    "theme": {},
    "pages": [],
    "navigation": {},
    "i18n": {},
    "a11y_seo": {},
    "assets": []
  },

  "infrastructure": {
    "networking": {},
    "cicd": {},
    "observability": {},
    "environments": []
  },

  "cross_cutting": {
    "testing": {},
    "docs": {}
  },

  "realize": {
    "app_name": "my-app",
    "output_dir": "output",
    "model": "claude-sonnet-4-6",
    "concurrency": 4,
    "verify": true,
    "dry_run": false,
    "section_models": {
      "backend": "Claude · Sonnet",
      "data": "Claude · Sonnet",
      "contracts": "Claude · Haiku",
      "frontend": "Claude · Sonnet",
      "infra": "Claude · Haiku",
      "crosscut": "Claude · Haiku"
    }
  },

  "configured_providers": {}
}
```

---

## Provider Configuration

Open the **Provider Menu** with `Shift+M` to configure LLM providers interactively.

Supported providers and their model tiers:

| Provider | Fast | Medium | Slow |
|----------|------|--------|------|
| **Claude** | Haiku (`claude-haiku-4-5-20251001`) | Sonnet (`claude-sonnet-4-6`) | Opus (`claude-opus-4-6`) |
| **ChatGPT** | o3-mini | 4o (`gpt-4o`) | o1 |
| **Gemini** | Flash (`gemini-2.0-flash`) | Pro (`gemini-2.0-pro-exp`) | Ultra (`gemini-ultra`) |
| **Mistral** | Nemo (`open-mistral-nemo`) | Small (`mistral-small-2409`) | Large (`mistral-large-2411`) |
| **Llama** | 8B (`llama-3.2-8b-preview`) | 70B (`llama-3.3-70b-versatile`) | 405B (`llama-3.1-405b-reasoning`) |

Authentication is configured per provider via API key or OAuth 2.0 PKCE flow. Credentials are stored in `manifest.json` under `configured_providers`.

Per-section overrides in `section_models` use the format `"Provider · Tier"` (e.g. `"Claude · Sonnet"`). Sections without an override inherit the global model selection.

**Environment variable fallback:**

| Provider | Environment Variable |
|----------|---------------------|
| Claude | `ANTHROPIC_API_KEY` |
| ChatGPT | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Llama (Groq) | `GROQ_API_KEY` |

---

## Code Generation (`realize`)

The `realize` binary reads `manifest.json` and drives a parallel, agentic code-generation pipeline.

```
manifest.json
    |
DAG construction     -> execution graph with dependency edges
    |
Orchestrator         -> parallel task dispatch (bounded by --parallel)
    |  (per task)
Runner               -> deterministic fixes -> agent call -> verify -> retry (escalating tier)
    |
Shared memory        -> stores completed outputs; downstream agents read upstream signatures
    |
File writer          -> writes generated files to --output directory
```

### CLI Flags

```
--manifest   path to manifest.json       (default: manifest.json)
--output     output directory            (default: output)
--skills     skills directory            (default: .vibemenu/skills)
--retries    max retry attempts per task (default: 3)
--parallel   max concurrent tasks        (default: 1)
--dry-run    print task plan, no agents
--verbose    print token usage + logs
```

### Model Tiering

Tasks are automatically assigned a model tier based on complexity. On verification failure, the engine escalates to the next tier for the retry:

| Tier | Task Kinds | Claude / OpenAI / Gemini |
|------|-----------|--------------------------|
| **Fast** | Contracts, docs, Docker, CI/CD | Haiku 4.5 / o3-mini / Flash |
| **Medium** | Services, auth, data, frontend, Terraform, testing | Sonnet 4.6 / 4o / Pro |
| **Slow** | Escalation fallback | Opus 4.6 / o1 / Ultra |

### Verification

After each task, generated code is checked by a language-specific verifier. Failed tasks apply deterministic fixes (formatting, unused imports) before retrying with an escalated model tier.

| Language | Check |
|----------|-------|
| Go | `go build` + `go vet` |
| TypeScript | `tsc --noEmit` |
| Python | `python -m py_compile` |
| Terraform | `terraform validate` |

---

## Skills System

Skills are markdown files that inject domain-specific guidance into agent prompts.

**Location:** `.vibemenu/skills/` (override with `--skills`)

Each `.md` file defines a named skill. Technology aliases automatically map framework names (e.g. `nextjs`) to the relevant skill file. Universal skills apply to all tasks of a given kind regardless of tech stack.

```
.vibemenu/
  skills/
    nextjs.md
    postgres.md
    terraform-aws.md
```
