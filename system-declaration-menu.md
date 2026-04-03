# System Declaration Menu — Specification

> A declarative menu for fully specifying a web system's architecture, data, contracts, frontend, and infrastructure. Each section builds context that downstream sections reference (e.g., Domains defined in **Data** are selectable in **Backend** modules).

---

## Global Concepts

These are referenced throughout the menu:

| Concept          | Definition                                                                 |
|------------------|----------------------------------------------------------------------------|
| **Domain**       | A bounded context with its own entities/attributes (defined in Data tab)   |
| **Service Unit** | A microservice, module, or the monolith itself — any deployable backend unit |
| **Contract**     | A typed interface between any two boundaries (frontend↔backend, service↔service) |
| **Data Flow**    | A directional description of when/why/what data moves between two units    |

---

## 1 · Backend Tab

### 1.1 Architecture Pattern *(top-level selector)*

| Option              | Sub-tabs Unlocked                                                              |
|---------------------|--------------------------------------------------------------------------------|
| Monolith            | ENV · SERVICES · JOBS · SECURITY · AUTH                                        |
| Modular Monolith    | ENV · SERVICES · COMM · JOBS · SECURITY · AUTH                                 |
| Microservices       | ENV · SERVICES · COMM · API GW · JOBS · SECURITY · AUTH                        |
| Event-Driven        | ENV · SERVICES · COMM · MESSAGING · JOBS · SECURITY · AUTH                     |
| Hybrid              | ENV · SERVICES · COMM · MESSAGING · API GW · JOBS · SECURITY · AUTH            |

> Selecting an architecture pattern sets the **communication defaults** and **deployment topology** but every option shares the same service-unit definition shape below.

---

### 1.2 Environment Sub Tab

| Field                  | Options / Input                                                                                              |
|------------------------|--------------------------------------------------------------------------------------------------------------|
| Compute Environment    | `Bare Metal` · `VM` · `Containers (Docker)` · `Kubernetes` · `Serverless (FaaS)` · `PaaS`                   |
| Cloud Provider         | `AWS` · `GCP` · `Azure` · `Cloudflare` · `Hetzner` · `Self-hosted` · `Other (specify)`                      |
| Container Orchestrator | `Docker Compose` · `K3s` · `K8s (managed)` · `Nomad` · `ECS` · `Cloud Run` · `None`                        |
| Region(s)              | Multi-select: `us-east-1/2` · `us-west-1/2` · `eu-west-1/2` · `eu-central-1` · `ap-southeast-1/2` · `ap-northeast-1` · `sa-east-1` · `ca-central-1` · `af-south-1` |
| Environment Stages     | `Development` · `Development + Staging` · `Development + Staging + Production` · `Staging + Production` · `Production only` |
| Language               | *(Monolith only)* `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Framework              | *(Monolith only)* Dynamically filtered by language — see §1.3 for options                                    |
| CORS Strategy          | `Permissive` · `Strict allowlist` · `Same-origin`                                                            |
| CORS Origins           | Free text                                                                                                    |
| Session Mgmt           | `Stateless (JWT only)` · `Server-side sessions (Redis)` · `Database sessions` · `None`                      |
| Linter                 | Dynamically filtered by language — see table below                                                           |

Linter options per language:

| Language        | Linters                                                                               |
|-----------------|---------------------------------------------------------------------------------------|
| Go              | `golangci-lint` · `staticcheck` · `go vet` · `None`                                  |
| TypeScript/Node | `ESLint` · `Biome` · `TSLint (legacy)` · `None`                                      |
| Python          | `Ruff` · `Flake8` · `Pylint` · `mypy` · `None`                                       |
| Java            | `Checkstyle` · `SpotBugs` · `PMD` · `SonarLint` · `None`                             |
| Kotlin          | `ktlint` · `detekt` · `SonarLint` · `None`                                           |
| C#/.NET         | `Roslyn Analyzers` · `StyleCop` · `SonarLint` · `None`                               |
| Rust            | `Clippy` · `cargo-audit` · `None`                                                     |
| Ruby            | `RuboCop` · `StandardRB` · `None`                                                     |
| PHP             | `PHP-CS-Fixer` · `PHPStan` · `Psalm` · `None`                                        |
| Elixir          | `Credo` · `Dialyxir` · `None`                                                         |

---

### 1.3 Service Units Sub Tab

#### Adding / Editing a Service Unit

| Field             | Input                                                                                                         |
|-------------------|---------------------------------------------------------------------------------------------------------------|
| Name              | Free text identifier (e.g., `auth-service`, `billing-module`)                                                 |
| Responsibility    | Free text description of what this unit owns                                                                   |
| Language          | `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Framework         | *(dynamically filtered by language — see below)*                                                               |
| Technologies      | Multi-select: `WebSocket` · `gRPC` · `REST` · `GraphQL` · `SSE` · `tRPC` · `MQTT` · `Kafka consumer`         |
| Pattern Tag       | *(Hybrid only)* `Monolith part` · `Modular module` · `Microservice` · `Event processor` · `Serverless function` |
| Healthcheck Path  | Free text (default: `/healthz`)                                                                                |
| Error Format      | `RFC 7807 (Problem Details)` · `Custom JSON envelope` · `Platform default`                                    |
| Service Discovery | `DNS-based` · `Consul` · `Kubernetes DNS` · `Eureka` · `Static config` · `None`                               |

Framework suggestions per language:

