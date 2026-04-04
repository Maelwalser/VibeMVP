# Missing Skills Tracker

## Overview

This document catalogs every gap between the configuration options supported by the VibeMenu manifest and the skill coverage in `.vibemenu/skills/`. It is structured as a prioritised checklist for developers.

There are two distinct problem types:

- **Part A — Alias routing fixes**: Skill files already exist on disk but are silently unreachable because `internal/realize/skills/aliases.go` maps manifest values to the wrong key. These are zero-effort wins — fix the string, gain the skill.
- **Part B — New skill files needed**: Configuration options that have no actionable skill at all. Each entry describes what to write and where.

There is also a **Part C** covering a pipeline-level code gap that blocks an entire category of skills regardless of file content.

---

## Part A — Alias Routing Fixes (`internal/realize/skills/aliases.go`)

### A1. Wrong target key — file exists under a different name (22 entries)

The alias map value does not match the actual filename stem. The skill loader tries to open `<value>.md` and gets a miss. Fix: update the map value to match the real filename stem.

| Manifest value | Current alias target | Correct alias target | File on disk |
|---|---|---|---|
| `Echo` | `go-echo` | `go-echo-chi` | `go-echo-chi.md` |
| `Chi` | `go-chi` | `go-echo-chi` | `go-echo-chi.md` |
| `Express` | `typescript-express` | `node-express` | `node-express.md` |
| `Fastify` | `typescript-fastify` | `node-fastify` | `node-fastify.md` |
| `NestJS` | `typescript-nestjs` | `node-nestjs` | `node-nestjs.md` |
| `Hono` | `typescript-hono` | `node-hono-elysia` | `node-hono-elysia.md` |
| `Flask` | `python-flask` | `python-flask-litestar` | `python-flask-litestar.md` |
| `Actix-web` | `rust-actix` | `rust-actix-web` | `rust-actix-web.md` |
| `Nuxt.js` | `vue-nuxt` | `web-vue-nuxt` | `web-vue-nuxt.md` |
| `SvelteKit` | `svelte-kit` | `web-svelte-sveltekit` | `web-svelte-sveltekit.md` |
| `Flutter` | `flutter` | `mobile-flutter` | `mobile-flutter.md` |
| `React Native` | `react-native` | `mobile-react-native-expo` | `mobile-react-native-expo.md` |
| `MySQL` | `mysql` | `db-mysql-mariadb` | `db-mysql-mariadb.md` |
| `MongoDB` | `mongodb` | `db-mongodb-couchdb` | `db-mongodb-couchdb.md` |
| `Redis` | `redis` | `db-redis-memcached` | `db-redis-memcached.md` |
| `Terraform` | `terraform` | `iac-terraform-pulumi` | `iac-terraform-pulumi.md` |
| `Pulumi` | `pulumi` | `iac-terraform-pulumi` | `iac-terraform-pulumi.md` |
| `DynamoDB` | `dynamodb` | `db-dynamodb` | `db-dynamodb.md` |
| `SQLite` | `sqlite` | `db-sqlite` | `db-sqlite.md` |
| `Kafka` | `kafka` | `broker-kafka` | `broker-kafka.md` |
| `RabbitMQ` | `rabbitmq` | `broker-rabbitmq` | `broker-rabbitmq.md` |
| `NATS` | `nats` | `broker-nats` | `broker-nats.md` |

### A2. Dead aliases — no file exists at all (6 entries)

The alias map entry points to a key for which no `.md` file exists anywhere in the skills directory. Recommended fix is to redirect to the nearest existing file rather than create net-new files for these.

| Manifest value | Dead alias target | Recommended redirect | Rationale |
|---|---|---|---|
| `GitLab CI` | `gitlab-ci` | `cicd-pipelines` | `cicd-pipelines.md` covers pipeline stages generically; GitLab-specific syntax is minor |
| `Terraform (AWS)` | `terraform-aws` | `iac-terraform-pulumi` | Existing file covers AWS provider patterns |
| `AWS SQS/SNS` | `aws-sqs-sns` | `broker-cloud` | `broker-cloud.md` covers SQS/SNS/Pub-Sub |
| `Auth0` | `auth0` | `idp-integrations` | `idp-integrations.md` covers Auth0, Okta, social login |
| `Clerk` | `clerk` | `idp-integrations` | Same file; Clerk is an OIDC identity provider |
| `Cognito` | `aws-cognito` | `idp-integrations` | Same file; Cognito is a managed IDP |

