# Realization Effects — Config Option Reference

> For every configuration option in the VibeMenu declaration menu, this document
> describes the **exact effect on the code-generation pipeline** when the Realize
> engine consumes the saved `manifest.json`.
>
> The pipeline runs as a DAG of agent tasks executed in dependency order:
> `data → backend → contracts → frontend → infrastructure → cross-cutting`
>
> Options marked **triggers task** create a new DAG node.
> Options marked **injected into** are passed as payload to an existing task.
> Options marked **orchestrator config** control the engine itself (not an agent).

---

## 1 · Backend Tab

### 1.1 Architecture Pattern

The **single most impactful** option in the manifest. It determines the entire
backend task graph topology.

| Value | Realization Effect |
|---|---|
| `Monolith` | One service task chain is emitted under the name `monolith`. All application logic is generated in a single Go module. Sub-tabs for COMM and API GW are not surfaced. |
| `Modular Monolith` | Same as Monolith (one chain), but service units are treated as internal modules within the same binary. COMM links are treated as in-process calls. |
| `Microservices` | One independent task chain per service unit in `m.Backend.Services`. Each chain emits `svc.<slug>.plan → svc.<slug>.deps → svc.<slug>.repository → svc.<slug>.service → svc.<slug>.handler → svc.<slug>.bootstrap`. An API Gateway task is also expected. |
| `Event-Driven` | One task chain per service unit + a `backend.messaging` task for broker config and event stubs. |
| `Hybrid` | All chains + both `backend.messaging` and `backend.gateway` tasks. Each service unit carries a Pattern Tag used to annotate generated code. |

---

### 1.2 Environment Sub Tab

All environment fields are bundled into `EnvConfig` (`m.Backend.Env`) and injected
into every service task chain as payload. They primarily drive `infra.docker` and
`infra.terraform`.

| Field | Realization Effect |
|---|---|
| **Compute Environment** | Sets the deployment target in generated Dockerfiles and IaC. `Containers (Docker)` → multi-stage Dockerfile + docker-compose. `Kubernetes` → K8s manifest generation in IaC task. `Serverless (FaaS)` → function-handler entrypoints, no long-running server. `PaaS` → minimal container with platform-native config (e.g. Procfile). `Bare Metal` / `VM` → systemd unit or init script in IaC. |
| **Cloud Provider** | Shapes IaC resource names, registry URLs, and secret manager references. `AWS` → ECR, Secrets Manager, ALB resources in Terraform. `GCP` → GCR, GCP Secret Manager, Cloud Run/GKE. `Azure` → ACR, Key Vault. `Cloudflare` → Workers/R2/D1 primitives. `Hetzner` / `Self-hosted` → generic Docker/K3s config. |
| **Container Orchestrator** | Determines what compose/manifest files the `infra.docker` task generates. `Docker Compose` → single `docker-compose.yml`. `K8s (managed)` / `K3s` → Deployment, Service, Ingress YAML. `Nomad` → Nomad job spec. `ECS` → task definition JSON. `Cloud Run` → Cloud Run service YAML. `None` → bare Dockerfile only. |
| **Region(s)** | Multi-region selections are embedded in IaC (`infra.terraform`) as provider regions and replication targets. |
| **Environment Stages** | Determines how many environment blocks the CI/CD task generates (e.g. `Development + Staging + Production` → three workflow stages with separate secret scopes and deploy targets). |
| **Language** *(Monolith only)* | Sets the programming language for the monolith task chain. Controls which verifier runs after generation (`go build + vet`, `tsc`, `python -m py_compile`, `terraform validate`). |
| **Framework** *(Monolith only)* | Injected into the monolith service plan task. The plan agent generates `go.mod` (or `package.json` / `pyproject.toml`) using this framework's imports and wires the router in bootstrap. |
| **CORS Strategy** | Injected into service bootstrap. `Permissive` → `AllowAll()` middleware. `Strict allowlist` → allowlist middleware using the CORS Origins value. `Same-origin` → deny cross-origin requests. Also influences nginx config in `infra.docker` when CORS is enforced at proxy level. |
| **CORS Origins** | Free-text list of allowed origins. Injected alongside CORS Strategy into bootstrap and nginx. |
| **Session Mgmt** | Injected into handler/bootstrap layers. `Stateless (JWT only)` → no session store wiring. `Server-side sessions (Redis)` → Redis session middleware added to bootstrap. `Database sessions` → DB-backed session table/repo generated. `None` → no session handling. |
| **Linter** | Injected into CI/CD pipeline task. Options are language-specific; the generated workflow YAML adds a lint step using the selected tool. Examples: `golangci-lint run` / `staticcheck` / `go vet` (Go), `eslint .` / `biome check .` (TypeScript/Node), `ruff check .` / `mypy .` (Python), `./gradlew ktlintCheck` / `detekt` (Kotlin), `cargo clippy` / `cargo audit` (Rust), `rubocop` / `standardrb` (Ruby), `php-cs-fixer fix --dry-run` / `phpstan analyse` (PHP), `mix credo` / `mix dialyzer` (Elixir). |

---

### 1.3 Service Units Sub Tab

Each service unit triggers a **six-task generation chain**:

```
svc.<slug>.plan  →  svc.<slug>.deps  →  svc.<slug>.repository
                                                ↓
                                      svc.<slug>.service
                                                ↓
                                      svc.<slug>.handler
                                                ↓
                                      svc.<slug>.bootstrap
```

| Field | Realization Effect |
|---|---|
| **Name** | Becomes the task ID slug (`svc.<slug>.*`) and the Go module path. All six generated files use this as their module name in `go.mod` and import paths. |
| **Responsibility** | Included in the agent prompt as a human-readable description of what the service owns. Guides the agent when naming types, files, and packages. |
| **Language** | Determines which verifier validates the generated output. Also controls which framework skeleton is scaffolded in the plan task. |
| **Framework** | Injected into the plan task. The plan agent generates `go.mod` (or equivalent) listing this framework as a direct dependency. |
| **Technologies** | Multi-select injected into plan + handler tasks. `WebSocket` → WebSocket upgrade handler and hub generated. `gRPC` → `.proto` files and gRPC server generated. `GraphQL` → GraphQL resolver scaffolding. `REST` → HTTP router. `SSE` → SSE endpoint handler. `Kafka consumer` → consumer group setup in messaging layer. |
| **Pattern Tag** *(Hybrid only)* | Annotates generated code with architectural comments and affects how the bootstrap task wires the service (e.g. `Serverless function` → handler-only entrypoint without a long-running server). |
| **Healthcheck Path** | Injected into the handler task. The router includes a `GET <path>` endpoint returning `200 OK` + status JSON. Also used in docker-compose `healthcheck:` and K8s liveness probe. |
| **Error Format** | Injected into handler task. `RFC 7807 (Problem Details)` → error responses use `application/problem+json` envelope. `Custom JSON envelope` → custom struct. `Platform default` → framework's built-in error handler. |
| **Service Discovery** | Injected into bootstrap. `DNS-based` → service addresses read from env vars as hostnames. `Consul` → Consul SDK client registration at startup. `Kubernetes DNS` → ClusterIP service addresses. `Eureka` → Eureka client registration. `Static config` → hardcoded addresses from env. `None` → no discovery wiring. |

---

### 1.4 Communication Sub Tab

Each communication link is injected into the **handler task** of the `From` service,
and optionally the `To` service.

| Field | Realization Effect |
|---|---|
| **From / To** | Scopes the link to the relevant service handler tasks via `commLinksFor()`. The `From` service's handler generates the client call; the `To` service's handler generates the receiving endpoint/consumer. |
| **Direction** | `Unidirectional` → one-way client call in From's handler. `Bidirectional` → both services get client and server stubs. `Pub/Sub (fan-out)` → publisher in From, subscriber/consumer in To. |
| **Protocol** | `REST (HTTP)` → HTTP client + endpoint. `gRPC` → gRPC stub + server. `GraphQL` → GraphQL client operation. `WebSocket` → WS connection setup. `Message Queue` / `Event Bus` → producer/consumer code (tied to Messaging config). `Internal (in-process)` → direct function call (no network code). |
| **Trigger / Flow** | Included as a code comment in the generated client call and handler to document when the communication occurs. |
| **Sync/Async** | `Synchronous` → blocking call with context timeout. `Asynchronous` → goroutine + channel or queue enqueue. `Fire-and-forget` → goroutine with no response handling. |
| **Resilience** | Injected into the handler's client wrapper. `Circuit breaker` → circuit-breaker middleware (e.g. `gobreaker`). `Retry with backoff` → exponential retry loop. `Timeout` → `context.WithTimeout`. `Bulkhead` → semaphore-limited concurrency. `None` → raw call. |

---

### 1.5 Messaging Sub Tab

**Triggers task:** `backend.messaging` — generates broker config and event stubs.

#### Broker Configuration