| Language        | Frameworks                                                            |
|-----------------|-----------------------------------------------------------------------|
| Go              | `Fiber` · `Gin` · `Echo` · `Chi` · `net/http (stdlib)` · `Connect`   |
| TypeScript/Node | `Express` · `Fastify` · `NestJS` · `Hono` · `tRPC` · `Elysia (Bun)` |
| Python          | `FastAPI` · `Django` · `Flask` · `Litestar` · `Starlette`            |
| Java            | `Spring Boot` · `Quarkus` · `Micronaut` · `Jakarta EE`               |
| Kotlin          | `Ktor` · `Spring Boot (Kotlin)` · `http4k`                           |
| C#/.NET         | `ASP.NET Core` · `Minimal APIs` · `Carter`                           |
| Rust            | `Axum` · `Actix-web` · `Rocket` · `Warp`                             |
| Ruby            | `Rails` · `Sinatra` · `Hanami` · `Roda`                              |
| PHP             | `Laravel` · `Symfony` · `Slim` · `Laminas`                           |
| Elixir          | `Phoenix` · `Plug` · `Bandit`                                        |

---

### 1.4 Communication Sub Tab

> Shared across **all** multi-service architecture patterns. Each link is a directed edge in the system graph.

| Field         | Input                                                                                                    |
|---------------|----------------------------------------------------------------------------------------------------------|
| From          | Select a service unit                                                                                    |
| To            | Select a service unit                                                                                    |
| Direction     | `Unidirectional (→)` · `Bidirectional (↔)` · `Pub/Sub (fan-out)`                                        |
| Protocol      | `REST (HTTP)` · `gRPC` · `GraphQL` · `WebSocket` · `Message Queue` · `Event Bus` · `Internal (in-process)` |
| Trigger / Flow | Free text description of **when** this communication happens                                            |
| Sync/Async    | `Synchronous` · `Asynchronous` · `Fire-and-forget`                                                      |
| Resilience    | Multi-select: `Circuit breaker` · `Retry with backoff` · `Timeout` · `Bulkhead` · `None`                |

---

### 1.5 Messaging Sub Tab

> Visible for Event-Driven and Hybrid patterns.

**Broker Configuration**

| Field                | Options / Input                                                                    |
|----------------------|------------------------------------------------------------------------------------|
| Broker Technology    | `Kafka` · `NATS` · `RabbitMQ` · `Redis Streams` · `AWS SQS/SNS` · `Google Pub/Sub` · `Azure Service Bus` · `Pulsar` |
| Deployment           | `Managed (cloud)` · `Self-hosted` · `Embedded`                                    |
| Serialization Format | `JSON` · `Protobuf` · `Avro` · `MessagePack` · `CloudEvents`                      |
| Delivery Guarantee   | `At-most-once` · `At-least-once` · `Exactly-once`                                 |

**Event Catalog** *(repeatable)*

| Field       | Input                                           |
|-------------|-------------------------------------------------|
| Event Name  | e.g., `order.placed`, `user.registered`         |
| Domain      | Select from **Data → Domains**                  |
| Description | When/why this event fires                       |

---

### 1.6 API Gateway Sub Tab

> Auto-suggested for Microservices and Hybrid patterns.

| Field              | Options / Input                                                              |
|--------------------|------------------------------------------------------------------------------|
| Gateway Technology | `Kong` · `Traefik` · `NGINX` · `Envoy` · `AWS API Gateway` · `Cloudflare Workers` · `Custom (specify)` · `None` |
| Routing Strategy   | `Path-based` · `Header-based` · `Domain-based`                               |
| Features           | Multi-select: `Rate limiting` · `JWT validation` · `SSL termination` · `Load balancing` · `Request caching` · `Logging & tracing` · `Request transformation` · `CORS handling` · `IP allowlist/blocklist` · `Circuit breaking` · `Health checks` |

---

### 1.7 Jobs Sub Tab

> Background/scheduled job queue configuration. *(repeatable — add multiple queues)*

| Field          | Options / Input                                                                      |
|----------------|--------------------------------------------------------------------------------------|
| Name           | Free text identifier for this queue                                                  |
| Technology     | `Temporal` · `BullMQ` · `Sidekiq` · `Celery` · `Faktory` · `Asynq` · `River` · `Custom` |
| Concurrency    | Free text (default: `10`)                                                            |
| Max Retries    | Free text (default: `3`)                                                             |
| Retry Policy   | `Exponential backoff` · `Fixed interval` · `Linear backoff` · `None`                 |
| Dead Letter Q  | `false` · `true`                                                                     |
| Worker Service | Select from defined service units — which service hosts this worker                  |
| Payload DTO    | Select from defined DTOs — the job payload type                                      |

**Cron Jobs** *(repeatable within each queue)*

| Field    | Options / Input                                                              |
|----------|------------------------------------------------------------------------------|
| Name     | Free text identifier (e.g., `nightly-cleanup`)                               |
| Schedule | Cron expression (e.g., `0 2 * * *`)                                         |
| Handler  | Free text — handler function / method name                                   |
| Timeout  | Free text — maximum execution time (e.g., `30s`, `5m`)                      |

---

### 1.8 Security Sub Tab

| Field               | Options / Input                                                                                 |
|---------------------|-------------------------------------------------------------------------------------------------|
| WAF Provider        | `Cloudflare WAF` · `AWS WAF` · `ModSecurity` · `NGINX ModSec` · `None`                         |
| WAF Ruleset         | `OWASP Core Rule Set` · `Managed rules` · `Custom` · `None`                                     |
| CAPTCHA             | `hCaptcha` · `reCAPTCHA v2` · `reCAPTCHA v3` · `Cloudflare Turnstile` · `None`                 |
| Bot Protection      | `Cloudflare Bot Management` · `Imperva` · `DataDome` · `Custom` · `None`                        |
| Rate Limit Strategy | `Token bucket (Redis)` · `Sliding window` · `Fixed window` · `Leaky bucket` · `None`            |
| Rate Limit Backend  | `Redis` · `Memcached` · `In-memory` · `None`                                                    |
| DDoS Protection     | `CDN-level (Cloudflare)` · `Provider-managed` · `None`                                          |