### A3. Language or framework with no alias entry at all (19 entries)

These technology values can appear in the manifest but have zero routing in `aliasMap`. The corresponding skill files exist on disk — only the alias entries are missing.

**PHP** (5 entries — `PHP` language key + 4 frameworks):

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `PHP` | `php-laravel-symfony` | `php-laravel-symfony.md` |
| `Laravel` | `php-laravel-symfony` | `php-laravel-symfony.md` |
| `Symfony` | `php-laravel-symfony` | `php-laravel-symfony.md` |
| `Slim` | `php-slim` | `php-slim.md` |
| `Laminas` | `php-laminas` | `php-laminas.md` |

**Ruby** (5 entries — `Ruby` language key + 4 frameworks):

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `Ruby` | `ruby-rails-sinatra` | `ruby-rails-sinatra.md` |
| `Rails` | `ruby-rails-sinatra` | `ruby-rails-sinatra.md` |
| `Sinatra` | `ruby-rails-sinatra` | `ruby-rails-sinatra.md` |
| `Hanami` | `ruby-hanami` | `ruby-hanami.md` |
| `Roda` | `ruby-roda` | `ruby-roda.md` |

**Elixir** (4 entries — `Elixir` language key + 3 frameworks):

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `Elixir` | `elixir-phoenix` | `elixir-phoenix.md` |
| `Phoenix` | `elixir-phoenix` | `elixir-phoenix.md` |
| `Plug` | `elixir-plug-bandit` | `elixir-plug-bandit.md` |
| `Bandit` | `elixir-plug-bandit` | `elixir-plug-bandit.md` |

**Rust language key + unaliased frameworks** (3 entries):

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `Rust` | `rust-axum` | `rust-axum.md` |
| `Rocket` | `rust-rocket-warp` | `rust-rocket-warp.md` |
| `Warp` | `rust-rocket-warp` | `rust-rocket-warp.md` |

**Desktop** (2 entries):

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `Tauri` | `desktop-tauri-electron` | `desktop-tauri-electron.md` |
| `Electron` | `desktop-tauri-electron` | `desktop-tauri-electron.md` |

---

## Part B — New Skill Files Needed

These configuration options are fully wired in the manifest but have no actionable skill content to guide the generation agent. Each entry specifies: the suggested filename, which manifest field triggers it, which task kinds should receive it, and what the skill must cover.

### B1. Job Queues — alias routing only (4 files exist, aliases missing)

Four job queue skill files are on disk but have no alias entries. They are completely unreachable despite being directly relevant to `BackendPillar.JobQueues[*].Technology`.

Add these alias entries to `aliasMap`:

| Manifest value | Add alias → | File on disk |
|---|---|---|
| `BullMQ` | `jobs-bullmq` | `jobs-bullmq.md` |
| `Bull` | `jobs-bullmq` | `jobs-bullmq.md` |
| `Temporal` | `jobs-temporal` | `jobs-temporal.md` |
| `Sidekiq` | `jobs-sidekiq-celery` | `jobs-sidekiq-celery.md` |
| `Celery` | `jobs-sidekiq-celery` | `jobs-sidekiq-celery.md` |
| `Dramatiq` | `jobs-sidekiq-celery` | `jobs-sidekiq-celery.md` (closest match) |
| `Faktory` | `jobs-faktory-asynq-river` | `jobs-faktory-asynq-river.md` |
| `Asynq` | `jobs-faktory-asynq-river` | `jobs-faktory-asynq-river.md` |
| `River` | `jobs-faktory-asynq-river` | `jobs-faktory-asynq-river.md` |

Technologies still needing new files (no existing skill is close enough):

- `Hangfire` → needs `jobs-hangfire.md` (.NET background job framework)
- `Laravel Queues` → needs `jobs-laravel-queues.md` (PHP native queue driver)
- `Oban` → needs `jobs-oban.md` (Elixir/Ecto job processing)

Also add job queue skills to `universalSkillsForKind[TaskKindServiceBootstrap]` when the service has job queues in its payload (see Part C for why this requires a payload fix first).

### B2. New file: `go-background-workers.md`

**Manifest field**: any Go service with `BackendPillar.JobQueues` entries  
**Task kinds**: `TaskKindServiceBootstrap`, `TaskKindServiceLogic`  
**Content to cover**:
- Goroutine worker pool with fixed pool size and `chan Job` input queue
- Graceful shutdown: `context.Context` cancellation + `sync.WaitGroup` drain before exit
- Buffered vs. unbuffered channel tradeoffs for back-pressure
- Exponential backoff retry with jitter before dead-letter handoff
- Dead-letter queue integration pattern (write failed jobs to a separate queue/table)
- `slog`-structured logging per job with job ID, attempt count, and error fields