| Field | Realization Effect |
|---|---|
| **Broker Technology** | Determines which client library is used. `Kafka` → `confluent-kafka-go` or `sarama` producer/consumer. `NATS` → `nats.go` connection setup. `RabbitMQ` → AMQP channel setup. `Redis Streams` → `go-redis` stream producer/consumer. `AWS SQS/SNS` → AWS SDK v2 SQS client. `Google Pub/Sub` → GCP Pub/Sub client. `Azure Service Bus` → Azure SDK client. `Pulsar` → Apache Pulsar Go client. |
| **Deployment** | `Managed (cloud)` → connection string from env var only, no local setup. `Self-hosted` → docker-compose service block for the broker added by `infra.docker`. `Embedded` → in-process queue implementation. |
| **Serialization Format** | `JSON` → `encoding/json` marshal/unmarshal. `Protobuf` → `.proto` schema + generated Go stubs. `Avro` → schema registry client + Avro codec. `MessagePack` → `vmihailenco/msgpack` codec. `CloudEvents` → CloudEvents SDK wrapping. |
| **Delivery Guarantee** | `At-most-once` → no acknowledgment/commit logic. `At-least-once` → manual ack after processing. `Exactly-once` → transactional producer + idempotent consumer with dedup table. |

#### Event Catalog *(repeatable)*

| Field | Realization Effect |
|---|---|
| **Event Name** | Becomes a constant, a payload struct, and a producer function in the generated event stubs (e.g. `order.placed` → `OrderPlacedEvent` struct + `PublishOrderPlaced()` function). |
| **Domain** | Links the event to a domain struct. The event payload struct is generated with fields derived from the referenced domain's attributes. |
| **Description** | Included as a godoc comment on the event struct. |

---

### 1.6 API Gateway Sub Tab

**Triggers task:** `backend.gateway` — generates gateway configuration files.

| Field | Realization Effect |
|---|---|
| **Gateway Technology** | Determines which config format is generated. `Kong` → `kong.yml` declarative config. `Traefik` → `traefik.yml` + dynamic config. `NGINX` → `nginx.conf` with upstream blocks. `Envoy` → `envoy.yaml` bootstrap. `AWS API Gateway` → OpenAPI + SAM/CloudFormation resource. `Cloudflare Workers` → Worker script + `wrangler.toml`. `Custom` → README placeholder with spec. `None` → task is skipped. |
| **Routing Strategy** | `Path-based` → `/api/<service>/*` routing rules per service. `Header-based` → `X-Service` or similar header routing. `Domain-based` → per-service subdomain upstream rules. |
| **Features** | Each selected feature adds a config block. `Rate limiting` → rate-limit plugin/directive. `JWT validation` → JWT verify middleware at gateway. `SSL termination` → TLS listener config. `Load balancing` → upstream group with balancing algorithm. `Request caching` → cache plugin config. `Logging & tracing` → access log + tracing plugin. `Request transformation` → header/body rewrite rules. `CORS handling` → CORS plugin (replaces app-level CORS if selected). `IP allowlist/blocklist` → IP restriction plugin. `Circuit breaking` → circuit-breaker plugin. `Health checks` → upstream health check config pointing to each service's healthcheck path. |

---

### 1.7 Jobs Sub Tab

Each job queue is injected into the **service plan and bootstrap** tasks.
*(repeatable — multiple queues generate multiple worker setups)*

| Field | Realization Effect |
|---|---|
| **Name** | Identifies the queue. Used as the Go struct name (e.g. `EmailQueue`) and as a named worker in the bootstrap's worker runner. |
| **Technology** | Determines which job library is imported. `Temporal` → Temporal workflow worker setup. `BullMQ` → Node.js BullMQ worker (if Node service). `Sidekiq` → Ruby/Sidekiq worker. `Celery` → Python Celery app. `Faktory` → Faktory worker client. `Asynq` → `hibiken/asynq` task server in Go. `River` → `riverqueue/river` Go worker. `Custom` → interface stub with a README note. |
| **Concurrency** | Sets the worker concurrency value in the generated worker config (e.g. `asynq.Config{Concurrency: 10}`). |
| **Max Retries** | Sets the max retry count on the task/job options struct. |
| **Retry Policy** | `Exponential backoff` → delays computed as `2^attempt` seconds. `Fixed interval` → constant delay. `Linear backoff` → delay proportional to attempt. `None` → no retry config (uses library default). |
| **Dead Letter Q** | `true` → failed tasks after MaxRetries are enqueued to a dead-letter queue. Generated bootstrap includes a DLQ handler stub. `false` → failed tasks are discarded. |
| **Worker Service** | Links this job queue to a specific service unit. The job worker setup code is generated inside that service's bootstrap layer rather than as a standalone process. If unset, the worker is generated as an independent entry point. |
| **Payload DTO** | Links the job payload to a DTO type. The generated worker struct is typed against the selected DTO (e.g. `type EmailJob struct { Payload SendEmailRequest }`), and the enqueue function accepts that DTO directly. |

**Cron Jobs** *(within each job queue)*

| Field | Realization Effect |
|---|---|
| **Name** | Used as the cron handler function name (e.g. `NightlyCleanup`) registered in the worker's scheduler. |
| **Schedule** | The cron expression is passed to the scheduler (e.g. `cron.AddFunc("0 2 * * *", handler)`). |
| **Handler** | Sets the function name that the scheduler calls. If left empty the name is derived from the job name. |
| **Timeout** | Wraps the handler execution in a `context.WithTimeout` call with the specified duration. |

---

### 1.8 Security Sub Tab

Security fields are injected into the **service handler and bootstrap** tasks,
and into the **`infra.docker`** task for reverse proxy config.

| Field | Realization Effect |
|---|---|
| **WAF Provider** | `Cloudflare WAF` / `AWS WAF` → IaC task generates WAF association resources. `ModSecurity` / `NGINX ModSec` → ModSecurity directives added to nginx config in `infra.docker`. `None` → no WAF config. |
| **WAF Ruleset** | `OWASP Core Rule Set` → CRS include directive in nginx/modsec config. `Managed rules` → provider-managed rule group reference in IaC. `Custom` → placeholder comment for custom rules. `None` → skipped. |
| **CAPTCHA** | `hCaptcha` / `reCAPTCHA v2/v3` / `Cloudflare Turnstile` → middleware added to relevant handler endpoints (login, register, contact). Client-side widget script tag injected into frontend task if frontend is configured. `None` → skipped. |
| **Bot Protection** | `Cloudflare Bot Management` / `Imperva` / `DataDome` → SDK client injected into middleware chain. `Custom` → stub middleware. `None` → skipped. |
| **Rate Limit Strategy** | `Token bucket (Redis)` → Redis-backed limiter middleware (e.g. `go-redis/rate`). `Sliding window` → sliding window counter in Redis. `Fixed window` → fixed window counter. `Leaky bucket` → leaky bucket implementation. `None` → no rate limiting middleware generated (unless API Gateway handles it). |
| **Rate Limit Backend** | `Redis` → Redis client initialized in bootstrap for rate limiter. `Memcached` → Memcached client. `In-memory` → in-process atomic counter (not shared across replicas). `None` → fallback to gateway-level rate limiting. |
| **DDoS Protection** | `CDN-level (Cloudflare)` → DNS proxy annotation in IaC. `Provider-managed` → cloud provider DDoS shield resource in IaC. `None` → skipped. |

---

### 1.9 Auth & Identity Sub Tab

**Triggers task:** `backend.auth` — generates authentication middleware, JWT handling,
and identity integration. Auth config is also injected into every service handler
task and the bootstrap task.

| Field | Realization Effect |
|---|---|
| **Auth Strategy** | Multi-select. Each selected strategy adds a middleware and token-handling layer. `JWT (stateless)` → JWT parse/validate middleware using the configured secret/key. `Session-based` → session middleware wired to Session Mgmt backend. `OAuth 2.0 / OIDC` → OIDC client + callback handler. `API Keys` → API key extraction and validation middleware. `mTLS` → TLS client certificate verification in the server config. `None` → no auth middleware generated; all routes are public. |
| **Identity Provider** | `Self-managed` → custom user table + bcrypt password hashing generated. `Auth0` → Auth0 SDK client + JWKS validation. `Clerk` → Clerk SDK middleware. `Supabase Auth` → Supabase JWT validation middleware. `Firebase Auth` → Firebase Admin SDK token verification. `Keycloak` → Keycloak OIDC adapter. `AWS Cognito` → Cognito JWT validator. `Other` → stub middleware with a TODO comment. |
| **Authorization Model** | Shapes the permission-checking code injected into handlers. `RBAC` → role array check against the request context. `ABAC` → attribute-based policy evaluation stub. `ACL` → per-resource access list lookup. `ReBAC` → relationship-based check (e.g. "owns resource"). `Policy-based (OPA/Cedar)` → OPA/Cedar policy engine client call. `Custom` → stub function. |
| **Roles** | Each role becomes a typed constant (e.g. `const RoleAdmin = "admin"`) in the generated auth package. Role constants are referenced in handler middleware guards and in the frontend page access control. |
| **Token Storage (client)** | `HttpOnly cookie` → Set-Cookie header in auth response. `Authorization header (Bearer)` → `Authorization: Bearer <token>` instruction in API docs and CORS config. `WebSocket protocol header` → WS handshake token extraction. These values also inform the frontend task's API client. |
| **Refresh Token** | `None` → only access token issued. `Rotating` → new refresh token issued on each use + old token invalidated. `Non-rotating` → static refresh token stored in DB/Redis. `Sliding window` → TTL extended on each access. Generates token rotation endpoints and storage logic. |
| **MFA Support** | `None` → no MFA code generated. `TOTP` → TOTP enrollment and validation endpoints + QR code generation. `SMS` → SMS provider integration stub (injected alongside Identity Provider). `Email` → OTP email send/verify endpoints. `Passkeys/WebAuthn` → WebAuthn server library integration (e.g. `go-webauthn`). |