---

### 1.9 Auth & Identity Sub Tab

| Field                  | Options / Input                                                                               |
|------------------------|-----------------------------------------------------------------------------------------------|
| Auth Strategy          | Multi-select: `JWT (stateless)` · `Session-based` · `OAuth 2.0 / OIDC` · `API Keys` · `mTLS` · `None` |
| Identity Provider      | `Self-managed` · `Auth0` · `Clerk` · `Supabase Auth` · `Firebase Auth` · `Keycloak` · `AWS Cognito` · `Other` |
| Authorization Model    | `RBAC` · `ABAC` · `ACL` · `ReBAC` · `Policy-based (OPA/Cedar)` · `Custom`                    |
| Token Storage (client) | Multi-select: `HttpOnly cookie` · `Authorization header (Bearer)` · `WebSocket protocol header` · `Other` |
| Refresh Token          | `None` · `Rotating` · `Non-rotating` · `Sliding window`                                       |
| MFA Support            | `None` · `TOTP` · `SMS` · `Email` · `Passkeys/WebAuthn`                                       |

**Permissions** *(repeatable — define named permission strings before defining roles)*

| Field       | Input                                                              |
|-------------|--------------------------------------------------------------------|
| Name        | e.g., `users:read`, `orders:write`, `reports:export`               |
| Description | What this permission allows                                        |

**Roles** *(repeatable — full CRUD role editor, not a preset multi-select)*

| Field       | Input                                                              |
|-------------|--------------------------------------------------------------------|
| Name        | e.g., `admin`, `editor`, `viewer`                                  |
| Description | What this role represents in the system                            |
| Permissions | Multi-select from defined Permissions above                        |
| Inherits    | Multi-select from other defined roles (role hierarchy / inheritance) |

> Roles defined here are referenced in **Contracts → Endpoints** (`auth_roles`) and **Frontend → Pages** (`auth_roles`) for access control.

---

## 2 · Data Tab

### 2.1 Databases Sub Tab

#### Adding a Database

| Field       | Options / Input                                                                                                        |
|-------------|------------------------------------------------------------------------------------------------------------------------|
| Alias       | Free text identifier (e.g., `primary-postgres`, `cache-redis`)                                                         |
| Type        | `PostgreSQL` · `MySQL` · `SQLite` · `MongoDB` · `DynamoDB` · `Cassandra` · `Redis` · `Memcached` · `ClickHouse` · `Elasticsearch` · `other` |
| Version     | Free text (e.g., `16`, `7.x`)                                                                                          |
| Namespace   | Free text (database name / schema)                                                                                      |
| Is Cache    | `no` · `yes`                                                                                                           |
| SSL Mode    | *(PostgreSQL/MySQL only)* `require` · `disable` · `verify-ca` · `verify-full`                                          |
| Consistency | *(Cassandra/MongoDB/DynamoDB only)* `strong` · `eventual` · `LOCAL_QUORUM` · `ONE` · `QUORUM` · `ALL` · `LOCAL_ONE`   |
| Replication | *(not available for Redis/Memcached/SQLite)* `single-node` · `primary-replica` · `multi-region`                       |
| Pool Min    | *(not available for Redis/Memcached)* Free text integer — minimum connection pool size                                  |
| Pool Max    | *(not available for Redis/Memcached)* Free text integer — maximum connection pool size                                  |
| Notes       | Free text                                                                                                              |

---

### 2.2 Domains Sub Tab

> Domains are the **source of truth** for your system's data model. They are referenced by service units, contracts, and frontend pages.

#### Adding a Domain

| Field       | Input                                                              |
|-------------|--------------------------------------------------------------------|
| Name        | e.g., `User`, `Order`, `Product`, `Message`                        |
| Description | What this domain represents in the business context                |
| Databases   | Multi-select from databases created in §2.1                        |
| Attr Names  | Comma-separated attribute names for batch creation                 |

#### Domain Attributes *(repeatable)*

| Field       | Options / Input                                                                                               |
|-------------|---------------------------------------------------------------------------------------------------------------|
| Name        | e.g., `id`, `email`, `created_at`                                                                             |
| Type        | `String` · `Int` · `Float` · `Boolean` · `DateTime` · `UUID` · `Enum(values)` · `JSON/Map` · `Binary` · `Array(type)` · `Ref(Domain)` |
| Constraints | Multi-select: `required` · `unique` · `not_null` · `min` · `max` · `min_length` · `max_length` · `email` · `url` · `regex` · `positive` · `future` · `past` · `enum` |
| Default     | *(optional)* Default value or generation strategy                                                             |
| Sensitive   | `false` · `true` — marks field for encryption/masking/audit                                                   |
| Validation  | Multi-select: `email` · `url` · `regex` · `min_length` · `max_length` · `min_value` · `max_value` · `phone` · `uuid` · `date_format` · `enum` · `custom` |
| Indexed     | `false` · `true`                                                                                              |
| Unique      | `false` · `true`                                                                                              |

#### Domain Relationships *(repeatable)*

| Field          | Input                                                       |
|----------------|-------------------------------------------------------------|
| Related Domain | Select from other domains                                   |
| Relationship   | `One-to-One` · `One-to-Many` · `Many-to-Many`              |
| Cascade        | `CASCADE` · `SET NULL` · `RESTRICT` · `NO ACTION` · `SET DEFAULT` |

---

### 2.3 Caching Sub Tab

*(repeatable — add multiple named caching configurations)*

