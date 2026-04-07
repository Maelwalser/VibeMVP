# Realize Phase — Configuration Reference

Complete reference for all configuration options that govern the `cmd/realize` code-generation pipeline. Covers CLI flags, manifest fields, model tiering, provider registry, skills, dependency resolution, verification, and retry behaviour.

---

## Table of Contents

1. [CLI Flags](#1-cli-flags)
2. [Manifest Realize Options](#2-manifest-realize-options)
3. [Provider Assignments](#3-provider-assignments)
4. [Default Constants](#4-default-constants)
5. [Task Kinds & Trigger Conditions](#5-task-kinds--trigger-conditions)
6. [Task Output Directories](#6-task-output-directories)
7. [Model Tiering & Escalation](#7-model-tiering--escalation)
8. [Provider & Model Registry](#8-provider--model-registry)
9. [Section Model Overrides](#9-section-model-overrides)
10. [Skills System](#10-skills-system)
11. [Agent Role Descriptions](#11-agent-role-descriptions)
12. [Dependency Resolution](#12-dependency-resolution)
13. [Verification System](#13-verification-system)
14. [Deterministic Fixes](#14-deterministic-fixes)
15. [Retry & Error Classification](#15-retry--error-classification)
16. [Task Payload Fields](#16-task-payload-fields)
17. [Progress State & Resume](#17-progress-state--resume)
18. [Wave Parallelism](#18-wave-parallelism)

---

## 1. CLI Flags

**Source:** `cmd/realize/main.go`

```
realize [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-manifest` | string | `manifest.json` | Path to the manifest JSON file |
| `-output` | string | `output` | Directory where generated code is written |
| `-skills` | string | `.vibemenu/skills` | Directory containing skill markdown files |
| `-retries` | int | `3` | Max verification retry attempts per task |
| `-parallel` | int | `0` (= 1 at runtime) | Max tasks running concurrently |
| `-dry-run` | bool | `false` | Print task plan without calling any agent |
| `-verbose` | bool | `false` | Print token usage and thinking logs |
| `-provider` | string | `""` | Default LLM provider: `Claude`, `Gemini`, `ChatGPT`, `Mistral`, `Llama` |
| `-api-key` | string | `""` | API key for the default provider (falls back to env var) |

> **Note:** When `-parallel` is set to `0` or any value ≤ 0, it resolves to `1` at runtime.

> **Bundled skills:** If the `-skills` directory does not exist, bundled skills are automatically extracted from the embedded filesystem to that path at startup.

---

## 2. Manifest Realize Options

**Source:** `internal/manifest/manifest.go` — `RealizeOptions`

Stored under the `"realize"` key in `manifest.json`. Configured via the **Realize** tab in the TUI.

```json
{
  "realize": {
    "app_name": "my-app",
    "output_dir": ".",
    "concurrency": 4,
    "verify": true,
    "dry_run": false,
    "provider": "Claude",
    "tier_fast": "claude-haiku-4-5-20251001",
    "tier_medium": "claude-sonnet-4-6",
    "tier_slow": "claude-opus-4-6"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `app_name` | string | Application name used in generated code identifiers |
| `output_dir` | string | Root directory for all generated files |
| `concurrency` | int | Max parallel tasks; mirrors `-parallel` CLI flag |
| `verify` | bool | Run language verifiers after each agent call |
| `dry_run` | bool | Print task plan without executing; mirrors `-dry-run` CLI flag |
| `provider` | string | Provider label for model selection (empty = use default Claude) |
| `tier_fast` | string | Model ID for low-complexity tasks |
| `tier_medium` | string | Model ID for medium-complexity tasks |
| `tier_slow` | string | Model ID for high-complexity / escalation tasks |
| `model` | string | *(legacy)* Global default model ID — not editable from UI |
| `section_models` | map[string]string | *(legacy)* Per-pillar model override — not editable from UI |

---

## 3. Provider Assignments

**Source:** `internal/manifest/manifest.go` — `ProviderAssignments`

Stored under `"configured_providers"` in `manifest.json`. Configured via the **Provider Menu** (`Shift+M`) in the TUI.

```json
{
  "configured_providers": {
    "Claude": {
      "provider": "Claude",
      "model": "Sonnet",
      "version": "",
      "auth": "API Key",
      "credential": "sk-ant-..."
    },
    "ChatGPT": {
      "provider": "ChatGPT",
      "model": "4o",
      "version": "4o-2024",
      "auth": "API Key",
      "credential": "sk-..."
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Provider name: `"Claude"`, `"ChatGPT"`, `"Gemini"`, `"Mistral"`, `"Llama"`, `"Custom"` |
| `model` | string | Tier name within that provider (e.g. `"Sonnet"`, `"4o"`, `"Flash"`) |
| `version` | string | Optional specific version string (e.g. `"o3-mini"`, `"1.5"`); empty = use fallback |
| `auth` | string | `"API Key"` or `"OAuth"` |
| `credential` | string | API key or OAuth token for this provider |

Provider credentials are also persisted locally to `~/.config/vibemenu/providers.json` with 0600 file permissions.

**Provider environment variables** (fallback when no credential configured):

| Provider | Env Var |
|----------|---------|
| Claude | `ANTHROPIC_API_KEY` |
| ChatGPT | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Llama | `GROQ_API_KEY` |

---

## 4. Default Constants

**Source:** `internal/realize/config/defaults.go`

| Constant | Value | Description |
|----------|-------|-------------|
| `DefaultModel` | `"claude-opus-4-6"` | Fallback model when no tier or section override applies |
| `DefaultMaxTokens` | `64000` | Max output tokens per agent call (applies to all providers) |
| `MaxSkillBytes` | `2000` | Skill markdown files are truncated to this many bytes before injection |
| `MaxFileChars` | `4000` | Max characters included from a single dependency output file |
| `MaxTotalChars` | `12000` | Total character budget across all dependency outputs per prompt (fallback) |
| `RateLimitBackoffBase` | `60` | Base seconds for rate-limit backoff: wait = `(attempt+1) × 60s` |

### Per-Task Character Budgets

Tasks that aggregate more upstream layers get a larger shared-memory window:

| Task Kind | Budget | Rationale |
|-----------|--------|-----------|
| `backend.service.bootstrap` | `30000` | Sees repo + service + handler simultaneously |
| `backend.service.handler` | `20000` | Sees repo + service + auth |
| `backend.auth` | `20000` | Needs all service interfaces |
| `backend.gateway` | `20000` | Needs full service surface |
| `contracts` | `20000` | Aggregates all service + data output |
| `frontend` | `20000` | Needs contracts + data types |
| `backend.service.logic` | `15000` | Sees repo + data schemas |
| `backend.service.repository` | `10000` | Sees data schemas only |
| *(all others)* | `12000` | Default `MaxTotalChars` |

---

## 5. Task Kinds & Trigger Conditions

**Source:** `internal/realize/dag/dag.go`, `internal/realize/dag/builder.go`

Each task kind maps to one LLM agent call (except `backend.service.deps` which runs the package manager).

### Data Pillar

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `data.schemas` | `data.schemas` | Always created (domains/databases context available) |
| `data.migrations` | `data.migrations` | Always created alongside `data.schemas` |

### Backend Pillar — Service Chain

Each service produces **6 tasks** chained in dependency order:

| Task Kind | Task ID Pattern | Description |
|-----------|-----------------|-------------|
| `backend.service.plan` | `svc.<slug>.plan` | Architect phase: `go.mod` skeleton + repository interfaces |
| `backend.service.deps` | `svc.<slug>.deps` | Package manager: `go mod tidy` (no LLM) |
| `backend.service.repository` | `svc.<slug>.repository` | Data-access layer implementations |
| `backend.service.logic` | `svc.<slug>.service` | Business logic / service layer |
| `backend.service.handler` | `svc.<slug>.handler` | HTTP handlers + routing |
| `backend.service.bootstrap` | `svc.<slug>.bootstrap` | `main.go`, `.env.example`, wiring |

**Architecture patterns:**
- **Monolith / Modular Monolith** — single synthetic service named `"monolith"` (`svc.monolith.*`)
- **Microservices / Event-Driven / Hybrid** — one chain per `m.Backend.Services` entry; slug = lowercase name with spaces replaced by hyphens

### Backend Pillar — Auxiliary Tasks

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `backend.auth` | `backend.auth` | `m.Backend.Auth.Strategy != ""` |
| `backend.messaging` | `backend.messaging` | `m.Backend.Messaging != nil` |
| `backend.gateway` | `backend.gateway` | `m.Backend.APIGateway != nil` |

### Contracts Pillar

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `contracts` | `contracts` | `m.Contracts.DTOs` or `m.Contracts.Endpoints` non-empty |

### Frontend Pillar

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `frontend` | `frontend` | `m.Frontend.Tech.Framework != ""` |

### Infrastructure Pillar

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `infra.docker` | `infra.docker` | Always when services or frontend exist |
| `infra.terraform` | `infra.terraform` | `m.Infra.CICD.IaCTool != ""` and `!= "None"` |
| `infra.cicd` | `infra.cicd` | `m.Infra.CICD.Platform != ""` and `!= "none"` |

### Cross-Cutting Pillar

| Task Kind | Task ID | Trigger Condition |
|-----------|---------|-------------------|
| `crosscut.testing` | `crosscut.testing` | `m.CrossCut.Testing.Unit != ""` or `m.CrossCut.Testing.E2E != ""` |
| `crosscut.docs` | `crosscut.docs` | `m.CrossCut.Docs.APIDocs != ""` |

### Post-Pipeline Tasks

| Task Kind | Task ID | Description |
|-----------|---------|-------------|
| `backend.reconciliation` | `backend.reconciliation` | Cross-task compilation fix phase (runs after all service chains) |
| `integration.repair` | `integration.repair` | Post-pipeline integration repair (synthetic kind) |

---

## 6. Task Output Directories

**Source:** `internal/realize/dag/builder.go` — `serviceOutputDirs()`

The `OutputDir` field in `TaskPayload` tells each agent where to place its files, relative to the output root.

| Scenario | Service Dirs |
|----------|-------------|
| Backend-only monolith | `{"monolith": "."}` |
| Monolith + frontend | `{"monolith": "backend", "frontend": "frontend"}` |
| Microservices + frontend | `{"user-api": "services/user-api", "order-api": "services/order-api", "frontend": "frontend"}` |

The `service_dirs` map is also injected into the `infra.docker` payload so Dockerfile build contexts are set correctly without the agent guessing paths.

---

## 7. Model Tiering & Escalation

**Source:** `internal/realize/orchestrator/tier.go`

### Default Tier Per Task Kind

| Task Kind | Default Tier | Rationale |
|-----------|-------------|-----------|
| `data.schemas` | Medium (Sonnet) | Entity design requires reasoning |
| `data.migrations` | Fast (Haiku) | Pure SQL DDL, no reasoning needed |
| `backend.service.plan` | Medium (Sonnet) | Architectural reasoning about interfaces |
| `backend.service.deps` | *(no LLM)* | Package manager only |
| `backend.service.repository` | Fast (Haiku) | Repetitive CRUD boilerplate |
| `backend.service.logic` | Medium (Sonnet) | Business rules require reasoning |
| `backend.service.handler` | Medium (Sonnet) | Routing + auth integration |
| `backend.service.bootstrap` | Fast (Haiku) | Wiring boilerplate |
| `backend.auth` | Medium (Sonnet) | Security-critical |
| `backend.messaging` | Medium (Sonnet) | Distributed systems reasoning |
| `backend.gateway` | Medium (Sonnet) | Platform integration |
| `contracts` | Fast (Haiku) | Well-understood DTO/OpenAPI format |
| `frontend` | Medium (Sonnet) | Multi-file framework reasoning |
| `infra.docker` | Fast (Haiku) | Deterministic Dockerfile patterns |
| `infra.terraform` | Medium (Sonnet) | IaC multi-resource reasoning |
| `infra.cicd` | Fast (Haiku) | Standard CI pipeline structure |
| `crosscut.testing` | Medium (Sonnet) | TDD reasoning across layers |
| `crosscut.docs` | Medium (Sonnet) | Documentation reasoning |
| `backend.reconciliation` | Slow (Opus) | Cross-task compilation fix |
| `integration.repair` | Slow (Opus) | Cross-file integration repair |

### Escalation Path on Retry Failure

```
Attempt 0:  base model (from table above)
Attempt 1:  Haiku → Sonnet, Sonnet stays Sonnet
Attempt 2+: Sonnet → Opus, Opus stays Opus
```

Unknown model IDs are returned unchanged across all retry attempts.

---

## 8. Provider & Model Registry

**Source:** `internal/realize/orchestrator/models.go`

### Claude

| Tier | Model ID | Notes |
|------|----------|-------|
| Haiku | `claude-haiku-4-5-20251001` | Default for boilerplate tasks |
| Sonnet | `claude-sonnet-4-6` | Default for most tasks |
| Opus | `claude-opus-4-6` | Escalation target; also `DefaultModel` constant |

Provider default: `claude-opus-4-6`

### ChatGPT (OpenAI)

| Tier | Version | Model ID |
|------|---------|----------|
| Mini | `o3-mini` | `o3-mini` |
| Mini | *(default)* | `gpt-4o-mini` |
| 4o | `4o-2024` | `gpt-4o-2024-11-20` |
| 4o | *(default)* | `gpt-4o` |
| o1 | `o1-preview` | `o1-preview` |
| o1 | *(default)* | `o1` |

Provider default (no tier matched): `gpt-4o`

### Gemini

| Tier | Version | Model ID |
|------|---------|----------|
| Flash | `1.5` | `gemini-1.5-flash` |
| Flash | *(default)* | `gemini-2.0-flash` |
| Pro | `1.5` | `gemini-1.5-pro` |
| Pro | *(default)* | `gemini-2.0-pro-exp` |
| Ultra | *(default)* | `gemini-ultra` |

Provider default: `gemini-2.0-flash`

### Mistral

| Tier | Version | Model ID |
|------|---------|----------|
| Nemo | *(default)* | `open-mistral-nemo` |
| Small | `3.0` | `mistral-small-2402` |
| Small | *(default)* | `mistral-small-2409` |
| Large | `2.0` | `mistral-large-2407` |
| Large | *(default)* | `mistral-large-2411` |

Provider default: `mistral-large-2411`. API base: `https://api.mistral.ai`

### Llama (via Groq)

| Tier | Version | Model ID |
|------|---------|----------|
| 8B | `3.1` | `llama-3.1-8b-instant` |
| 8B | *(default)* | `llama-3.2-8b-preview` |
| 70B | `3.1` | `llama-3.1-70b-versatile` |
| 70B | *(default)* | `llama-3.3-70b-versatile` |
| 405B | *(default)* | `llama-3.1-405b-reasoning` |

Provider default: `llama-3.3-70b-versatile`. API base: `https://api.groq.com/openai`

---

## 9. Section Model Overrides

**Source:** `internal/realize/orchestrator/models.go` — `buildProviderAssignments()`

`section_models` in the manifest maps a **section ID** to a `"Provider · Tier"` string.

```json
"section_models": {
  "backend":   "Claude · Sonnet",
  "data":      "ChatGPT · 4o",
  "contracts": "Gemini · Flash",
  "frontend":  "Claude · Opus",
  "infra":     "default",
  "crosscut":  "Mistral · Large"
}
```

**Valid section IDs:** `backend`, `data`, `contracts`, `frontend`, `infra`, `crosscut`

**Rules:**
- Value `"default"` or empty string = use the tier-based default agent (no override)
- The referenced provider must have a credential in `configured_providers`; otherwise the override is silently skipped and the default agent is used
- Format must be exactly `"<Provider> · <Tier>"` with the middle dot (`·`) separator
- The version field of the registered provider credential is ignored when a section override is active (the fallback model for that tier is used)

**Section-to-task-ID routing:** Task IDs follow `<section>.<name>` — the section is extracted as everything before the first `.`. So `backend.auth` routes to the `"backend"` section override, `infra.docker` to `"infra"`, etc.

---

## 10. Skills System

**Source:** `internal/realize/skills/`

### Loading

Skills are markdown files (`*.md`) loaded from the directory specified by `-skills` (default: `.vibemenu/skills`). If the directory does not exist, bundled skills are extracted from the embedded filesystem at startup. Each file is truncated to `MaxSkillBytes` (2000 bytes) before being injected into prompts.

### Technology Alias Map

**Source:** `internal/realize/skills/aliases.go` — `aliasMap`

Maps manifest technology names to skill file base names (without `.md` extension):

| Manifest Name | Skill File |
|---------------|-----------|
| **Go frameworks** | |
| `Go` | `golang-patterns` |
| `Fiber` | `go-fiber` |
| `Gin` | `go-gin` |
| `Echo` | `go-echo-chi` |
| `Chi` | `go-echo-chi` |
| **TypeScript / Node** | |
| `TypeScript` | `coding-standards` |
| `JavaScript` | `coding-standards` |
| `Express` | `node-express` |
| `Fastify` | `node-fastify` |
| `NestJS` | `node-nestjs` |
| `Hono` | `node-hono-elysia` |
| **Python** | |
| `Python` | `python-patterns` |
| `FastAPI` | `python-fastapi` |
| `Django` | `python-django` |
| `Flask` | `python-flask-litestar` |
| **Java / Spring** | |
| `Java` | `java-coding-standards` |
| `Spring Boot` | `java-spring-boot` |
| `JPA` | `jpa-patterns` |
| **Kotlin / Android / KMP** | |
| `Kotlin` | `kotlin-patterns` |
| `Ktor` | `kotlin-ktor` |
| `Android` | `android-clean-architecture` |
| `Kotlin Multiplatform` | `compose-multiplatform` |
| **Swift / iOS** | |
| `Swift` / `SwiftUI` / `iOS` | `swiftui` |
| **Native / Systems** | |
| `Perl` | `perl-patterns` |
| `C++` / `C` | `cpp-coding-standards` |
| `Axum` | `rust-axum` |
| `Actix-web` | `rust-actix` |
| **Frontend frameworks** | |
| `React` | `frontend-patterns` |
| `Next.js` | `react-nextjs` |
| `Vue` | `frontend-patterns` |
| `Nuxt.js` | `vue-nuxt` |
| `Svelte` | `frontend-patterns` |
| `SvelteKit` | `svelte-kit` |
| `Angular` | `frontend-patterns` |
| `Flutter` | `flutter` |
| `React Native` | `react-native` |
| **Databases** | |
| `PostgreSQL` / `Postgres` | `postgres-patterns` |
| `MySQL` | `db-mysql-mariadb` |
| `MongoDB` | `db-mongodb-couchdb` |
| `Redis` | `db-redis-memcached` |
| `DynamoDB` | `db-dynamodb` |
| `SQLite` | `db-sqlite` |
| `Valkey` | `db-valkey` |
| **Message Brokers** | |
| `Kafka` | `broker-kafka` |
| `RabbitMQ` | `broker-rabbitmq` |
| `NATS` | `broker-nats` |
| `AWS SQS/SNS` | `broker-cloud` |
| **Job Queues** | |
| `Bull` / `BullMQ` | `jobs-bullmq` |
| `Temporal` | `jobs-temporal` |
| `Sidekiq` / `Celery` | `jobs-sidekiq-celery` |
| **Auth Strategies** | |
| `JWT` | `auth-jwt-stateless` |
| `Session-based` | `auth-session-based` |
| `OIDC` / `SAML` | `auth-oauth2-oidc` |
| `API Key` | `auth-apikey` |
| `mTLS` | `auth-mtls` |
| **File Storage** | |
| `S3` / `GCS` | `storage-s3-gcs` |
| `MinIO` | `storage-minio` |
| `Azure Blob` | `storage-azure-blob` |
| `R2` | `storage-r2` |
| **Styling** | |
| `Tailwind CSS` / `Tailwind` | `tailwind` |
| **Infrastructure** | |
| `Docker` | `docker-patterns` |
| `Terraform` | `terraform` |
| `Terraform (AWS)` | `terraform-aws` |
| `Pulumi` | `pulumi` |
| `GitHub Actions` | `github-actions` |
| `GitLab CI` | `gitlab-ci` |
| **Auth Providers** | |
| `Auth0` | `auth0` |
| `Clerk` | `clerk` |
| `Cognito` | `aws-cognito` |
| **Go DB drivers** | |
| `pgx` / `pgxv5` / `pgxmock` | `go-pgx-repository` |
| **Testing** | |
| `Jest` / `Vitest` / `pytest` / `JUnit` | `test-unit` |
| `Playwright` / `Cypress` | `test-playwright-cypress` |
| `k6` / `Locust` | `test-load` |
| `Pact` | `test-contract` |

### Universal Skills Per Task Kind

Always injected regardless of which technologies appear in the manifest:

| Task Kind | Universal Skills |
|-----------|-----------------|
| `backend.service.plan` | `backend-patterns`, `go-pgx-repository` |
| `backend.service.repository` | `backend-patterns`, `coding-standards`, `go-pgx-repository`, `go-caching-impl` |
| `backend.service.logic` | `backend-patterns`, `coding-standards`, `go-background-workers` |
| `backend.service.handler` | `backend-patterns`, `security-review`, `api-design`, `coding-standards`, `pagination-impl`, `api-versioning-impl` |
| `backend.service.bootstrap` | `backend-patterns`, `coding-standards`, `file-storage-patterns` |
| `backend.auth` | `security-review`, `coding-standards`, `security-scan` |
| `data.schemas` | `database-migrations`, `multi-tenancy` |
| `data.migrations` | `database-migrations`, `postgres-patterns`, `multi-tenancy` |
| `backend.messaging` | `backend-patterns`, `coding-standards` |
| `backend.gateway` | `backend-patterns`, `security-review`, `api-design`, `grpc-gateway` |
| `contracts` | `api-design`, `coding-standards`, `domain-dtos` |
| `frontend` | `frontend-patterns`, `coding-standards`, `frontend-bundle-optimization`, `frontend-error-boundaries`, `frontend-realtime-client` |
| `infra.docker` | `docker-patterns`, `deployment-patterns` |
| `infra.terraform` | `deployment-patterns` |
| `infra.cicd` | `deployment-patterns`, `verification-loop` |
| `crosscut.testing` | `tdd-workflow`, `e2e-testing`, `coding-standards`, `verification-loop` |
| `crosscut.docs` | `api-design`, `coding-standards` |

---

## 11. Agent Role Descriptions

**Source:** `internal/realize/agent/roles.go`

Each task kind is given a specific role that scopes the agent's output. Key constraints per task:

### `backend.service.plan`
**Role:** Go software architect
**Output:** `go.mod`, `internal/repository/interfaces.go`, `internal/domain/errors.go`
**CRITICAL:**
- No implementation code — interfaces only
- Only direct (first-party) dependencies in `go.mod`; `go mod tidy` resolves transitive deps
- Use exact module paths/versions from the Dependency & API Reference section
- Repository interfaces are the binding contract for all downstream layers
- For PostgreSQL: define `PgxPool` interface (Exec, Query, QueryRow, SendBatch, Begin) in `interfaces.go`

### `data.schemas`
**Role:** Expert Go developer
**Output:** `internal/domain/*.go` (one file per entity)
**CRITICAL:** No DB logic, no HTTP — pure struct types, input/update structs, domain errors only

### `data.migrations`
**Role:** Expert database engineer
**Output:** `db/migrations/*.sql` (up/down pairs)

### `backend.service.repository`
**Role:** Expert Go backend engineer
**Output:** `internal/repository/interfaces.go`, `internal/repository/postgres/<entity>_repository.go`, `internal/repository/postgres/db.go`, `*_test.go` files
**CRITICAL:** Implement only the interfaces from the plan task; use the `PgxPool` interface, never `*pgxpool.Pool` directly

### `backend.service.logic`
**Role:** Expert Go backend engineer
**Output:** `internal/service/*.go`, `*_test.go`
**CRITICAL:** Accept repo interfaces via constructor DI; no HTTP code

### `backend.service.handler`
**Role:** Expert Go backend engineer
**Output:** `internal/handler/*.go`, `internal/router/router.go`, `internal/middleware/auth.go`, `*_test.go`
**CRITICAL:** Auth middleware injection, route registration

### `backend.service.bootstrap`
**Role:** Expert Go backend engineer
**Output:** `main.go`, `.env.example`
**CRITICAL:** Do NOT generate `go.mod` / `go.sum` — the locked dependency tree from the deps phase must not be overwritten

### `backend.auth`
**Role:** Expert security engineer
Authentication/authorization middleware, JWT handling, identity integration

### `backend.messaging`
**Role:** Expert distributed systems engineer
Message broker configuration, event producer/consumer boilerplate, event schemas

### `backend.gateway`
**Role:** Expert platform engineer
API gateway routing rules, rate limiting, middleware configuration

### `contracts`
**Role:** Expert API designer
DTO types, request/response models, OpenAPI specification

### `frontend`
**Role:** Expert frontend engineer
**CRITICAL:**
- Use `next.config.mjs` (NOT `next.config.ts`)
- Tailwind CSS v4: use `@tailwindcss/postcss` plugin in `postcss.config.mjs`, NOT the old `tailwindcss` plugin
- Pin all package versions from the Infrastructure & Dependency Reference

### `infra.docker`
**Role:** Expert DevOps engineer
**Output:** `Dockerfile`(s), `docker-compose.yml`, `.air.toml`, `Makefile`, `nginx.conf`, `.dockerignore`, setup scripts
**CRITICAL:**
- Use `service_dirs` from the payload for build contexts — do not invent paths
- Go Dockerfile layer order: `FROM golang:<version>-alpine` → `RUN go install <air>@<version>` → `COPY go.mod go.sum ./` → `RUN go mod download` → `COPY . .` → `CMD ["air", ...]`
- Air module path: `github.com/air-verse/air` (NEVER `github.com/cosmtrek/air`)
- Node: `FROM node:20-alpine`; use `npm install` NOT `npm ci`
- Do NOT generate any Go or TypeScript source files

### `infra.terraform`
**Role:** Expert infrastructure engineer
IaC configuration (Terraform/Pulumi) for all cloud resources

### `infra.cicd`
**Role:** Expert DevOps engineer
CI/CD pipeline configuration (build, test, deploy stages)

### `crosscut.testing`
**Role:** Expert test engineer
Test scaffolding: unit tests (table-driven), integration tests, E2E setup; target 80%+ coverage on business logic using RED-GREEN-REFACTOR TDD cycle

### `crosscut.docs`
**Role:** Expert technical writer
API documentation, OpenAPI specs, changelog files

### `backend.reconciliation`
**Role:** Code reviewer / Go compiler expert
Cross-task compilation fix phase — resolves compile errors that arise from type/interface mismatches between independently generated service layers

### `integration.repair`
**Role:** Go compiler expert
Post-pipeline integration repair — fixes cross-file compilation errors across the entire generated codebase

---

## 12. Dependency Resolution

**Source:** `internal/realize/deps/`

### Language Detection

The dep resolver is selected by the service's declared language:

| Manifest Language | Resolver |
|-------------------|----------|
| `Go` | `go mod tidy` |
| `TypeScript`, `JavaScript`, `Node.js`, `TypeScript/Node` | `npm install --package-lock-only` |
| `Python` | `pip-compile requirements.in → requirements.txt` |
| All others | Skipped |

### Go — Well-Known Modules

**Source:** `internal/realize/deps/go_modules.go` — `WellKnownGoModules`

Versions are fallbacks; `ResolveGoModuleVersions()` fetches the latest from `proxy.golang.org` at pipeline startup.

| Key | Module Path | Fallback Version | Test Deps |
|-----|-------------|------------------|-----------|
| **Web frameworks** | | | |
| `Fiber` | `github.com/gofiber/fiber/v2` | `v2.52.5` | `github.com/stretchr/testify v1.9.0` |
| `Gin` | `github.com/gin-gonic/gin` | `v1.10.0` | |
| `Echo` | `github.com/labstack/echo/v4` | `v4.12.0` | |
| `Chi` | `github.com/go-chi/chi/v5` | `v5.1.0` | |
| **Database drivers** | | | |
| `pgx` | `github.com/jackc/pgx/v5` | `v5.7.2` | `github.com/pashagolub/pgxmock/v4 v4.4.0` |
| `PostgreSQL` | `github.com/jackc/pgx/v5` | `v5.7.2` | |
| `MySQL` | `github.com/go-sql-driver/mysql` | `v1.8.1` | |
| `SQLite` | `modernc.org/sqlite` | `v1.34.4` | |
| `MongoDB` | `go.mongodb.org/mongo-driver` | `v1.17.1` | |
| `Redis` | `github.com/redis/go-redis/v9` | `v9.7.3` | |
| `sqlx` | `github.com/jmoiron/sqlx` | `v1.4.0` | |
| `CockroachDB` | `github.com/jackc/pgx/v5` | `v5.7.2` | |
| **ORM** | | | |
| `GORM` | `gorm.io/gorm` | `v1.25.12` | |
| `gorm-postgres` | `gorm.io/driver/postgres` | `v1.5.11` | |
| `gorm-mysql` | `gorm.io/driver/mysql` | `v1.5.7` | |
| `gorm-sqlite` | `gorm.io/driver/sqlite` | `v1.5.7` | |
| `ent` | `entgo.io/ent` | `v0.14.1` | |
| **Messaging** | | | |
| `NATS` | `github.com/nats-io/nats.go` | `v1.37.0` | |
| `Kafka` | `github.com/segmentio/kafka-go` | `v0.4.47` | |
| `RabbitMQ` | `github.com/rabbitmq/amqp091-go` | `v1.10.0` | |
| **RPC / API** | | | |
| `gRPC` | `google.golang.org/grpc` | `v1.70.0` | |
| `protobuf` | `google.golang.org/protobuf` | `v1.36.5` | |
| `ConnectRPC` | `connectrpc.com/connect` | `v1.18.1` | |
| **Auth** | | | |
| `JWT` | `github.com/golang-jwt/jwt/v5` | `v5.2.1` | |
| `bcrypt` | `golang.org/x/crypto` | `v0.31.0` | |
| **Testing** | | | |
| `testify` | `github.com/stretchr/testify` | `v1.9.0` | |
| `pgxmock` | `github.com/pashagolub/pgxmock/v4` | `v4.4.0` | |
| `gomock` | `go.uber.org/mock` | `v0.5.0` | |
| `httptest` | `net/http/httptest` | *(stdlib)* | |
| **Validation** | | | |
| `validator` | `github.com/go-playground/validator/v10` | `v10.22.1` | |
| **Logging** | | | |
| `zap` | `go.uber.org/zap` | `v1.27.0` | |
| `zerolog` | `github.com/rs/zerolog` | `v1.33.0` | |
| `slog` | `log/slog` | *(stdlib)* | |
| **Observability** | | | |
| `prometheus` | `github.com/prometheus/client_golang` | `v1.21.1` | |
| `otel` | `go.opentelemetry.io/otel` | `v1.34.0` | |
| `otel-http` | `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | `v1.34.0` | |
| **UUID** | | | |
| `uuid` | `github.com/google/uuid` | `v1.6.0` | |
| **Config** | | | |
| `envconfig` | `github.com/kelseyhightower/envconfig` | `v1.4.0` | |
| `viper` | `github.com/spf13/viper` | `v1.19.0` | |
| `godotenv` | `github.com/joho/godotenv` | `v1.5.1` | |
| **Scheduling** | | | |
| `cron` | `github.com/robfig/cron/v3` | `v3.0.1` | |
| **HTTP routing** | | | |
| `gorilla/mux` | `github.com/gorilla/mux` | `v1.8.1` | |
| **Serialisation** | | | |
| `sonic` | `github.com/bytedance/sonic` | `v1.13.2` | |

Every service's `go.mod` automatically includes `testify` and `uuid` regardless of the framework.

### Go — Dev Tools (Dockerfiles Only)

**Source:** `internal/realize/deps/go_modules.go` — `WellKnownGoDevTools`

These are installed via `go install` in Docker images, **not** added to `go.mod`.

| Name | Module Path | Fallback Version | Min Go |
|------|-------------|------------------|--------|
| `air` | `github.com/air-verse/air` | `v1.61.5` | `1.23` |

> **Warning:** The old module path `github.com/cosmtrek/air` was renamed. Never use it.

The actual version and `MinGoVersion` are resolved from `proxy.golang.org` at pipeline startup; the table above shows fallbacks used when the proxy is unreachable.

### TypeScript / npm — Well-Known Packages

**Source:** `internal/realize/deps/npm_modules.go` — `WellKnownNpmPackages`

Versions are fallbacks; `resolveAllNpmVersions()` fetches the latest from `registry.npmjs.org` at pipeline startup.

| Package | Fallback Version |
|---------|-----------------|
| `next` | `15.3.0` |
| `react` | `19.1.0` |
| `react-dom` | `19.1.0` |
| `typescript` | `5.7.2` |
| `@types/react` | `19.1.0` |
| `@types/react-dom` | `19.1.0` |
| `@types/node` | `22.10.0` |
| `tailwindcss` | `3.4.17` |
| `@tailwindcss/postcss` | `4.0.17` |
| `postcss` | `8.5.1` |
| `autoprefixer` | `10.4.20` |
| `eslint` | `9.17.0` |
| `eslint-config-next` | `15.3.0` |
| `axios` | `1.7.9` |
| `@tanstack/react-query` | `5.62.3` |
| `zustand` | `5.0.2` |
| `zod` | `3.24.1` |
| `react-hook-form` | `7.54.2` |
| `@hookform/resolvers` | `3.9.1` |
| `lucide-react` | `0.468.0` |
| `clsx` | `2.1.1` |
| `tailwind-merge` | `2.5.5` |
| `vue` | `3.5.13` |
| `nuxt` | `3.16.1` |
| `@angular/core` | `19.2.5` |
| `express` | `4.21.2` |
| `fastify` | `5.3.2` |
| `@nestjs/core` | `11.1.0` |
| `hono` | `4.7.5` |
| `@trpc/server` | `11.1.0` |
| `prisma` | `6.5.0` |
| `drizzle-orm` | `0.41.0` |
| `typeorm` | `0.3.21` |
| `next-auth` | `4.24.11` |

> **Tailwind CSS v4 note:** For Tailwind v4, use `@tailwindcss/postcss` plugin in `postcss.config.mjs` instead of the old `tailwindcss` plugin. Both v3 and v4 packages are tracked.

### Python — Dependency Resolution

1. Agent generates `requirements.in` with direct dependencies
2. `pip-compile requirements.in` resolves and pins all transitive deps into `requirements.txt`

---

## 13. Verification System

**Source:** `internal/realize/verify/`

Verification is run after every agent call (first attempt and each retry) when `verify: true` in the manifest.

### Language-to-Verifier Mapping

| Language | Verifier | Detection |
|----------|----------|-----------|
| Go | `GoVerifier` | `service.Language == "Go"` |
| TypeScript | `TsVerifier` | `service.Language == "TypeScript"`, `"JavaScript"`, `"Node.js"` |
| Python | `PythonVerifier` | `service.Language == "Python"` |
| Terraform | `TfVerifier` | Task kind is `infra.terraform` |
| *(all others)* | `NullVerifier` | Always passes |

### Go Verifier (`go_verifier.go`)

Operates on each directory containing a `go.mod` among the generated files.

```
1. go mod tidy         — resolve/update dependencies
2. go build ./...      — compile (skips vet/test/fmt if this fails)
3. go vet ./...        — static analysis
4. go test ./...       — run all tests
5. gofmt -l .          — check formatting (fails if any file listed)
```

**Special behaviour:**
- Detects `invalid version` and `404 Not Found` patterns in `go mod tidy` output and emits actionable `replace` directive suggestions
- All 5 steps run sequentially; `go vet`, `go test`, and `gofmt` are skipped if `go build` fails

### TypeScript Verifier (`ts_verifier.go`)

```
1. Check tsc in PATH   — graceful degrade: pass if not installed
2. Find tsconfig.json  — graceful degrade: pass if none found
3. tsc --noEmit        — type-check without emitting files
```

### Python Verifier (`python_verifier.go`)

```
1. Find *.py files     — pass if none found
2. Check ruff in PATH  — graceful degrade: skip check if not installed
3. ruff check .        — lint + style check
```

### Terraform Verifier (`tf_verifier.go`)

```
1. Check terraform in PATH              — graceful degrade: pass if not installed
2. Find *.tf files                      — pass if none found
3. terraform init -backend=false        — initialize providers without remote state
4. terraform validate                   — validate configuration syntax
```

---

## 14. Deterministic Fixes

**Source:** `internal/realize/verify/deterministic_fixes.go`

Applied **before every verification attempt** (attempt 0 and all retries) without consuming a retry slot. Only applies to Go files.

### `fixGoEscapeSequences`

Scans double-quoted string literals for invalid Go escape sequences (e.g. `\p`, `\s`, `\d`). Rewrites affected strings as raw string literals (`` `...` ``). Skips existing raw strings and comments.

### `fixDuplicateTypes`

Detects `type Foo struct/interface` declarations that appear in multiple files within the same package. Keeps the declaration in the file with the most type declarations; removes it from all others.

### `fixGofmt`

Runs `gofmt -w` on every `.go` file in the generated output. Reports how many files were changed.

---

## 15. Retry & Error Classification

**Source:** `internal/realize/orchestrator/runner.go`

### Error Types

On each retry, the previous verification output is inspected to choose a fix strategy:

| Error Type | Detection Pattern | Fix Strategy |
|------------|------------------|--------------|
| `errTypeEscape` | `"unknown escape sequence"` | Deterministic: `fixGoEscapeSequences` |
| `errTypeGofmt` | `"files not gofmt-clean"` (no `undefined:` or `FAIL`) | Deterministic: `fixGofmt` |
| `errTypeDeps` | `"missing go.sum entry"` / `"invalid version"` / `"cannot find module"` | Locked `go.mod` restoration + `go mod tidy`; LLM retry if still failing |
| `errTypeDuplicate` | `"redeclared in this block"` | Deterministic: `fixDuplicateTypes` |
| `errTypeTestFail` | `"--- FAIL:"` | LLM retry with test output injected |
| `errTypeUndefined` | `"undefined:"` / `"does not implement"` | LLM retry with targeted guidance |
| `errTypeUnknown` | *(none of the above)* | LLM retry |

### Locked `go.mod` Restoration Flow

When the deps task (`backend.service.deps`) has produced a locked `go.mod`:

1. On deps-type error: restore the locked `go.mod` and re-run `go mod tidy`
2. If verification passes after restoration → commit files, skip LLM retry
3. If still failing → proceed to LLM retry with escalated model

### `TryFix` In-Memory Fix Layer

After a failed attempt, the system tries applying the deterministic fixes a second time (in addition to the pre-attempt application) using `fixer.TryFix()`. If verification passes after this:
- Files are committed; the retry slot is **not** consumed

### Rate-Limit Backoff

When an agent call returns HTTP 429 (rate limit):

```
wait = (attempt + 1) × RateLimitBackoffBase seconds
     = (attempt + 1) × 60 seconds
```

### Debug Logging

Every attempt's verification output is appended to `.realize/debug/<task-id>.log` in the output directory.
- Failures: always logged
- Successes: logged only when `-verbose` is set

---

## 16. Task Payload Fields

**Source:** `internal/realize/dag/payload.go`

The `TaskPayload` struct is serialised to JSON and injected into each agent's user message. Only fields relevant to the task's kind are populated; all others are omitted (`omitempty`).

| Field | Type | Populated For |
|-------|------|---------------|
| `module_path` | string | All service-chain tasks (e.g. `"core-api"`) |
| `arch_pattern` | ArchPattern | All tasks |
| `env_config` | EnvConfig | All tasks |
| `domains` | []DomainDef | Data, backend, contracts tasks |
| `databases` | []DBSourceDef | Data tasks |
| `cachings` | []CachingConfig | Data tasks |
| `file_storages` | []FileStorageDef | Data tasks |
| `service` | *ServiceDef | Per-service tasks (single service context) |
| `all_services` | []ServiceDef | Cross-service tasks (auth, contracts, infra) |
| `comm_links` | []CommLink | Backend tasks |
| `messaging` | *MessagingConfig | `backend.messaging` task |
| `api_gateway` | *APIGatewayConfig | `backend.gateway` task |
| `auth` | *AuthConfig | `backend.auth` task |
| `dtos` | []DTODef | `contracts` task |
| `endpoints` | []EndpointDef | `contracts` task |
| `versioning` | APIVersioning | `contracts` task |
| `frontend` | *FrontendPillar | `frontend` task |
| `service_dirs` | map[string]string | `infra.docker` task (build context mapping) |
| `output_dir` | string | All tasks (relative path within output root) |
| `infra` | *InfraPillar | Infrastructure tasks |
| `cross_cut` | *CrossCutPillar | `crosscut.testing`, `crosscut.docs` tasks |

**Module path derivation:** `svcSlug(name)` = lowercase name with spaces replaced by hyphens. Example: `"Core API"` → `"core-api"`.

---

## 17. Progress State & Resume

**Source:** `internal/realize/state/state.go`

The orchestrator tracks completed tasks in:

```
<output_dir>/.realize-progress.json
```

On each successful task completion, the task ID is persisted. On a subsequent run of `cmd/realize` against the same output directory, tasks already present in `.realize-progress.json` are skipped with an `"already completed"` log message. This enables safe resume after partial failures or interruptions without re-running successful tasks.

---

## 18. Wave Parallelism

**Source:** `internal/realize/dag/dag.go` — `Levels()`

The DAG is partitioned into **waves** (topologically sorted levels):
- All tasks in wave `N` can execute concurrently
- Wave `N+1` starts only after all tasks in wave `N` have completed or failed
- Concurrency within a wave is bounded by the `-parallel` CLI flag

Wave assignment:
```
level(task) = max(level(dep) for dep in task.Dependencies) + 1
```

Tasks with no dependencies are in wave 0. The service chain (`plan → deps → repository → logic → handler → bootstrap`) for each service forms a linear sequence spanning 6 waves.