**Permissions** *(defined before roles)*

| Field | Realization Effect |
|---|---|
| **Name** | Each permission becomes a typed constant in the generated auth package (e.g. `const PermUsersRead = "users:read"`). The constants are referenced in role definitions and in ABAC/policy-based middleware checks. |
| **Description** | Godoc comment on the generated permission constant. |

**Roles** *(full CRUD editor)*

| Field | Realization Effect |
|---|---|
| **Name** | Becomes a role constant (e.g. `const RoleAdmin = "admin"`) in the generated auth package. Role names are injected into handler middleware guards and into frontend page access-control checks. |
| **Description** | Godoc comment on the role constant. |
| **Permissions** | Each selected permission is added to the role's permission set in the generated RBAC registry (e.g. `rolePermissions["admin"] = []string{PermUsersRead, PermOrdersWrite, ...}`). Middleware uses this map to enforce fine-grained permission checks. |
| **Inherits** | Role inheritance is resolved at startup: the generated auth package merges the parent role's permission set into the child role. Circular inheritance is detected and causes a startup panic. |

---

## 2 · Data Tab

### 2.1 Databases Sub Tab

**Triggers tasks:** `data.schemas` (Go domain structs) and `data.migrations`
(SQL up/down pairs). Both tasks receive the full database list as payload.

| Field | Realization Effect |
|---|---|
| **Alias** | Used as a named connection in the generated bootstrap. Each alias becomes an env var (e.g. `PRIMARY_POSTGRES_DSN`) and a named pool/client instance. |
| **Type** | Determines the database driver library in `go.mod`. `PostgreSQL` → `pgx/v5`. `MySQL` → `go-sql-driver/mysql`. `SQLite` → `mattn/go-sqlite3` or `modernc.org/sqlite`. `MongoDB` → `mongo-driver`. `DynamoDB` → AWS SDK v2 `dynamodb`. `Cassandra` → `gocql`. `Redis` → `go-redis/v9`. `Memcached` → `bradfitz/gomemcache`. `ClickHouse` → `ClickHouse/clickhouse-go`. `Elasticsearch` → `elastic/go-elasticsearch`. Migration SQL syntax is also type-specific. |
| **Version** | Pinned in the docker-compose service image tag (e.g. `postgres:16`) generated by `infra.docker`. |
| **Namespace** | Used as the database name in migration files (`CREATE DATABASE`, `USE`, schema prefix) and as the DSN path segment. |
| **Is Cache** | `yes` → database is treated as a cache store. Repository layer generates cache-specific access patterns (TTL-based get/set) rather than full CRUD. The database is also referenced in the Caching layer config. |
| **SSL Mode** *(PostgreSQL/MySQL)* | Appended to the generated DSN string (e.g. `sslmode=require`). |
| **Consistency** *(Cassandra/MongoDB/DynamoDB)* | Sets the database client's consistency level in the generated connection file. For Cassandra: `gocql` session consistency (e.g. `gocql.LocalQuorum`). For MongoDB: `ReadConcern`/`WriteConcern` levels. For DynamoDB: read consistency mode (`strongly consistent` vs `eventually consistent`). `strong` / `eventual` map to the DB-native equivalents. |
| **Replication** | Drives the deployment topology of the database in `infra.docker` and `infra.terraform`. `single-node` → single container/instance, no HA config. `primary-replica` → primary + read-replica container in docker-compose; managed replica config in IaC. `multi-region` → multi-region DB resource in IaC (e.g. RDS Multi-AZ, Cosmos DB geo-replication). |
| **Pool Min / Pool Max** | Injected into the generated database connection setup (e.g. `pgxpool.Config{MinConns: N, MaxConns: M}`). Sets the minimum and maximum connection pool size for the database driver. |
| **Notes** | Included as a comment in the generated connection file and migration preamble. |

---

### 2.2 Domains Sub Tab

**Triggers tasks:** `data.schemas` (Go structs) and `data.migrations` (SQL DDL).
Domains are also injected into every subsequent task (backend, contracts, frontend).

#### Domain Top-Level Fields

| Field | Realization Effect |
|---|---|
| **Name** | Becomes the Go struct name (e.g. `User`, `Order`) in `internal/domain/<name>.go`. Also becomes the SQL table name (snake_case plural, e.g. `users`, `orders`). |
| **Description** | Included as a package-level godoc comment on the domain struct file. |
| **Databases** | Links the domain to one or more databases. The `data.migrations` task generates migration files for each linked database using that database's SQL dialect. The repository layer generates implementations using each linked database's driver. |
| **Attr Names** | Convenience batch-creation field. Each name becomes a domain attribute with default type `String` (user can refine afterward). |

#### Domain Attributes *(repeatable)*

| Field | Realization Effect |
|---|---|
| **Name** | Becomes a struct field (PascalCase in Go, snake_case in SQL). |
| **Type** | Maps to a Go type and SQL column type. `String` → `string` / `TEXT`. `Int` → `int64` / `BIGINT`. `Float` → `float64` / `DOUBLE PRECISION`. `Boolean` → `bool` / `BOOLEAN`. `DateTime` → `time.Time` / `TIMESTAMPTZ`. `UUID` → `uuid.UUID` / `UUID`. `Enum(values)` → Go `const` block + `CHECK` constraint in SQL. `JSON/Map` → `map[string]any` / `JSONB`. `Binary` → `[]byte` / `BYTEA`. `Array(type)` → `[]type` / `ARRAY`. `Ref(Domain)` → foreign key field + `REFERENCES` constraint in SQL. |
| **Constraints** | Applied to both the Go struct (validation tags) and the SQL column. `required` / `not_null` → `NOT NULL`. `unique` → `UNIQUE` index. `min` / `max` / `min_length` / `max_length` → validation struct tags. `email` / `url` / `regex` / `phone` → validation struct tags. `positive` → `CHECK (value > 0)`. `future` / `past` → `CHECK` date constraint. `enum` → `CHECK (value IN (...))`. |
| **Default** | Added as a Go field default in the constructor and as `DEFAULT <value>` in the SQL column definition. |
| **Sensitive** | `true` → field is tagged with `json:"-"` (excluded from JSON serialization) and a `// sensitive` godoc comment. If `PII Encryption` is set in Governance, the field is wrapped in an encryption accessor. |
| **Validation** | Validation rules are added as struct tags (e.g. `validate:"email,max=255"`) and drive the validation logic in the service layer. |
| **Indexed** | `true` → `CREATE INDEX` statement added to the migration for this column. |
| **Unique** | `true` → `CREATE UNIQUE INDEX` statement added to the migration. |

#### Domain Relationships *(repeatable)*

| Field | Realization Effect |
|---|---|
| **Related Domain** | Adds a foreign key column to the owning domain's migration (e.g. `user_id UUID REFERENCES users(id)`). |
| **Relationship** | `One-to-One` → single FK + UNIQUE constraint. `One-to-Many` → FK on the "many" side. `Many-to-Many` → join table migration generated. |
| **Cascade** | `CASCADE` → `ON DELETE CASCADE`. `SET NULL` → `ON DELETE SET NULL`. `RESTRICT` → `ON DELETE RESTRICT`. `NO ACTION` → `ON DELETE NO ACTION`. `SET DEFAULT` → `ON DELETE SET DEFAULT`. |

---

### 2.3 Caching Sub Tab

Caching is a **repeatable list** — each entry is a named caching configuration.
Fields are injected into the **service layer task**. No separate task is triggered;
the repository and service implementations incorporate caching inline.