| Field         | Options / Input                                                              |
|---------------|------------------------------------------------------------------------------|
| Name          | Free text identifier for this caching configuration (e.g., `user-cache`, `session-cache`) |
| Caching Layer | `Application-level` · `Dedicated cache` · `CDN` · `None`                    |
| Cache DB      | *(only when Layer = `Dedicated cache`)* Select from databases with `Is Cache = yes` |
| Strategy      | Multi-select: `Cache-aside` · `Read-through` · `Write-through` · `Write-behind` |
| Invalidation  | `TTL-based` · `Event-driven` · `Manual` · `Hybrid`                           |
| TTL           | `30s` · `1m` · `5m` · `15m` · `1h` · `24h` · `Custom`                       |
| Entities      | Multi-select from domains (populated dynamically)                            |

---

### 2.4 File / Object Storage Sub Tab

> *(repeatable — add multiple storage buckets)*

| Field         | Options / Input                                                                           |
|---------------|-------------------------------------------------------------------------------------------|
| Technology    | `S3` · `GCS` · `Azure Blob` · `MinIO` · `Cloudflare R2` · `Local disk`                   |
| Purpose       | Free text (e.g., "User avatars", "Document uploads", "Backups")                           |
| Access        | `Public (CDN-fronted)` · `Private (signed URLs)` · `Internal only`                        |
| Max Size      | `1 MB` · `5 MB` · `10 MB` · `25 MB` · `50 MB` · `100 MB` · `500 MB` · `1 GB` · `Unlimited` |
| Domains       | Multi-select from domains (which domains store files here)                                |
| TTL Minutes   | `30` · `60` · `1440` · `10080` · `Custom` (for signed URL expiry)                        |
| Allowed Types | Multi-select: `image/*` · `application/pdf` · `video/*` · `audio/*` · `text/*` · `application/json` |

---

### 2.5 Governance Sub Tab

| Field               | Options / Input                                                                               |
|---------------------|-----------------------------------------------------------------------------------------------|
| Migration Tool      | Dynamically filtered by backend language — see table below                                    |
| Backup Strategy     | `Automated daily` · `Point-in-time recovery` · `Manual snapshots` · `Managed provider` · `None` |

Migration tool options per backend language:

| Language        | Tools                                                                                         |
|-----------------|-----------------------------------------------------------------------------------------------|
| Go              | `golang-migrate` · `Atlas` · `goose` · `None`                                                 |
| TypeScript/Node | `Prisma Migrate` · `TypeORM Migrations` · `Knex.js Migrations` · `db-migrate` · `None`        |
| Python          | `Alembic` · `Django Migrations` · `Flyway` · `None`                                           |
| Java            | `Flyway` · `Liquibase` · `None`                                                               |
| Kotlin          | `Flyway` · `Liquibase` · `Exposed Migrations` · `None`                                        |
| C#/.NET         | `EF Core Migrations` · `Flyway` · `Liquibase` · `None`                                        |
| Ruby            | `Active Record Migrations` · `Sequel Migrations` · `None`                                     |
| PHP             | `Doctrine Migrations` · `Phinx` · `Laravel Migrations` · `None`                               |
| Rust            | `SQLx Migrations` · `Diesel Migrations` · `refinery` · `None`                                 |
| Elixir          | `Ecto Migrations` · `None`                                                                    |
| *(no language)* | `golang-migrate` · `Atlas` · `Flyway` · `Liquibase` · `Prisma Migrate` · `Alembic` · `None`  |
| Search Tech         | `Elasticsearch` · `Meilisearch` · `Algolia` · `PostgreSQL FTS` · `Typesense` · `None`         |
| Retention Policy    | `30 days` · `90 days` · `1 year` · `3 years` · `7 years` · `Indefinite` · `Custom`            |
| Delete Strategy     | `Soft-delete` · `Hard-delete` · `Archival` · `Soft + periodic purge`                          |
| PII Encryption      | `Field-level AES-256` · `Full database encryption` · `Application-level` · `None`             |
| Compliance          | Multi-select: `GDPR` · `HIPAA` · `SOC2 Type II` · `PCI-DSS` · `ISO-27001` · `CCPA` · `PIPEDA` |
| Data Residency      | `US` · `EU` · `APAC` · `US + EU` · `Global` · `Custom`                                       |
| Archival Storage    | `S3 Glacier` · `GCS Archive` · `Azure Archive` · `On-premise` · `None`                        |

---

## 3 · Contracts Tab

### 3.1 DTOs Sub Tab

#### Adding a DTO

| Field            | Input                                                                                     |
|------------------|-------------------------------------------------------------------------------------------|
| Name             | e.g., `CreateUserRequest`, `OrderSummaryResponse`, `UserRegisteredEvent`                  |
| Category         | `Request` · `Response` · `Event Payload` · `Shared/Common`                                |
| Source Domain(s) | Multi-select from **Data → Domains**                                                      |
| Description      | What this DTO represents and when it's used                                               |

#### DTO Fields *(repeatable)*

| Field      | Options / Input                                                                                            |
|------------|------------------------------------------------------------------------------------------------------------|
| Name       | e.g., `email`, `order_items`, `total`                                                                      |
| Type       | `string` · `int` · `float` · `boolean` · `datetime` · `uuid` · `enum(values)` · `array(type)` · `nested(DTO)` · `map(key,value)` |
| Required   | `false` · `true`                                                                                           |
| Nullable   | `false` · `true`                                                                                           |
| Validation | Multi-select: `required` · `min_length` · `max_length` · `min_value` · `max_value` · `email` · `url` · `regex` · `uuid` · `enum` · `phone` · `pattern` · `custom` |
| Notes      | *(optional)* Mapping notes, transformation hints                                                           |

---

### 3.2 Endpoints / Operations Sub Tab

#### Adding an Endpoint