### B3. New file: `api-versioning-impl.md`

**Manifest field**: `ContractsPillar.Versioning` (`APIVersioning` struct — `Strategy`, `CurrentVersion`, `DeprecationPolicy`)  
**Task kinds**: `TaskKindServiceHandler`, `TaskKindContracts`  
**Content to cover**:
- URL path versioning: router group setup (`/v1/`, `/v2/`) in Go (Fiber/Gin/Echo/Chi), Node (Express/Fastify), Python (FastAPI)
- Header-based versioning: `Accept-Version` / `API-Version` middleware that rewrites routing or selects handler variant
- Content negotiation: `Accept: application/vnd.myapp.v2+json` matching
- Deprecation response headers: `Sunset`, `Deprecation`, `Link` (RFC 8594)
- Sunset enforcement: automatic `410 Gone` after deprecation deadline

### B4. New file: `grpc-gateway.md`

**Manifest field**: `APIGatewayConfig.Technology` containing gRPC transcoding (e.g., Envoy + gRPC-Gateway)  
**Task kinds**: `TaskKindGateway`  
**Content to cover**:
- `grpc-gateway` Go library setup: generating HTTP/JSON reverse-proxy from `.proto` annotations
- `google.api.http` annotation syntax for HTTP binding (`get`, `post`, `body`, `additional_bindings`)
- Envoy filter config for gRPC-Web and gRPC-JSON transcoding (`envoy.filters.http.grpc_json_transcoder`)
- Reflection endpoint: `grpc.reflection.v1alpha` for tools like `grpcurl` and Postman
- Shared error mapping: gRPC status codes → HTTP status codes via `google.rpc.Status`
- Streaming endpoint transcoding limitations and workarounds

### B5. New file: `file-storage-patterns.md`

**Manifest field**: `DataPillar.FileStorages` (`FileStorageDef` — `Provider`, `BucketName`, `AccessPattern`, `MaxFileSizeMB`, `AllowedMimeTypes`)  
**Task kinds**: `TaskKindServiceHandler`, `TaskKindServiceLogic`  
**Content to cover**:
- Multipart upload handler: streaming parse to avoid loading entire file in memory
- Presigned URL generation for direct browser-to-storage upload (S3 `PutObject`, GCS `SignedURL`, MinIO equivalent) — never proxy large files through the app server
- Streaming download handler: `io.Copy` to response writer with `Content-Disposition` and correct `Content-Type`
- MIME type validation: check magic bytes (not just extension or declared content-type)
- File size enforcement: reject at handler boundary before storage write
- Virus scanning hook point: pluggable interface called before finalizing storage write
- TUS resumable upload protocol: offset tracking, `Upload-Offset` header, checksum verification for large files

### B6. New file: `go-caching-impl.md`

**Manifest field**: `DataPillar.Cachings` entries with `CacheDB = "Redis"` or `"Valkey"`  
**Task kinds**: `TaskKindServiceRepository`, `TaskKindServiceLogic`  
**Content to cover**:
- `go-redis/v9` client setup: `redis.NewClient`, connection pool sizing, health check ping
- Cache key naming convention: `{service}:{entity}:{id}` with version prefix for easy bulk invalidation
- TTL management: per-entity TTL constants, sliding vs. absolute expiry
- Cache stampede prevention: `golang.org/x/sync/singleflight` to deduplicate concurrent cache misses for the same key
- Two-level cache: `sync.Map` as in-process L1 (TTL via goroutine sweep), Redis as L2
- Invalidation on write: `DEL` or `SET` with new value immediately after DB write in the same transaction context
- Testing with `miniredis`: drop-in Redis server for unit tests without a real Redis instance

### B7. New file: `multi-tenancy.md`