| Field | Realization Effect |
|---|---|
| **Name** | Used as the identifier for this caching config in generated code comments and in the service initializer (e.g. `// user-cache: dedicated Redis cache for User domain`). When multiple caching configs are defined, each is initialized separately. |
| **Caching Layer** | `Application-level` → in-memory cache map added to service structs. `Dedicated cache` → Redis/cache client initialized in bootstrap using the selected Cache DB; service methods call it before hitting the DB. `CDN` → cache-control headers added to HTTP handler responses. `None` → no caching code generated for this config. |
| **Cache DB** | *(only when Layer = `Dedicated cache`)* Selects which database alias (marked `Is Cache = yes`) is used as the cache backend. The generated bootstrap initializes a client specifically for this alias. Only shown when a dedicated cache layer is selected. |
| **Strategy** | `Cache-aside` → read-miss populates cache; writes invalidate cache entry. `Read-through` → cache proxy wraps repository. `Write-through` → writes go to cache and DB simultaneously. `Write-behind` → writes go to cache; async flush to DB. |
| **Invalidation** | `TTL-based` → cache SET calls include expiry from TTL field. `Event-driven` → cache invalidation calls added to event handlers. `Manual` → cache.Delete() calls inserted at relevant service methods. `Hybrid` → both TTL and event-driven invalidation. |
| **TTL** | The selected duration is passed to cache SET calls (e.g. `redis.Set(ctx, key, value, 5*time.Minute)`). `Custom` → a `CACHE_TTL` env var is read at startup. |
| **Entities** | Only the selected domains have caching logic generated in their service layer. Unselected domains have direct DB access only. |

---

### 2.4 File / Object Storage Sub Tab

File storage config is injected into the **service and handler tasks** for any domain
that references a storage bucket. `infra.docker` includes the storage client init.
*(repeatable)*

| Field | Realization Effect |
|---|---|
| **Technology** | Determines the storage SDK used. `S3` → AWS SDK v2 S3 client. `GCS` → GCP `cloud.google.com/go/storage`. `Azure Blob` → Azure SDK Blob client. `MinIO` → MinIO Go SDK (S3-compatible). `Cloudflare R2` → S3-compatible client pointed at R2 endpoint. `Local disk` → `os` package file operations (dev/test only). |
| **Purpose** | Becomes a descriptive comment in the generated storage service file and influences naming (e.g. `UserAvatarStorage`). |
| **Access** | `Public (CDN-fronted)` → objects stored with public ACL; CDN URL returned. `Private (signed URLs)` → presigned URL generation function included. `Internal only` → no URL returned; direct stream to caller only. |
| **Max Size** | Injected into the handler as a request body size limit check before upload. |
| **Domains** | Links the storage bucket to domain entities. Upload/download methods are added to those domains' service and handler layers. |
| **TTL Minutes** | Sets the expiry duration on generated presigned URL calls. |
| **Allowed Types** | Added as MIME type validation in the upload handler (reject request if `Content-Type` not in allowed list). |

---

### 2.5 Governance Sub Tab

Governance fields affect the **`data.migrations` task**, **`infra.cicd` task**,
and **IaC task**.

| Field | Realization Effect |
|---|---|
| **Migration Tool** | Options are filtered by backend language. Determines the format and runner for generated migration files. `golang-migrate` / `goose` → `.sql` files in `db/migrations/` with `up`/`down` naming. `Atlas` → `atlas.hcl` schema file. `Flyway` → `V<n>__<name>.sql` naming. `Liquibase` → XML/YAML changeset format. `Prisma Migrate` → `schema.prisma` file. `TypeORM Migrations` → TypeORM migration classes. `Knex.js Migrations` → Knex migration files. `db-migrate` → db-migrate config. `Alembic` → `alembic/versions/*.py` Python migration scripts. `Django Migrations` → `python manage.py makemigrations` Django migration modules. `Exposed Migrations` → Kotlin Exposed migration DSL. `EF Core Migrations` → `dotnet ef migrations add` C# migration classes. `Active Record Migrations` → Rails migration files. `Sequel Migrations` → Sequel migration Ruby files. `Doctrine Migrations` / `Phinx` / `Laravel Migrations` → PHP migration classes. `SQLx Migrations` / `Diesel Migrations` / `refinery` → Rust migration SQL or Rust migration structs. `Ecto Migrations` → Elixir Ecto migration modules. `None` → raw `.sql` files with no runner config. The CI/CD pipeline task adds the correct migration runner command for the selected tool. |
| **Backup Strategy** | `Automated daily` → cron job resource in IaC for scheduled snapshots. `Point-in-time recovery` → PITR enabled on managed DB resource in IaC. `Manual snapshots` → script generated in `scripts/backup.sh`. `Managed provider` → provider-native backup setting enabled in IaC. `None` → no backup config. |
| **Search Tech** | `Elasticsearch` / `Meilisearch` / `Typesense` → search client initialized in bootstrap; index sync methods added to relevant domain service layer. `Algolia` → Algolia client + index push on write operations. `PostgreSQL FTS` → `tsvector`/`tsquery` columns added to relevant migrations and FTS query methods in repository. `None` → no search integration. |
| **Retention Policy** | Embedded in IaC as a lifecycle policy on S3/GCS buckets and as a comment in DB backup scripts. |
| **Delete Strategy** | `Soft-delete` → `deleted_at TIMESTAMPTZ` column added to all domain migrations; repository queries include `WHERE deleted_at IS NULL`. `Hard-delete` → standard `DELETE` statements. `Archival` → deleted rows moved to `*_archive` tables. `Soft + periodic purge` → soft-delete plus a scheduled job that hard-deletes rows older than the retention period. |
| **PII Encryption** | `Field-level AES-256` → encrypt/decrypt wrappers generated for all `Sensitive = true` fields. `Full database encryption` → TDE note added to IaC DB resource. `Application-level` → AES-256 helper in `internal/crypto/` used in service layer. `None` → no encryption beyond transport TLS. |
| **Compliance** | Each selected standard adds a README section listing the controls that the generated code addresses. `GDPR` also forces soft-delete or archival and the PII encryption wrapper. `HIPAA` forces TLS and field-level encryption. `PCI-DSS` adds a note about cardholder data scope. |
| **Data Residency** | Passed to IaC task as region constraints for DB and storage resource placement. |
| **Archival Storage** | `S3 Glacier` / `GCS Archive` / `Azure Archive` → lifecycle transition rule added to IaC for aged objects. `On-premise` → script stub for local archival. `None` → skipped. |

---

## 3 · Contracts Tab

### 3.1 DTOs Sub Tab

**Triggers task:** `contracts` — generates DTO types, request/response models,
and the OpenAPI specification. The `contracts` task depends on all backend service
chains completing first.

#### DTO Top-Level Fields

| Field | Realization Effect |
|---|---|
| **Name** | Becomes the Go struct / TypeScript interface name (e.g. `CreateUserRequest`). Also used as the OpenAPI schema `$ref` name. |
| **Category** | `Request` → struct is used as handler input; validation tags applied. `Response` → struct is used as handler output; `json` tags applied. `Event Payload` → struct is used in messaging producer/consumer. `Shared/Common` → reusable struct imported by both request and response types. |
| **Source Domain(s)** | The DTO fields are pre-populated with attributes from the linked domains. Keeps DTO field types consistent with the domain model. |
| **Description** | Godoc comment on the generated struct. Also maps to the OpenAPI schema `description`. |

#### DTO Fields *(repeatable)*

| Field | Realization Effect |
|---|---|
| **Name** | Becomes a struct field (PascalCase Go, camelCase TypeScript/JSON). |
| **Type** | Maps to Go/TS type and OpenAPI type. `nested(DTO)` → references another DTO as a nested struct. `array(type)` → slice/array. `map(key,value)` → `map[K]V` / Record<K,V>. |
| **Required** | `true` → `validate:"required"` tag in Go; `required` in OpenAPI. |
| **Nullable** | `true` → pointer type in Go (`*string`); `nullable: true` in OpenAPI. |
| **Validation** | Each rule adds a `validate` struct tag entry and a corresponding OpenAPI constraint. Also drives the validation middleware logic in handlers. |
| **Notes** | Included as an inline `//` comment in the generated struct field. |

---

### 3.2 Endpoints / Operations Sub Tab

Endpoint definitions are injected into the `contracts` task (for type generation
and OpenAPI spec) and into each service's **handler task** (for route registration).

| Field | Realization Effect |
|---|---|
| **Service Unit** | Routes this endpoint to the named service's handler task. The route is registered in that service's router file. |
| **Name / Path** | Becomes the HTTP route path, gRPC method name, or GraphQL operation name in the generated router. |
| **Protocol** | `REST` → HTTP handler function + route registration. `GraphQL` → GraphQL resolver function. `gRPC` → `.proto` service method + server implementation stub. `WebSocket message` → WS message type handler in the WS hub. `Event` → event handler registered with the messaging consumer. |
| **Auth Required** | `true` → auth middleware applied to this route in the generated router. The middleware checks the configured Auth Strategy and enforces the selected Roles. `false` → route is public. |
| **Request DTO** | The handler function accepts this DTO as input. Deserialization + validation code is generated. Also wired as the OpenAPI `requestBody` schema. |
| **Response DTO** | The handler function returns this DTO. Serialization code is generated. Also wired as the OpenAPI `responses` schema. |
| **HTTP Method** *(REST)* | Sets the HTTP verb in the route registration (`router.POST(...)`, `router.GET(...)`, etc.). |
| **Operation Type** *(GraphQL)* | `Query` / `Mutation` / `Subscription` → resolver registered in the correct GraphQL schema section. |
| **Stream Type** *(gRPC)* | `Unary` → standard `rpc` definition. `Server stream` / `Client stream` / `Bidirectional` → stream RPC with appropriate Go stream types. |
| **WS Direction** *(WebSocket)* | Controls whether the hub reads, writes, or both for this message type. |
| **Pagination** | `Cursor-based` → `cursor` + `limit` params added to request DTO; `next_cursor` to response. `Offset/limit` → `offset` + `limit` params. `Keyset` → keyset fields added. `Page number` → `page` + `per_page` params. `None` → no pagination params. |
| **Rate Limit** | `Strict` → tight rate limit middleware on this specific route (overrides global). `Relaxed` → higher limits. `Default (global)` → inherits global rate limit config. `None` → no per-route limiting. |
| **Description** | Included as a godoc comment on the handler function and as the OpenAPI `summary`/`description`. |