| Field           | Options / Input                                                             |
|-----------------|-----------------------------------------------------------------------------|
| Service Unit    | Select which backend unit exposes this                                      |
| Name / Path     | e.g., `POST /api/v1/users`, `getUser`, `UserService.Create`                 |
| Protocol        | `REST` · `GraphQL` · `gRPC` · `WebSocket message` · `Event` *(filtered by service technologies)* |
| Auth Required   | `false` · `true`                                                            |
| Request DTO     | Select from DTOs tab                                                        |
| Response DTO    | Select from DTOs tab                                                        |
| HTTP Method     | *(REST only)* `GET` · `POST` · `PUT` · `PATCH` · `DELETE`                   |
| Operation Type  | *(GraphQL only)* `Query` · `Mutation` · `Subscription`                      |
| Stream Type     | *(gRPC only)* `Unary` · `Server stream` · `Client stream` · `Bidirectional` |
| WS Direction    | *(WebSocket only)* `Client→Server` · `Server→Client` · `Bidirectional`      |
| Pagination      | `Cursor-based` · `Offset/limit` · `Keyset` · `Page number` · `None`         |
| Rate Limit      | `Default (global)` · `Strict` · `Relaxed` · `None`                          |
| Description     | What this endpoint does                                                     |

---

### 3.3 API Versioning Sub Tab

| Field               | Options / Input                                                           |
|---------------------|---------------------------------------------------------------------------|
| Versioning Strategy | `URL path (/v1/)` · `Header (Accept-Version)` · `Query param` · `None`   |
| Current Version     | Free text (e.g., `v1`)                                                    |
| Deprecation Policy  | `None` · `Sunset header` · `Versioned removal notice` · `Changelog entry` · `Custom` |

---

### 3.4 External APIs Sub Tab

> *(repeatable — define each third-party API dependency)*

| Field            | Options / Input                                                                               |
|------------------|-----------------------------------------------------------------------------------------------|
| Provider         | Free text (e.g., `Stripe`, `SendGrid`, `Twilio`)                                              |
| Auth Mechanism   | `API Key` · `OAuth2 Client Credentials` · `OAuth2 PKCE` · `Bearer Token` · `Basic Auth` · `mTLS` · `None` |
| Base URL         | Free text                                                                                     |
| Rate Limit       | Free text (e.g., `100 req/min`)                                                               |
| Webhook Path     | Free text (your inbound webhook endpoint)                                                     |
| Failure Strategy | `Circuit Breaker` · `Retry with backoff` · `Fallback value` · `Timeout + fail` · `None`       |

---

## 4 · Frontend Tab

### 4.1 Technologies Sub Tab

| Field            | Options                                                                                                         |
|------------------|-----------------------------------------------------------------------------------------------------------------|
| Language         | `TypeScript` · `JavaScript` · `Dart` · `Kotlin` · `Swift`                                                      |
| Platform         | `Web (SPA)` · `Web (SSR/SSG)` · `Mobile (cross-platform)` · `Mobile (native)` · `Desktop`                      |
| Framework        | *(filtered by language — see below)*                                                                            |
| Meta-framework   | *(filtered by framework — see below)*                                                                           |
| Package Manager  | *(filtered by language)* `npm` · `yarn` · `pnpm` · `bun` *(TypeScript/JavaScript)* · `pub` *(Dart)* · `Gradle` *(Kotlin)* · `SwiftPM` *(Swift)* |
| Styling          | *(filtered by language)* `Tailwind CSS` · `CSS Modules` · `Styled Components` · `Sass/SCSS` · `Vanilla CSS` · `UnoCSS` *(TypeScript/JavaScript only)* · `None` · `Custom` |
| Component Lib    | *(filtered by framework — see below)*                                                                           |
| State Mgmt       | *(filtered by framework — see below)*                                                                           |
| Data Fetching    | *(filtered by framework — see below)*                                                                           |
| Form Handling    | *(filtered by framework — see below)*                                                                           |
| Validation       | *(filtered by language)* `Zod` · `Yup` · `Valibot` · `Joi` · `Class-validator` *(TypeScript)* · `Zod` · `Yup` · `Valibot` · `Joi` *(JavaScript)* · `None` *(Dart/Kotlin/Swift)* |
| PWA Support      | *(filtered by platform)* `None` · `Basic (manifest + service worker)` · `Full offline` · `Push notifications` *(Web only)* · `None` *(Mobile/Desktop)* |
| Real-time        | `WebSocket` · `SSE` · `Polling` · `None`                                                                        |
| Image Optim.     | *(filtered by platform)* `Next/Image (built-in)` · `Cloudinary` · `Imgix` · `Sharp (self-hosted)` · `CDN transform` · `None` *(Web only)* · `None` *(Mobile/Desktop)* |
| Auth Flow        | `Redirect (OAuth/OIDC)` · `Modal login` · `Magic link` · `Passwordless` · `Social only`                        |
| Error Boundary   | *(filtered by framework — see below)*                                                                           |
| Bundle Optim.    | *(filtered by language)* `Code splitting (route-based)` · `Dynamic imports` · `Tree shaking only` · `None` *(TypeScript/JavaScript)* · `None` *(Dart/Kotlin/Swift)* |
| FE Testing       | *(filtered by language)* `Vitest` · `Jest` · `Testing Library` · `Storybook` · `None` *(TypeScript/JavaScript)* · `None` *(Dart/Kotlin/Swift)* |
| Linter           | *(filtered by language)* `ESLint + Prettier` · `Biome` · `oxlint` · `Stylelint` · `Custom` · `None` *(TypeScript/JavaScript)* · `Custom` · `None` *(Dart/Kotlin/Swift)* |

Framework options per language:

| Language   | Frameworks                                                       |
|------------|------------------------------------------------------------------|
| TypeScript | `React` · `Vue` · `Svelte` · `Angular` · `Solid` · `Qwik` · `HTMX` |
| JavaScript | `React` · `Vue` · `Svelte` · `Angular` · `Solid` · `Qwik` · `HTMX` |
| Dart       | `Flutter`                                                        |
| Kotlin     | `Jetpack Compose` · `KMP (Compose Multiplatform)`                |
| Swift      | `SwiftUI` · `UIKit`                                              |