**Manifest field**: `DomainDef` entries with multi-tenant scope, `DataPillar.Governances` with data residency requirements  
**Task kinds**: `TaskKindDataSchemas`, `TaskKindDataMigrations`, `TaskKindServiceRepository`  
**Content to cover**:
- Strategy comparison: row-per-tenant (simplest, most common), schema-per-tenant (isolation without separate DB), database-per-tenant (maximum isolation, highest cost)
- PostgreSQL Row Level Security (RLS): `CREATE POLICY`, `ALTER TABLE ENABLE ROW LEVEL SECURITY`, `SET app.tenant_id = $1` session variable
- Tenant context propagation: store tenant ID in `context.Context` via typed key; extract in repository layer before every query
- Migration strategy for adding RLS to existing tables: non-destructive rollout with `FORCE ROW LEVEL SECURITY`
- Index design: always prefix with `(tenant_id, ...)` — e.g., `CREATE INDEX ON orders (tenant_id, created_at DESC)`
- Bypass policy for admin/migration roles: `BYPASSRLS` privilege or separate superuser connection

### B8. New file: `pagination-impl.md`

**Manifest field**: `ContractsPillar.Endpoints` with list operations, `PaginationStrategy` field on REST endpoints  
**Task kinds**: `TaskKindServiceHandler`, `TaskKindServiceRepository`, `TaskKindContracts`  
**Content to cover**:
- Cursor-based pagination: opaque cursor = base64(JSON{`last_id`, `last_created_at`}); encode on response, decode on next request
- Keyset SQL pattern: `WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC LIMIT $3` — avoids offset scan
- Offset/limit tradeoffs: simple to implement, degrades at high offsets; acceptable for small datasets
- Opt-in total count: `X-Total-Count` response header (never default — expensive `COUNT(*)` for large tables)
- OpenAPI schema for `PaginatedResponse<T>`: `data[]`, `next_cursor`, `has_more`, optional `total`
- Go, Node.js, and Python implementation snippets for each strategy

### B9. New file: `frontend-bundle-optimization.md`

**Manifest field**: `FrontendTechConfig.BundleOptimization`  
**Task kinds**: `TaskKindFrontend`  
**Content to cover**:
- Next.js route-based automatic code splitting: how the App Router splits by segment
- Manual dynamic `import()` for heavy components (charts, editors, modals): `next/dynamic` with `{ ssr: false }` where appropriate
- Tree-shaking configuration: ES module imports, `sideEffects: false` in `package.json`, avoid barrel re-exports that defeat tree-shaking
- Bundle analyzer: `@next/bundle-analyzer` setup in `next.config.js`, how to read the treemap, what to look for (large node_modules, duplicated packages)
- Module federation: Webpack 5 `ModuleFederationPlugin` for micro-frontend code sharing (advanced)
- Lighthouse CI: `.lighthouserc.json` budget file with `maxNumericChange` thresholds; integrate into CI pipeline

### B10. New file: `frontend-error-boundaries.md`

**Manifest field**: `FrontendTechConfig.ErrorBoundary`  
**Task kinds**: `TaskKindFrontend`  
**Content to cover**:
- React class `ErrorBoundary` component: `componentDidCatch` + `getDerivedStateFromError`, fallback UI prop pattern
- Next.js App Router error handling: `error.tsx` (segment-level), `global-error.tsx` (root layout), `not-found.tsx`
- Error reporting integration: call Sentry/custom logger inside `componentDidCatch`; include component stack
- `Suspense` + `ErrorBoundary` composition: wrap data-fetching components so loading and error states are co-located
- Vue equivalent: `errorCaptured` lifecycle hook + `onErrorCaptured` composition API
- Graceful fallback UI patterns: retry button, error ID for support, avoid leaking stack traces to users in production

### B11. New file: `frontend-realtime-client.md`

**Manifest field**: `FrontendTechConfig.RealtimeStrategy`  
**Task kinds**: `TaskKindFrontend`  
**Content to cover**:
- `useWebSocket` React hook: connect on mount, reconnect with exponential backoff + max attempts, clean disconnect on unmount
- Offline message buffer: queue outgoing messages while disconnected, drain on reconnect
- Optimistic update pattern: update local state immediately, roll back on server error acknowledgement
- Pusher JS SDK integration: `Pusher`, `channel.bind`, authentication endpoint setup
- Ably JS SDK integration: `Ably.Realtime`, channel subscribe/publish, connection state events
- `EventSource` for Server-Sent Events: one-way stream, auto-reconnect, `lastEventId` for resumption
- Type-safe message schema sharing: define message types in a shared `contracts/` module imported by both backend handler and frontend hook

### B12. New file: `multi-provider-llm.md`