---

### 3.3 API Versioning Sub Tab

Versioning fields are injected into the `contracts` task (OpenAPI spec) and into
each service's **handler task** (route prefix or middleware).

| Field | Realization Effect |
|---|---|
| **Versioning Strategy** | `URL path (/v1/)` → all routes prefixed with `/v<n>/`. `Header (Accept-Version)` → version extracted from request header; routing middleware generated. `Query param` → version extracted from `?version=v1`. `None` → no versioning; routes registered without version prefix. |
| **Current Version** | Sets the prefix string (e.g. `/v1/`, `v1`). Used in the OpenAPI `info.version` field and route prefixes. |
| **Deprecation Policy** | `Sunset header` → deprecated routes include `Sunset: <date>` response header. `Versioned removal notice` → comment block in router noting removal version. `Changelog entry` → CHANGELOG.md entry generated in `crosscut.docs`. `Custom` → placeholder comment. `None` → no deprecation handling. |

---

### 3.4 External APIs Sub Tab

External API definitions are injected into the `contracts` task for type generation
and into the **service layer task** for client integration.
*(repeatable)*

| Field | Realization Effect |
|---|---|
| **Provider** | Names the generated client struct (e.g. `StripeClient`, `SendGridClient`). |
| **Auth Mechanism** | `API Key` → API key read from env var, added to request headers. `OAuth2 Client Credentials` → token endpoint called at startup; token cached + refreshed. `OAuth2 PKCE` → PKCE flow implementation for browser-based clients. `Bearer Token` → static bearer token from env. `Basic Auth` → base64 username:password header. `mTLS` → client certificate loaded from env/file; TLS config set. `None` → unauthenticated requests. |
| **Base URL** | Set as a constant or env-var-backed config in the generated client. |
| **Rate Limit** | Parsed and used to initialize a client-side rate limiter (e.g. `golang.org/x/time/rate` with the specified rate). |
| **Webhook Path** | Generates an inbound webhook handler route at this path in the relevant service's router, including HMAC signature verification. |
| **Failure Strategy** | `Circuit Breaker` → circuit-breaker wraps all client calls. `Retry with backoff` → exponential retry loop with jitter. `Fallback value` → fallback return value on error. `Timeout + fail` → `context.WithTimeout` wrapping. `None` → raw HTTP call. |

---

## 4 · Frontend Tab

All frontend fields are bundled into the `frontend` task payload. The `frontend`
task generates a complete frontend application in the selected framework.

### 4.1 Technologies Sub Tab