Meta-framework options per framework:

| Framework | Meta-frameworks                          |
|-----------|------------------------------------------|
| React     | `Next.js` · `Remix` · `Astro` · `None`  |
| Vue       | `Nuxt` · `Astro` · `None`               |
| Svelte    | `SvelteKit` · `Astro` · `None`          |
| Solid     | `Astro` · `None`                        |
| All others | `None`                                  |

Component library options per framework:

| Framework | Component Libraries                                                                   |
|-----------|---------------------------------------------------------------------------------------|
| React     | `shadcn/ui` · `Radix` · `Material UI` · `Ant Design` · `Headless UI` · `DaisyUI` · `None` · `Custom` |
| Vue       | `Material UI` · `None` · `Custom`                                                     |
| Angular   | `Material UI` · `None` · `Custom`                                                     |
| All others | `None` · `Custom`                                                                    |

State management options per framework:

| Framework | State Management                                             |
|-----------|--------------------------------------------------------------|
| React     | `React Context` · `Zustand` · `Redux Toolkit` · `Jotai` · `None` |
| Vue       | `Pinia` · `None`                                             |
| Svelte    | `Svelte stores` · `None`                                     |
| Angular / Solid / Qwik | `Signals` · `None`                            |
| All others | `None`                                                      |

Data fetching options per framework:

| Framework | Data Fetching                                                              |
|-----------|----------------------------------------------------------------------------|
| React     | `TanStack Query` · `SWR` · `Apollo Client` · `tRPC client` · `RTK Query` · `Native fetch` |
| Vue       | `TanStack Query` · `Apollo Client` · `Native fetch`                        |
| Svelte    | `TanStack Query` · `SWR` · `Native fetch`                                  |
| Angular   | `Apollo Client` · `Native fetch`                                           |
| Solid     | `TanStack Query` · `Native fetch`                                          |
| All others | `Native fetch`                                                            |

Form handling options per framework:

| Framework | Form Handling                                           |
|-----------|---------------------------------------------------------|
| React     | `React Hook Form` · `Formik` · `Zod + native` · `None` |
| Vue       | `Vee-Validate` · `Zod + native` · `None`                |
| Svelte / Angular / Solid / Qwik | `Zod + native` · `None`        |
| All others | `None`                                                 |

Error boundary options per framework:

| Framework | Error Boundary                                                       |
|-----------|----------------------------------------------------------------------|
| React     | `React Error Boundary` · `Global try-catch` · `Framework default` · `Custom` |
| Vue / Angular / Svelte / Solid / Qwik | `Global try-catch` · `Framework default` · `Custom` |
| HTMX      | `Global try-catch` · `Custom`                                        |
| Flutter / Compose / SwiftUI / UIKit | `Framework default` · `Custom` |

---

### 4.2 Theming Sub Tab

| Field         | Input                                                                   |
|---------------|-------------------------------------------------------------------------|
| Dark Mode     | `None` · `Toggle (user preference)` · `System preference` · `Dark only` |
| Border Radius | `Sharp (0)` · `Subtle (4px)` · `Rounded (8px)` · `Pill (999px)` · `Custom` |
| Spacing       | `Compact (4px base)` · `Default (8px base)` · `Spacious (12px base)`   |
| Elevation     | `Shadows` · `Borders` · `Both` · `Flat`                                |
| Motion        | `None` · `Subtle transitions` · `Animated (spring/ease)`               |
| Vibe          | `Professional` · `Playful` · `Minimal` · `Bold` · `Elegant` · `Technical` · `Creative` · `Friendly` · `Serious` · `Modern` |
| Colors        | Free text (hex, hsl, or description)                                    |
| Description   | Free text (prose description of visual feel)                           |

---

### 4.3 Pages Sub Tab

#### Adding a Page

| Field          | Input                                                                      |
|----------------|----------------------------------------------------------------------------|
| Name           | e.g., `Dashboard`, `User Profile`, `Checkout`                              |
| Route          | e.g., `/dashboard`, `/users/:id`, `/checkout`                              |
| Auth Required  | `false` · `true`                                                           |
| Layout         | `Default` · `Sidebar` · `Full-width` · `Blank` · `Custom (specify)`        |
| Description    | Free text — what this page does, its purpose                               |
| Core Actions   | Free text list of what the user can do on this page                        |
| Loading        | `Skeleton` · `Spinner` · `Progressive` · `Instant (SSR/SSG)`               |
| Error Handling | `Inline` · `Toast` · `Error boundary / fallback page` · `Retry`            |
| Auth Roles     | Multi-select from roles defined in **Backend → Auth**                       |
| Linked Pages   | Multi-select from other page routes                                        |

---

### 4.4 Navigation Sub Tab

| Field       | Options / Input                                                      |
|-------------|----------------------------------------------------------------------|
| Nav Type    | `Top bar` · `Sidebar` · `Bottom tabs (mobile)` · `Hamburger menu` · `Combined` |
| Breadcrumbs | `false` · `true`                                                     |
| Auth-Aware  | `false` · `true` — show/hide items based on auth state               |

---

### 4.5 I18N Sub Tab