**Manifest field**: `ConfiguredProviders` / `ProviderAssignments` (Claude, ChatGPT, Gemini, Mistral, Llama)  
**Task kinds**: inject universally into `universalSkillsForKind` for all task kinds  
**Content to cover**:
- Portable system prompt patterns: avoid model-specific directives (`<thinking>`, `tool_use` XML) in shared prompts; use neutral instruction phrasing
- Cost-aware model routing: use Haiku/Flash/Mini for high-frequency low-complexity tasks; escalate to Sonnet/Pro/4o only when verification fails
- Structured output consistency: JSON mode availability differs — Claude uses `response_format: {type: "json_object"}` (some models), Gemini has `responseMimeType`, OpenAI has `response_format: {type: "json_object"}`; use schema validation as the ground truth, not the mode flag
- Fallback chain: primary provider → secondary provider → Opus/Ultra on final retry
- Token budget differences: Claude context window (200k), GPT-4o (128k), Gemini 1.5 Pro (1M) — calibrate prompt verbosity per provider
- Temperature consistency: `temperature: 0` for deterministic code generation across all providers

---

## Part C — Pipeline Gap (code change required, not a skill gap)

`BackendPillar.JobQueues` (and nested `CronJobDef`) is configured in the manifest UI and saved to `manifest.json`, but it is **silently dropped before any agent task receives it**.

The root cause is two-part:

1. **`internal/realize/dag/payload.go`** — `TaskPayload` has no `JobQueues` or `CronJobs` field. The skill loader resolves skills from `payload.Technologies []string`, which is populated from `ServiceDef.Technologies` — not from `JobQueueDef.Technology`. Even if all job queue aliases were fixed, the skill would never be injected.

2. **`internal/realize/dag/builder.go`** — `addServiceTaskChain()` never reads `BackendPillar.JobQueues` when constructing task payloads for service tasks.

**Fix required** (separate from skills work):
- Add `JobQueues []manifest.JobQueueDef` and `CronJobs []manifest.CronJobDef` fields to `TaskPayload`
- In `builder.go`, populate these fields for the service that owns the job queues
- In `loader.go` or `aliases.go`, extend technology resolution to also iterate `payload.JobQueues[*].Technology` when building the skill list

Until this is done, job queue skills have no effect regardless of how many alias entries are added.

---

## Appendix — Counts & Priority Order

| Category | Count |
|---|---|
| Wrong-target aliases (A1) | 22 |
| Dead aliases — file missing (A2) | 6 |
| Language/framework with no alias at all (A3) | 19 |
| Job queue technologies with no alias (B1 partial) | 9 |
| Job queue technologies needing new files (B1 gaps) | 3 |
| New skill files needed (B2–B12) | 11 |
| Pipeline code changes needed (Part C) | 2 files |

### Recommended priority order

1. **Fix all 22 wrong-target aliases** (A1) — zero risk, immediate wins for Echo, Chi, Express, Fastify, NestJS, Flask, Kafka, RabbitMQ, NATS, MySQL, MongoDB, Redis, Terraform, DynamoDB users
2. **Redirect 6 dead aliases** (A2) — point to nearest existing file; avoids silent misses for GitLab CI, Auth0, Clerk, Cognito, AWS SQS/SNS, Terraform (AWS) users
3. **Add 19 missing language/framework alias entries** (A3) — unlocks PHP, Ruby, Elixir, Rust (language key), desktop frameworks for all users of those stacks
4. **Add 9 job queue alias entries** (B1) — unlocks BullMQ, Temporal, Sidekiq, Celery, Faktory, Asynq, River, Dramatiq, Bull
5. **Fix the pipeline gap** (Part C) — `payload.go` + `builder.go` — prerequisite for job queue skills to have any effect
6. **Write 3 missing job queue files** (B1 gaps) — Hangfire, Laravel Queues, Oban
7. **Write 11 new skill files** (B2–B12), ordered by expected manifest usage frequency:
   - B8 `pagination-impl.md` — nearly every list endpoint needs this
   - B5 `file-storage-patterns.md` — common in most apps
   - B3 `api-versioning-impl.md` — used whenever versioning strategy is set
   - B9 `frontend-bundle-optimization.md` — used whenever bundle_optimization is set
   - B10 `frontend-error-boundaries.md` — used whenever error_boundary is set
   - B11 `frontend-realtime-client.md` — used whenever realtime_strategy is set
   - B6 `go-caching-impl.md` — Go services with Redis caching
   - B2 `go-background-workers.md` — Go services with job queues
   - B4 `grpc-gateway.md` — gRPC transcoding gateway config
   - B7 `multi-tenancy.md` — multi-tenant domain configurations
   - B12 `multi-provider-llm.md` — universal injection for multi-provider setups