| Field | Realization Effect |
|---|---|
| **Language** | Determines the TypeScript/JavaScript/Dart/Kotlin/Swift project skeleton. Also selects which framework options are available. |
| **Platform** | `Web (SPA)` → client-side only app (Vite/CRA scaffold). `Web (SSR/SSG)` → server-rendered app (Next.js/Nuxt/SvelteKit). `Mobile (cross-platform)` → Flutter/KMP scaffold. `Mobile (native)` → platform-specific project. `Desktop` → Electron/Tauri scaffold. |
| **Framework** | The core scaffolding language/framework (React, Vue, Svelte, Flutter, etc.). Determines the component file format and build tooling in `package.json`. |
| **Meta-framework** | Options filtered by framework. `Next.js` → `next.config.mjs`, app router, server components (React only). `Nuxt` → Nuxt config, pages/ directory (Vue only). `SvelteKit` → SvelteKit routes (Svelte only). `Remix` → Remix loader/action pattern (React only). `Astro` → Astro islands (React/Vue/Svelte/Solid). `None` → plain framework with Vite. |
| **Package Manager** | Options filtered by language. TypeScript/JavaScript: `npm install`, `yarn install`, `pnpm install`, `bun install`. Dart: `pub get`. Kotlin: `./gradlew build`. Swift: `swift package resolve`. The selected command is used in generated Dockerfiles and CI pipeline steps. |
| **Styling** | Options filtered by language. TypeScript/JavaScript only: `Tailwind CSS` → `tailwind.config.js` + PostCSS config; utility classes in components. `CSS Modules` → `.module.css` per component. `Styled Components` → `styled` wrapper components. `Sass/SCSS` → `.scss` files + sass dependency. `Vanilla CSS` → plain `.css`. `UnoCSS` → UnoCSS config. Dart/Kotlin/Swift → `None` / `Custom` only (platform-native styling). |
| **Component Lib** | Options filtered by framework. `shadcn/ui` → `components.json` + component folder (React only). `Radix` → Radix primitives (React). `Material UI` → MUI ThemeProvider wrapper (React/Vue/Angular). `Ant Design` → AntD ConfigProvider (React). `Headless UI` / `DaisyUI` → React. Non-web frameworks: `None` / `Custom` only. `Custom` → placeholder README note. |
| **State Mgmt** | Options filtered by framework. `Zustand` → store file per domain slice (React). `Redux Toolkit` → RTK slice + store setup (React). `Jotai` → atom definitions (React). `React Context` → context + provider wrapper (React). `Pinia` → Pinia store (Vue). `Svelte stores` → writable stores (Svelte). `Signals` → signals primitives (Angular/Solid/Qwik). `None` → local state only. |
| **Data Fetching** | Options filtered by framework. `TanStack Query` → `QueryClient` setup + `useQuery`/`useMutation` hooks (React/Vue/Svelte/Solid). `SWR` → `useSWR` hooks (React/Svelte). `Apollo Client` → `ApolloProvider` + GraphQL operations (React/Vue/Angular). `tRPC client` → tRPC hooks (React). `RTK Query` → RTK Query service (React). `Native fetch` → plain `fetch()` calls in service files (all frameworks). |
| **Form Handling** | Options filtered by framework. `React Hook Form` → `useForm` + `Controller` in form components (React). `Formik` → `Formik` wrapper (React). `Zod + native` → controlled inputs with Zod parse on submit (React/Vue/Svelte/Angular/Solid/Qwik). `Vee-Validate` → VeeValidate (Vue). `None` → uncontrolled forms. |
| **Validation** | Options filtered by language. TypeScript: `Zod` → Zod schema co-located with each DTO. `Yup` → Yup schema. `Valibot` → Valibot schema. `Joi` → Joi schema. `Class-validator` → decorator-based class validation. JavaScript: same minus `Class-validator`. Dart/Kotlin/Swift: `None` only. |
| **PWA Support** | Options filtered by platform. Web (SPA/SSR): `None` → no PWA. `Basic` → Web App Manifest + service worker registration. `Full offline` → offline-first service worker with caching strategies. `Push notifications` → Push API integration + VAPID key setup. Mobile/Desktop platforms: always `None` (PWA concepts don't apply). |
| **Real-time** | `WebSocket` → WS client setup + connection hook; reconnect logic. `SSE` → `EventSource` hook per relevant endpoint. `Polling` → interval-based refetch in data fetching layer. `None` → no real-time. |
| **Image Optim.** | Options filtered by platform. Web only: `Next/Image (built-in)` → `<Image>` component from `next/image`. `Cloudinary` → Cloudinary URL transformer. `Imgix` → Imgix URL params. `Sharp (self-hosted)` → Sharp optimization API route. `CDN transform` → CDN URL query param resize. `None` → plain `<img>` tags. Mobile/Desktop: always `None`. |
| **Auth Flow** | `Redirect (OAuth/OIDC)` → redirect to IdP + callback page. `Modal login` → modal component with credential form. `Magic link` → email magic link request form. `Passwordless` → passkey/biometric prompt. `Social only` → social login buttons only. |
| **Error Boundary** | Options filtered by framework. React: `React Error Boundary` → `ErrorBoundary` wrapper around route-level components. `Global try-catch` → global error handler in app entry (all web frameworks). `Framework default` → framework's built-in error page (`error.tsx` in Next.js, `+error.svelte` in SvelteKit, etc.). `Custom` → custom error component. Flutter/Compose/SwiftUI/UIKit: `Framework default` or `Custom` only. |
| **Bundle Optim.** | Options filtered by language. TypeScript/JavaScript only: `Code splitting (route-based)` → dynamic import per route. `Dynamic imports` → `React.lazy`/`defineAsyncComponent` for heavy components. `Tree shaking only` → Vite/Webpack tree-shake config. `None` → no explicit optimization config. Dart/Kotlin/Swift: always `None` (bundling handled by platform toolchain). |
| **FE Testing** | Options filtered by language. TypeScript/JavaScript: `Vitest` → `vitest.config.ts` + test files. `Jest` → `jest.config.js`. `Testing Library` → `@testing-library/react` setup. `Storybook` → `.storybook/` config + story files per component. `None` → no FE test scaffolding. Dart/Kotlin/Swift: always `None` (covered by the E2E testing field in Cross-Cutting). |
| **Linter** | Options filtered by language. TypeScript/JavaScript: `ESLint + Prettier` → `.eslintrc.json` + `.prettierrc`. `Biome` → `biome.json`. `oxlint` → oxlint config. `Stylelint` → `.stylelintrc`. Dart/Kotlin/Swift: `Custom` or `None`. Added to CI pipeline as a lint step. |

---

### 4.2 Theming Sub Tab

All theming fields are injected into the `frontend` task as style configuration.

| Field | Realization Effect |
|---|---|
| **Dark Mode** | `None` → light-only theme. `Toggle (user preference)` → dark mode toggle component + `localStorage` persistence. `System preference` → `prefers-color-scheme` media query. `Dark only` → dark CSS variables only, no toggle. |
| **Border Radius** | Sets the CSS variable `--radius` (or equivalent) in the generated theme file (e.g. `0`, `4px`, `8px`, `999px`, custom value). |
| **Spacing** | Sets the base spacing unit CSS variable (`--spacing-unit`). Scales all padding/margin/gap tokens accordingly. |
| **Elevation** | `Shadows` → `box-shadow` utility classes generated. `Borders` → `border` variants. `Both` → both shadow and border tokens. `Flat` → no elevation — flat design. |
| **Motion** | `None` → no transition/animation CSS. `Subtle transitions` → short `transition: all 150ms ease` on interactive elements. `Animated (spring/ease)` → spring physics animations (Framer Motion or CSS spring). |
| **Vibe** | Influences the agent's component generation style (e.g. `Playful` → rounded, colorful; `Minimal` → whitespace-heavy, monochrome; `Technical` → monospace accents, dense). |
| **Colors** | Injected as seed colors into the CSS variable palette. HSL/hex values used directly in the theme file. |
| **Description** | Free-text prose passed to the frontend agent as creative direction. Guides component naming, imagery choices, and copy tone. |

---

### 4.3 Pages Sub Tab

Each page is generated as a route file (e.g. `app/dashboard/page.tsx` in Next.js).
*(repeatable)*

| Field | Realization Effect |
|---|---|
| **Name** | Used as the component display name and directory name. |
| **Route** | Maps to the file path in the framework's router (e.g. `/users/:id` → `app/users/[id]/page.tsx`). |
| **Auth Required** | `true` → page wrapped in an auth guard component that redirects to login if unauthenticated. Role check also applied if Auth Roles are set. |
| **Layout** | `Default` → shared layout wrapper. `Sidebar` → sidebar layout. `Full-width` → no max-width constraint. `Blank` → no layout. `Custom` → custom layout import. |
| **Description** | Page-level comment in the generated file. |
| **Core Actions** | Each action becomes a UI element stub (button, form, link) in the generated page component. |
| **Loading** | `Skeleton` → skeleton component shown during data fetch. `Spinner` → loading spinner. `Progressive` → progressive content reveal. `Instant (SSR/SSG)` → no client loading state; data pre-fetched server-side. |
| **Error Handling** | `Inline` → error message rendered inline in the component. `Toast` → toast notification call on error. `Error boundary / fallback page` → error boundary wrapping. `Retry` → retry button + re-fetch logic. |
| **Auth Roles** | Multi-select. Each selected role is checked in the auth guard. Users without a matching role are redirected to an unauthorized page. |
| **Linked Pages** | Generates navigation links (`<Link href="...">`) within the page component pointing to each linked route. |

---

### 4.4 Navigation Sub Tab

Navigation config is injected into the `frontend` task's layout generation.

| Field | Realization Effect |
|---|---|
| **Nav Type** | `Top bar` → `<Header>` component with nav links. `Sidebar` → `<Sidebar>` component with collapsible menu. `Bottom tabs (mobile)` → fixed bottom tab bar. `Hamburger menu` → hamburger icon + slide-out drawer. `Combined` → responsive top bar that collapses to hamburger on mobile. |
| **Breadcrumbs** | `true` → `<Breadcrumb>` component generated and placed in the layout above page content, automatically derived from the route hierarchy. |
| **Auth-Aware** | `true` → nav items conditionally rendered based on auth state (show/hide login button, user menu, role-restricted links). Reads from the auth state store. |

---

### 4.5 I18N Sub Tab

I18N config is injected into the `frontend` task. When enabled, all generated UI
strings are extracted to locale files rather than hardcoded.

| Field | Realization Effect |
|---|---|
| **Enabled** | `false` → all UI strings are hardcoded in components. `true` → activates all I18N fields below. |
| **Default Locale** | Selected from a dropdown of ~45 locales. Sets the fallback locale in the I18N library config (e.g. `i18next.init({ lng: 'en' })`). Also used as the default `lang` attribute on the `<html>` element. |
| **Supported Locales** | Multi-select from the same locale list. Each selected locale gets a translation message file (e.g. `messages/fr.json`, `public/locales/de/common.json`). The generated language switcher component renders a menu item for each selected locale. |
| **I18N Library** | `i18next` → `i18next` + `react-i18next` setup. `next-intl` → `next-intl` middleware + messages directory. `react-i18next` → `useTranslation` hook in components. `LinguiJS` → Lingui CLI + `.po` files. `vue-i18n` → Vue I18n plugin. `Custom` → stub config with placeholder. `None` → no I18N even if Enabled is true. |
| **Timezone Handling** | `UTC always` → all `Date` objects formatted in UTC. `User preference` → timezone stored in user profile; formatter reads it. `Auto-detect` → `Intl.DateTimeFormat().resolvedOptions().timeZone` used. `Manual` → timezone selector component generated. |

---

### 4.6 A11Y / SEO Sub Tab

Accessibility and SEO fields are injected into the `frontend` task.

| Field | Realization Effect |
|---|---|
| **WCAG Level** | `A` → basic ARIA labels on interactive elements. `AA` → full color contrast, focus management, keyboard navigation. `AAA` → enhanced contrast + additional ARIA roles. `None` → no accessibility annotations. |
| **SEO Rendering** | `SSR` → server-side rendered metadata in `<head>`. `SSG` → static `<head>` meta generated at build. `ISR` → incremental static regeneration. `Prerender` → prerender plugin config. `None` → no SEO optimization. |
| **Sitemap** | `true` → sitemap generation config added (e.g. `next-sitemap.config.js`). `false` → no sitemap. |
| **Meta Tags** | `Manual` → static `<Head>` blocks in each page. `Automatic (react-helmet)` → `react-helmet-async` setup. `Framework-native` → Next.js `metadata` export / Nuxt `useHead`. `None` → no meta tag management. |
| **Analytics** | `PostHog` → PostHog client initialized in `_app.tsx`. `Google Analytics 4` → gtag script + GA4 event helpers. `Plausible` → Plausible script tag. `Mixpanel` → Mixpanel SDK init. `Segment` → Segment analytics.js. `Custom` → stub init function. `None` → no analytics. |
| **Frontend RUM** | `Sentry` → Sentry browser SDK init + `ErrorBoundary`. `Datadog RUM` → Datadog RUM init. `LogRocket` → LogRocket init + session replay. `New Relic Browser` → New Relic agent init. `Custom` → placeholder. `None` → no RUM. |

---

### 4.7 Assets Sub Tab

Asset definitions are injected into the `frontend` task as reference data.
*(repeatable)*

| Field | Realization Effect |
|---|---|
| **Name** | Used as the import identifier in the generated component (e.g. `import heroBg from '@/assets/hero-bg.png'`). |
| **Path** | The file path or URL is used as the `src` for the generated import or `<img>` / `<video>` tag. |
| **Asset Type** | Determines how the asset is referenced. `image` → `<img>` or `<Image>` component. `icon` → SVG icon component. `font` → `@font-face` declaration or `next/font` import. `video` → `<video>` element. `mockup` / `moodboard` → `inspiration` usage only — referenced in a design README, not in generated components. |
| **Format** | Informs import handling. `svg` → inline SVG or SVGR import. `png`/`jpg` → standard image import. `figma`/`sketch` → design tool reference (comment only). |
| **Usage** | `project` → asset is imported and used in generated components. `inspiration` → asset is listed in a design references README only; not imported. |
| **Description** | Comment in the generated import or component. |

---

## 5 · Infrastructure Tab

### 5.1 Networking Sub Tab

Networking fields are injected into the **`infra.docker`** (nginx/reverse proxy config)
and **`infra.terraform`** (DNS, TLS, CDN resources) tasks.

| Field | Realization Effect |
|---|---|
| **DNS Provider** | `Cloudflare` → Cloudflare DNS records in Terraform (`cloudflare_record` resource). `Route53` → AWS Route53 zone and A/CNAME records. `Cloud DNS` → GCP Cloud DNS records. `Other` → placeholder DNS record documentation. |
| **TLS/SSL** | `Let's Encrypt` → Certbot/ACME config in the reverse proxy. `Cloudflare` → Cloudflare-terminated TLS; origin cert note. `ACM` → ACM certificate resource in IaC. `Manual` → placeholder for cert file paths. `None (dev)` → HTTP only; warning comment added. |
| **Reverse Proxy** | `Nginx` → `nginx.conf` with upstream blocks + SSL termination. `Caddy` → `Caddyfile`. `Traefik` → `traefik.yml` + Let's Encrypt config. `Cloudflare Tunnel` → `cloudflared` tunnel config. `Cloud LB` → cloud-native load balancer resource in IaC. |
| **CDN** | `Cloudflare` → Cloudflare zone + page rules in IaC. `CloudFront` → CloudFront distribution + origin group. `Fastly` → Fastly service resource. `Vercel Edge` → `vercel.json` edge cache config. `None` → no CDN layer. |
| **Primary Domain** | Used in generated nginx/Caddy server_name, TLS certificate SANs, and API base URL env vars. |
| **Domain Strategy** | `Subdomain per service` → each service gets `<service>.domain.com` in the reverse proxy config. `Path-based routing` → `domain.com/api/<service>` path rules. `Single domain` → all traffic to one domain; internal routing only. `Custom` → placeholder comment. |
| **CORS Enforced** | `Reverse proxy (Nginx/Caddy)` → CORS headers added to nginx/Caddy config. `Application-level` → no proxy CORS (handled in app). `CDN/WAF` → CORS headers at CDN/WAF layer. `Both` → headers at both proxy and app. |
| **SSL Cert Mgmt** | `Auto-renew (certbot/ACME)` → certbot renew cron + webroot config. `Managed (cloud provider)` → managed cert resource in IaC. `Manual` → placeholder note. `Cloudflare proxy` → origin cert note; Cloudflare handles public cert. |

---

### 5.2 CI/CD Sub Tab

CI/CD fields **trigger the `infra.cicd` task** and optionally the `infra.terraform` task.

| Field | Realization Effect |
|---|---|
| **CI/CD Platform** | `GitHub Actions` → `.github/workflows/ci.yml` + `deploy.yml`. `GitLab CI` → `.gitlab-ci.yml`. `Jenkins` → `Jenkinsfile`. `CircleCI` → `.circleci/config.yml`. `ArgoCD` → `argocd-application.yaml` + Kustomize overlays. `Tekton` → Tekton Pipeline and Task manifests. Each platform's specific YAML syntax and secrets handling is used. |
| **Container Registry** | `Docker Hub` → `docker.io/<org>/<image>` in pipeline push steps. `GHCR` → `ghcr.io/<org>/<image>`. `ECR` → `<account>.dkr.ecr.<region>.amazonaws.com/<image>` + ECR login step. `GCR` → `gcr.io/<project>/<image>`. `Self-hosted` → custom registry URL from env var. |
| **Deploy Strategy** | `Rolling` → zero-downtime rolling update config in K8s Deployment or ECS. `Blue-green` → two environment slots with traffic switch. `Canary` → canary weight config (e.g. Argo Rollouts or Flagger). `Recreate` → stop old, start new (accepts downtime). |
| **IaC Tool** | **Triggers `infra.terraform` task.** `Terraform` → `.tf` files for all cloud resources. `Pulumi` → Pulumi program in the backend language. `CloudFormation` → CloudFormation template YAML. `Ansible` → Ansible playbook. `None` → no IaC task emitted. |
| **Secrets Mgmt** | `GitHub Secrets` → secrets referenced as `${{ secrets.KEY }}` in workflows. `HashiCorp Vault` → Vault agent sidecar or CI Vault token step. `AWS Secrets Manager` → `aws secretsmanager get-secret-value` call in CI + app startup. `GCP Secret Manager` → `gcloud secrets` command in CI. `None` → secrets from env vars only. |
| **Container Runtime** | Sets the base Docker image in generated Dockerfiles. `Node Alpine` → `FROM node:20-alpine`. `Go scratch` → `FROM scratch` with statically compiled binary. `Python slim` → `FROM python:3.12-slim`. `Distroless` → Google Distroless base. `Ubuntu` → `FROM ubuntu:22.04`. `Custom` → placeholder with env var. |
| **Backup/DR** | `Cross-region replication` → S3/GCS cross-region replication resource in IaC. `Daily snapshots` → automated snapshot policy in IaC. `Managed provider DR` → provider-managed HA/failover config. `None` → no DR config. |

---

### 5.3 Observability Sub Tab

Observability fields are injected into service **bootstrap tasks** (for SDK init)
and the **`infra.cicd`** task (for deployment of observability infrastructure).

| Field | Realization Effect |
|---|---|
| **Logging** | `Loki + Grafana` → `grafana/agent` config + Loki push endpoint in service logger setup. `ELK Stack` → Logstash/Beats config; structured JSON logging enforced. `CloudWatch` → CloudWatch Logs SDK + log group resource in IaC. `Datadog` → Datadog agent sidecar + DD log client. `Stdout/file` → plain `log.Printf` / structured logger writing to stdout. |
| **Metrics** | `Prometheus + Grafana` → `/metrics` endpoint (e.g. `promhttp.Handler()`) registered per service; Prometheus scrape config in docker-compose. `Datadog` → Datadog metrics client init. `CloudWatch` → CloudWatch metrics SDK calls. `New Relic` → New Relic Go agent init. `None` → no metrics endpoint. |
| **Tracing** | `OpenTelemetry + Jaeger` → OTel SDK init + Jaeger exporter; trace spans around DB calls and HTTP handlers. `OpenTelemetry + Tempo` → OTel SDK + Tempo exporter. `Datadog APM` → Datadog tracer init + middleware. `None` → no tracing. |
| **Error Tracking** | `Sentry` → Sentry Go SDK init + `sentry.CaptureException()` in error handlers. `Datadog` → Datadog error tracking via APM. `Rollbar` → Rollbar notifier. `Built-in` → structured error log with stack trace. `None` → no error tracking. |
| **Health Checks** | `true` → `GET /health` and `GET /ready` endpoints generated in every service's router, returning JSON status. Also used in docker-compose `healthcheck:` and K8s liveness/readiness probes. `false` → no health endpoints. |
| **Alerting** | `Grafana Alerting` → Grafana alert rule YAML generated for key SLOs. `PagerDuty` → PagerDuty integration key config. `OpsGenie` → OpsGenie alert rule. `CloudWatch Alarms` → CloudWatch Alarm resource in IaC. `None` → no alerting config. |
| **Log Retention** | Sets the CloudWatch log group retention, Loki retention config, or ELK ILM policy in the generated observability config. |

---

### 5.4 Environments Sub Tab

Environment fields are injected into the **`infra.cicd`** task.

| Field | Realization Effect |
|---|---|
| **Stages** | Determines how many pipeline stages are generated. Each stage gets its own deployment job with environment-specific secrets and deployment target. |
| **Promotion Pipeline** | Shapes the stage transition logic in CI: whether a merge/manual approval gates promotion between stages. `Dev → Prod (direct)` → two-stage pipeline. `Manual` → all promotions require manual trigger. |
| **Secret Keys** | `Per-environment` → each stage has its own secret set in the CI secret store. `Shared base + overrides` → base secrets shared + environment-specific overrides. `Fully shared` → one secret set for all environments. `None` → no secret management config. |
| **DB Migrations** | `Auto on deploy` → migration run step added before the deploy step in CI pipeline (uses the selected Migration Tool). `Manual CI step` → migration job commented out with instructions. `Flyway` / `Liquibase` / `Atlas` / `golang-migrate` → corresponding migration runner command in the CI step. `None` → no migration step. |
| **DB Seeding** | `Automatic (fixtures)` → seed script run step added after migration in dev/staging pipelines. `Manual` → seed command documented in README. `None` → no seeding. |
| **Preview Envs** | `true` → per-PR preview environment job added (e.g. GitHub Actions environment with PR-scoped deployment). `false` → no preview environments. |

---

## 6 · Cross-Cutting Concerns Tab

### 6.1 Testing Strategy Sub Tab

**Triggers task:** `crosscut.testing` — generates test scaffolding for all layers.
The task depends on every other task completing first.

| Field | Realization Effect |
|---|---|
| **Unit Testing** | Options are filtered by backend language. Generated test files use the selected framework's idioms. `Go testing` → table-driven `_test.go` files (always generated for Go services). `Testify` → Testify `assert`/`require` helpers in `_test.go` files. `Jest` / `Vitest` → `.test.ts` files with describe/it blocks. `pytest` → `test_*.py` files with fixtures. `unittest` → `TestCase` subclasses. `JUnit` / `TestNG` → JUnit 5 / TestNG annotated test classes. `Kotest` → Kotest spec files. `xUnit` / `NUnit` / `MSTest` → C# test project with the selected test framework. `cargo test` → Rust `#[test]` attribute functions. `RSpec` → `.spec.rb` files with describe/it blocks. `minitest` → Minitest `TestCase` classes. `PHPUnit` / `Pest` → PHP test files. `Other` → plain stub test file. |
| **Integration Tests** | `Testcontainers` → Testcontainers setup with real DB containers in integration test files. `Docker Compose` → separate `docker-compose.test.yml` for integration test environment. `In-memory fakes` → in-memory repository implementations for fast integration tests. `None` → no integration test scaffolding. |
| **E2E Testing** | Options are filtered by frontend platform/framework. Web: `Playwright` → `playwright.config.ts` + page object models per page. `Cypress` → `cypress.config.js` + spec files per page. `Selenium` → WebDriver test class per page. Flutter/Dart: `Flutter Driver` → Flutter Driver test setup. `Integration Test` → `integration_test/` directory with widget tests. Android/Kotlin: `Espresso` → Espresso instrumented test classes. `UI Automator` → UI Automator test setup. iOS/Swift: `XCUITest` → XCUITest UI test target. `EarlGrey` → EarlGrey test framework setup. `None` → no E2E scaffolding. |
| **API Testing** | `Bruno` → `bruno/` collection directory with request files per endpoint. `Hurl` → `.hurl` request files per endpoint. `Postman/Newman` → `postman_collection.json` + Newman runner script in CI. `REST Client` → `.http` files per endpoint. `None` → no API test files. |
| **Load Testing** | `k6` → `k6/scripts/` with load test scenarios per key endpoint. `Locust` → `locustfile.py` with user classes *(only available when Python is among the backend languages)*. `Artillery` → `artillery.yml` scenario file. `JMeter` → `.jmx` test plan. `None` → no load test scaffolding. |
| **Contract Testing** | `Pact` → Pact consumer test + provider verification test per service boundary. `Schemathesis` → Schemathesis test command referencing the generated OpenAPI spec. `Dredd` → Dredd hook files + CLI config. `None` → no contract test scaffolding. |

---

### 6.2 Documentation Sub Tab

**Triggers task:** `crosscut.docs` — generates API documentation and changelog files.

| Field | Realization Effect |
|---|---|
| **API Docs** | `OpenAPI/Swagger` → `openapi.yaml` generated from endpoint definitions + Swagger UI setup. `GraphQL Playground` → GraphQL Playground or Apollo Sandbox config. `gRPC reflection` → gRPC server reflection registration + `grpcui` or Buf Studio config. `None` → no API docs task is emitted. |
| **Auto-generation** | `true` → API doc generator annotations (e.g. `swaggo/swag` for Go, `openapi-gen` for TypeScript) added to handler code; CI step runs the generator. `false` → doc file is written once by the agent from the manifest; not regenerated from code. |
| **Changelog** | `Conventional Commits` → `CHANGELOG.md` with Keep a Changelog format generated; `commitlint.config.js` added to enforce conventional commit messages in CI. `Manual` → blank `CHANGELOG.md` with section template. `None` → no changelog. |

---

### 6.3 Standards Sub Tab

Standards fields are injected into the **`infra.cicd`** task and generate
repository configuration files.

| Field | Realization Effect |
|---|---|
| **Branch Strategy** | `GitHub Flow` → CI triggers on `main` push + PR; branch protection rule note in README. `GitFlow` → CI triggers on `main`, `develop`, `release/*`; separate deploy jobs per branch type. `Trunk-based` → CI triggers on every commit to trunk; feature flags expected. `Custom` → placeholder comment. |
| **Dependency Updates** | `Dependabot` → `.github/dependabot.yml` with weekly schedule for language ecosystem. `Renovate` → `renovate.json` config. `Manual` → note in README. `None` → no auto-update config. |
| **Code Review** | `Required (1 approval)` → branch protection config note in README + CI status check required. `Required (2 approvals)` → same with count 2. `Optional` / `None` → no review gate documented. |
| **Feature Flags** | `LaunchDarkly` → LaunchDarkly Go/Node SDK init + example flag evaluation in service layer. `Unleash` → Unleash client init. `Flagsmith` → Flagsmith client init. `Custom (env vars)` → `FEATURE_<NAME>` env var pattern documented in `.env.example`. `None` → no feature flag integration. |
| **Uptime SLO** | Embedded in the generated Grafana alert rule (or CloudWatch Alarm) as the availability threshold. Also documented in the generated `README.md` SLO section. |
| **Latency P99** | Embedded in the generated Grafana/Prometheus alert rule as the P99 latency threshold (e.g. `histogram_quantile(0.99, ...) > 0.1` for `<100ms`). |

---

## 7 · Realize Tab

All Realize Tab fields are **orchestrator config** — they control how the engine
runs, not what is generated.

| Field | Realization Effect |
|---|---|
| **App Name** | Used as the root directory name for generated output and injected into every task as the project name (appears in `go.mod` module path prefix, `package.json` `name` field, `README.md` title). |
| **Output Dir** | All generated files are written relative to this directory. Default `.` places files in the current working directory. |
| **Model** | The global LLM model used for all agent calls. `claude-haiku-4-5-20251001` → fastest and cheapest; best for simple scaffolding tasks. `claude-sonnet-4-6` → balanced capability and cost; recommended for most projects. `claude-opus-4-6` → highest capability; use for complex multi-service systems with intricate business logic. |
| **Concurrency** | `1` → tasks run sequentially (safest, easiest to debug). `2` / `4` / `8` → that many task chains run in parallel (bounded by DAG dependency edges). Higher values reduce wall-clock time but increase API token consumption and risk of context collision on shared resources. |
| **Verify** | `true` → after each agent call, the language-appropriate verifier runs (`go build + vet`, `tsc --noEmit`, `python -m py_compile`, `terraform validate`). Failed verification triggers a retry with the error message injected into the next agent prompt. `false` → files are written without verification (faster iteration, less reliable output). |
| **Dry Run** | `true` → the DAG is built and printed (task IDs, kinds, dependencies, labels) but no agent calls are made. Useful for auditing what will be generated before spending tokens. `false` → full execution. |

### Per-Section Model Overrides

Each pillar can use a different LLM model, independent of the global Model setting.
Set to `default` to inherit the global model.

| Field | Effect When Overridden |
|---|---|
| **Backend Model** | All `svc.*`, `backend.auth`, `backend.messaging`, `backend.gateway` tasks use this model. |
| **Data Model** | `data.schemas` and `data.migrations` tasks use this model. |
| **Contracts Model** | The `contracts` task uses this model. |
| **Frontend Model** | The `frontend` task uses this model. |
| **Infra Model** | `infra.docker`, `infra.terraform`, `infra.cicd` tasks use this model. |
| **Crosscut Model** | `crosscut.testing` and `crosscut.docs` tasks use this model. |

> **Recommended override strategy:** Use `claude-haiku-4-5-20251001` for `Data` and
> `Infra` (deterministic, template-like output), `claude-sonnet-4-6` for `Backend`
> and `Contracts`, and `claude-opus-4-6` for `Frontend` (high design judgment
> required) and complex `Backend` microservice systems.

---

## Task DAG Summary

Below is the complete set of tasks that can be emitted, in dependency wave order:

| Wave | Task ID | Triggered By |
|---|---|---|
| 0 | `data.schemas` | Always (if any domain defined) |
| 0 | `data.migrations` | Always (if any domain defined) |
| 1 | `svc.<slug>.plan` | Each service unit |
| 1 | `svc.<slug>.deps` | Each service unit |
| 1 | `svc.<slug>.repository` | Each service unit |
| 1 | `svc.<slug>.service` | Each service unit |
| 1 | `svc.<slug>.handler` | Each service unit |
| 1 | `svc.<slug>.bootstrap` | Each service unit |
| 1 | `backend.auth` | Auth Strategy ≠ empty |
| 1 | `backend.messaging` | Messaging sub-tab configured |
| 1 | `backend.gateway` | API Gateway ≠ None |
| 2 | `contracts` | Any DTO or Endpoint defined |
| 3 | `frontend` | Framework ≠ empty |
| 4 | `infra.docker` | Always |
| 4 | `infra.terraform` | IaC Tool ≠ None |
| 4 | `infra.cicd` | CI/CD Platform ≠ none |
| 5 | `crosscut.testing` | Unit or E2E testing configured |
| 5 | `crosscut.docs` | API Docs ≠ empty |