| Field               | Options / Input                                                                      |
|---------------------|--------------------------------------------------------------------------------------|
| Enabled             | `false` · `true`                                                                     |
| Default Locale      | Dropdown — `en` · `en-US` · `en-GB` · `en-AU` · `en-CA` · `fr` · `fr-FR` · `fr-CA` · `de` · `de-DE` · `de-AT` · `es` · `es-ES` · `es-MX` · `es-AR` · `pt` · `pt-BR` · `pt-PT` · `it` · `nl` · `nl-NL` · `pl` · `ru` · `ja` · `zh` · `zh-CN` · `zh-TW` · `ko` · `ar` · `hi` · `tr` · `sv` · `da` · `fi` · `nb` · `cs` · `hu` · `ro` · `vi` · `th` · `id` · `ms` · `uk` · `he` |
| Supported Locales   | Multi-select from the same locale list above                                         |
| I18N Library        | `i18next` · `next-intl` · `react-i18next` · `LinguiJS` · `vue-i18n` · `Custom` · `None` |
| Timezone Handling   | `UTC always` · `User preference` · `Auto-detect` · `Manual`                          |

---

### 4.6 A11Y / SEO Sub Tab

| Field            | Options / Input                                                                             |
|------------------|---------------------------------------------------------------------------------------------|
| WCAG Level       | `A` · `AA` · `AAA` · `None`                                                                 |
| SEO Rendering    | `SSR` · `SSG` · `ISR` · `Prerender` · `None`                                                |
| Sitemap          | `false` · `true`                                                                            |
| Meta Tags        | `Manual` · `Automatic (react-helmet)` · `Framework-native` · `None`                         |
| Analytics        | `PostHog` · `Google Analytics 4` · `Plausible` · `Mixpanel` · `Segment` · `Custom` · `None` |
| Frontend RUM     | `Sentry` · `Datadog RUM` · `LogRocket` · `New Relic Browser` · `Custom` · `None`            |

---

### 4.7 Assets Sub Tab

> *(repeatable — attach design assets, mockups, or inspiration references)*

| Field       | Options / Input                                                                       |
|-------------|---------------------------------------------------------------------------------------|
| Name        | Free text identifier                                                                  |
| Path        | File path or URL                                                                      |
| Asset Type  | `image` · `icon` · `font` · `video` · `mockup` · `moodboard`                         |
| Format      | `png` · `jpg` · `svg` · `gif` · `mp4` · `pdf` · `figma` · `sketch` · `other`         |
| Usage       | `project` *(used in the build)* · `inspiration` *(reference only)*                   |
| Description | Free text                                                                             |

---

## 5 · Infrastructure Tab

### 5.1 Networking Sub Tab

| Field           | Options / Input                                                             |
|-----------------|-----------------------------------------------------------------------------|
| DNS Provider    | `Cloudflare` · `Route53` · `Cloud DNS` · `Other`                            |
| TLS/SSL         | `Let's Encrypt` · `Cloudflare` · `ACM` · `Manual` · `None (dev)`            |
| Reverse Proxy   | `Nginx` · `Caddy` · `Traefik` · `Cloudflare Tunnel` · `Cloud LB`           |
| CDN             | `Cloudflare` · `CloudFront` · `Fastly` · `Vercel Edge` · `None`             |
| Primary Domain  | Free text (e.g., `myapp.com`)                                               |
| Domain Strategy | `Subdomain per service` · `Path-based routing` · `Single domain` · `Custom` |
| CORS Enforced   | `Reverse proxy (Nginx/Caddy)` · `Application-level` · `CDN/WAF` · `Both`   |
| SSL Cert Mgmt   | `Auto-renew (certbot/ACME)` · `Managed (cloud provider)` · `Manual` · `Cloudflare proxy` |

---

### 5.2 CI/CD Sub Tab

| Field              | Options / Input                                                             |
|--------------------|-----------------------------------------------------------------------------|
| CI/CD Platform     | `GitHub Actions` · `GitLab CI` · `Jenkins` · `CircleCI` · `ArgoCD` · `Tekton` |
| Container Registry | `Docker Hub` · `GHCR` · `ECR` · `GCR` · `Self-hosted`                      |
| Deploy Strategy    | `Rolling` · `Blue-green` · `Canary` · `Recreate`                            |
| IaC Tool           | `Terraform` · `Pulumi` · `CloudFormation` · `Ansible` · `None`              |
| Secrets Mgmt       | `GitHub Secrets` · `HashiCorp Vault` · `AWS Secrets Manager` · `GCP Secret Manager` · `None` |
| Container Runtime  | `Node Alpine` · `Go scratch` · `Python slim` · `Distroless` · `Ubuntu` · `Custom` |
| Backup/DR          | `Cross-region replication` · `Daily snapshots` · `Managed provider DR` · `None` |

---

### 5.3 Observability Sub Tab

| Field          | Options / Input                                                                    |
|----------------|------------------------------------------------------------------------------------|
| Logging        | `Loki + Grafana` · `ELK Stack` · `CloudWatch` · `Datadog` · `Stdout/file`         |
| Metrics        | `Prometheus + Grafana` · `Datadog` · `CloudWatch` · `New Relic` · `None`           |
| Tracing        | `OpenTelemetry + Jaeger` · `OpenTelemetry + Tempo` · `Datadog APM` · `None`        |
| Error Tracking | `Sentry` · `Datadog` · `Rollbar` · `Built-in` · `None`                             |
| Health Checks  | `false` · `true` — auto-generate `/health` and `/ready` endpoints per service unit |
| Alerting       | `Grafana Alerting` · `PagerDuty` · `OpsGenie` · `CloudWatch Alarms` · `None`       |
| Log Retention  | `7 days` · `30 days` · `90 days` · `1 year` · `Indefinite`                         |

---

### 5.4 Environments Sub Tab

| Field              | Options / Input                                                                               |
|--------------------|-----------------------------------------------------------------------------------------------|
| Stages             | `dev + prod` · `dev + staging + prod` · `dev + qa + staging + prod` · `dev + staging + qa + preview + prod` · `Custom` |
| Promotion Pipeline | `Dev → Staging → Prod` · `Dev → QA → Staging → Prod` · `Dev → Prod (direct)` · `Manual` · `None` |
| Secret Keys        | `Per-environment` · `Shared base + overrides` · `Fully shared` · `None`                       |
| DB Migrations      | `Auto on deploy` · `Manual CI step` · `Flyway` · `Liquibase` · `Atlas` · `golang-migrate` · `None` |
| DB Seeding         | `Automatic (fixtures)` · `Manual` · `None`                                                    |
| Preview Envs       | `false` · `true`                                                                              |

---

## 6 · Cross-Cutting Concerns Tab

### 6.1 Testing Strategy Sub Tab

| Field             | Options / Input                                                                  |
|-------------------|----------------------------------------------------------------------------------|
| Unit Testing      | Dynamically filtered by backend language — see table below                       |
| Integration Tests | `Testcontainers` · `Docker Compose` · `In-memory fakes` · `None`                |
| E2E Testing       | Dynamically filtered by frontend platform/framework — see table below            |
| API Testing       | `Bruno` · `Hurl` · `Postman/Newman` · `REST Client` · `None`                    |
| Load Testing      | `k6` · `Artillery` · `JMeter` · `None` *(+ `Locust` when Python is a backend language)* |
| Contract Testing  | `Pact` · `Schemathesis` · `Dredd` · `None`                                      |

Unit testing options per backend language:

| Language        | Unit Testing Tools                                                         |
|-----------------|----------------------------------------------------------------------------|
| Go              | `Go testing` · `Testify` · `Other`                                         |
| TypeScript/Node | `Jest` · `Vitest` · `Other`                                                |
| Python          | `pytest` · `unittest` · `Other`                                            |
| Java            | `JUnit` · `TestNG` · `Other`                                               |
| Kotlin          | `JUnit` · `Kotest` · `Other`                                               |
| C#/.NET         | `xUnit` · `NUnit` · `MSTest` · `Other`                                     |
| Rust            | `cargo test` · `Other`                                                     |
| Ruby            | `RSpec` · `minitest` · `Other`                                             |
| PHP             | `PHPUnit` · `Pest` · `Other`                                               |
| *(no language)* | `Jest` · `Vitest` · `pytest` · `Go testing` · `JUnit` · `xUnit` · `Other` |

E2E testing options per frontend platform/framework:

| Platform / Framework         | E2E Tools                                       |
|------------------------------|-------------------------------------------------|
| Web (any framework)          | `Playwright` · `Cypress` · `Selenium` · `None`  |
| Flutter / Dart               | `Flutter Driver` · `Integration Test` · `None`  |
| Jetpack Compose / Android    | `Espresso` · `UI Automator` · `None`            |
| SwiftUI / UIKit / iOS        | `XCUITest` · `EarlGrey` · `None`                |
| *(no frontend configured)*   | `None`                                          |

---

### 6.2 Documentation Sub Tab

| Field           | Options / Input                                                        |
|-----------------|------------------------------------------------------------------------|
| API Docs        | `OpenAPI/Swagger` · `GraphQL Playground` · `gRPC reflection` · `None`  |
| Auto-generation | `false` · `true` — generate specs from code annotations                |
| Changelog       | `Conventional Commits` · `Manual` · `None`                             |

---

### 6.3 Standards Sub Tab

| Field              | Options / Input                                                                  |
|--------------------|----------------------------------------------------------------------------------|
| Branch Strategy    | `GitHub Flow` · `GitFlow` · `Trunk-based` · `Custom`                            |
| Dependency Updates | `Dependabot` · `Renovate` · `Manual` · `None`                                   |
| Code Review        | `Required (1 approval)` · `Required (2 approvals)` · `Optional` · `None`        |
| Feature Flags      | `LaunchDarkly` · `Unleash` · `Flagsmith` · `Custom (env vars)` · `None`          |
| Uptime SLO         | `99.9%` · `99.95%` · `99.99%` · `Custom`                                         |
| Latency P99        | `<50ms` · `<100ms` · `<200ms` · `<500ms` · `<1s` · `Custom`                     |

---

## 7 · Realize Tab

> Configure code generation options before running the realization engine.

| Field           | Options / Input                                                                                                   |
|-----------------|-------------------------------------------------------------------------------------------------------------------|
| App Name        | Free text (e.g., `my-app`) — used as the output project name                                                      |
| Output Dir      | Free text path (default: `.`)                                                                                     |
| Model           | `claude-haiku-4-5-20251001` · `claude-sonnet-4-6` · `claude-opus-4-6`                                            |
| Concurrency     | `1` · `2` · `4` · `8` — max parallel tasks                                                                        |
| Verify          | `true` · `false` — run language verifier after each generated file                                                |
| Dry Run         | `false` · `true` — print task plan without invoking agents                                                        |

**Per-section model overrides** *(optional — override the global model for specific pillars)*:

| Field           | Options / Input                                                                               |
|-----------------|-----------------------------------------------------------------------------------------------|
| Backend Model   | `default` or provider·tier (e.g., `Claude · Sonnet`, `Gemini · Flash`) from configured providers |
| Data Model      | `default` or provider·tier                                                                    |
| Contracts Model | `default` or provider·tier                                                                    |
| Frontend Model  | `default` or provider·tier                                                                    |
| Infra Model     | `default` or provider·tier                                                                    |
| Crosscut Model  | `default` or provider·tier                                                                    |

---

## Dependency & Reference Graph

The tabs form a directed dependency graph — this is the intended order of definition:

```
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains)
    ↓
Contracts (DTOs reference Domains, Endpoints reference Service Units)
    ↓
Frontend (Pages reference Endpoints + DTOs; roles from Backend Auth)
    ↓
Infrastructure (references all deployable units)
    ↓
Cross-Cutting (references everything)
    ↓
Realize (consumes full manifest → generates code)
```

> **However**, the UI allows **non-linear editing** — users can start anywhere and link entities later. Empty references show as "unlinked" placeholders that can be resolved.
