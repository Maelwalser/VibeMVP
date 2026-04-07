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
| **Environment**  | A named deployment target (e.g. `dev`, `staging`, `prod`) defined in **Infrastructure → Environments** |

---

## 0 · Description

Free-text project description textarea. Describe the system in natural language before filling the structured pillars. Content is saved to `manifest.json` under the `description` field and injected into the code-generation prompt as high-level context.

---

## 1 · Backend Tab

### 1.1 Architecture Pattern *(top-level selector)*

| Option              | Sub-tabs Unlocked                                                                     |
|---------------------|---------------------------------------------------------------------------------------|
| Monolith            | ENV · SERVICES · JOBS · SECURITY · AUTH                                               |
| Modular Monolith    | ENV · SERVICES · STACK CONFIG · COMM · JOBS · SECURITY · AUTH                        |
| Microservices       | ENV · SERVICES · STACK CONFIG · COMM · API GW · JOBS · SECURITY · AUTH               |
| Event-Driven        | ENV · SERVICES · STACK CONFIG · COMM · MESSAGING · JOBS · SECURITY · AUTH             |
| Hybrid              | ENV · SERVICES · STACK CONFIG · COMM · MESSAGING · API GW · JOBS · SECURITY · AUTH   |

> Selecting an architecture pattern sets the **communication defaults** and **deployment topology** but every option shares the same service-unit definition shape below.

---

### 1.2 Environment Sub Tab

> For **Monolith** only — defines the single shared language/framework and links the monolith to an infrastructure environment. For multi-service architectures this tab only shows global health dependencies; per-service tech choices live in **Stack Config** and the **Services** form.

| Field          | Options / Input                                                                       |
|----------------|---------------------------------------------------------------------------------------|
| Language       | *(Monolith only)* `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Lang Version   | *(Monolith only)* Dynamically filtered by language                                    |
| Framework      | *(Monolith only)* Dynamically filtered by language — see §1.3 for options             |
| FW Version     | *(Monolith only)* Dynamically filtered by language + framework                        |
| Environment    | *(Monolith only)* Select from environments defined in **Infrastructure → Environments** |
| Health Deps    | Multi-select from defined databases/services — global health-check dependencies       |

---

### 1.3 Stack Config Sub Tab

> *(Non-monolith architectures only)* Reusable language/framework combinations that services and job queues can reference instead of defining their own stack inline.

#### Adding a Stack Config

| Field           | Input                                                                           |
|-----------------|---------------------------------------------------------------------------------|
| Name            | Free text identifier (e.g., `go-fiber`, `node-nestjs`)                          |
| Language        | `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Lang Version    | Dynamically filtered by language                                                |
| Framework       | Dynamically filtered by language — see §1.4 for options                         |
| FW Version      | Dynamically filtered by language + framework                                    |

---

### 1.4 Service Units Sub Tab

#### Adding / Editing a Service Unit

| Field             | Input                                                                                                         |
|-------------------|---------------------------------------------------------------------------------------------------------------|
| Name              | Free text identifier (e.g., `auth-service`, `billing-module`)                                                 |
| Responsibility    | Free text description of what this unit owns                                                                   |
| Stack Config      | *(non-monolith)* Select from defined Stack Configs — overrides inline language/framework when set             |
| Language          | `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Lang Version      | Dynamically filtered by language                                                                              |
| Framework         | *(dynamically filtered by language — see below)*                                                               |
| FW Version        | Dynamically filtered by language + framework                                                                  |
| Technologies      | Multi-select: `WebSocket` · `gRPC` · `REST` · `GraphQL` · `SSE` · `tRPC` · `MQTT` · `Kafka consumer`         |
| Pattern Tag       | *(Hybrid only)* `Monolith part` · `Modular module` · `Microservice` · `Event processor` · `Serverless function` |
| Health Deps       | Multi-select from defined databases — databases this service depends on for health checks                     |
| Error Format      | Dynamically filtered by selected technologies — see table below                                               |
| Service Discovery | `DNS-based` · `Consul` · `Kubernetes DNS` · `Eureka` · `Static config` · `None`                               |
| Environment       | Select from environments defined in **Infrastructure → Environments**                                         |

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

Error format options per selected technology:

| Technology  | Error Formats                                              |
|-------------|------------------------------------------------------------|
| REST        | `RFC 7807 (Problem Details)` · `Custom JSON envelope`     |
| GraphQL     | `GraphQL spec errors` · `Custom extensions`               |
| gRPC        | `gRPC status codes` · `google.rpc.Status`                 |
| WebSocket   | `Custom JSON envelope`                                    |
| SSE         | `Custom JSON envelope`                                    |
| tRPC        | `tRPC error format`                                       |
| *(default)* | `RFC 7807 (Problem Details)` · `Custom JSON envelope` · `Platform default` |

> Options from each active technology are unioned; `Platform default` is always appended last.

---

### 1.5 Communication Sub Tab

> Shared across **all** multi-service architecture patterns. Each link is a directed edge in the system graph.

| Field         | Input                                                                                                    |
|---------------|----------------------------------------------------------------------------------------------------------|
| From          | Free text (or select a service unit name)                                                                |
| To            | Free text (or select a service unit name)                                                                |
| Direction     | `Unidirectional (→)` · `Bidirectional (↔)` · `Pub/Sub (fan-out)`                                        |
| Protocol      | `REST (HTTP)` · `gRPC` · `GraphQL` · `WebSocket` · `Message Queue` · `Event Bus` · `Internal (in-process)` |
| Trigger / Flow | Free text description of **when** this communication happens                                            |
| Sync/Async    | `Synchronous` · `Asynchronous` · `Fire-and-forget`                                                      |
| Resilience    | Multi-select: `Circuit breaker` · `Retry with backoff` · `Timeout` · `Bulkhead` · `None`                |
| Payload DTO   | Multi-select from defined DTOs — request/payload data types for this link                               |
| Response DTO  | *(Bidirectional only)* Multi-select from defined DTOs — response data types                             |

---

### 1.6 Messaging Sub Tab

> Visible for Event-Driven and Hybrid patterns.

**Broker Configuration**

| Field                | Options / Input                                                                                       |
|----------------------|-------------------------------------------------------------------------------------------------------|
| Broker Technology    | `Kafka` · `NATS` · `RabbitMQ` · `Redis Streams` · `AWS SQS/SNS` · `Google Pub/Sub` · `Azure Service Bus` · `Pulsar` |
| Deployment           | Context-aware — filtered by broker tech + cloud provider; see table below                             |
| Serialization Format | `JSON` · `Protobuf` · `Avro` · `MessagePack` · `CloudEvents`                                          |
| Delivery Guarantee   | `At-most-once` · `At-least-once` · `Exactly-once`                                                     |
| Environment          | Select from environments defined in **Infrastructure → Environments**                                  |

Deployment options by broker + cloud provider:

| Cloud Provider | Broker Tech        | Deployment Options                          |
|----------------|--------------------|---------------------------------------------|
| AWS            | Kafka              | `AWS MSK (managed)` · `Self-hosted (EC2/K8s)` |
| AWS            | RabbitMQ           | `Amazon MQ` · `Self-hosted`                 |
| AWS            | Redis Streams      | `ElastiCache` · `Self-hosted`               |
| AWS            | NATS               | `Synadia Cloud` · `Self-hosted`             |
| AWS            | AWS SQS/SNS        | `AWS SQS/SNS (managed)`                     |
| GCP            | Kafka              | `Confluent Cloud` · `Self-hosted`           |
| GCP            | Google Pub/Sub     | `Google Pub/Sub (managed)`                  |
| Azure          | RabbitMQ / Azure SB| `Azure Service Bus (managed)` · `Self-hosted` |
| Other (NATS)   | NATS               | `Synadia Cloud` · `Self-hosted` · `Embedded`|
| *(default)*    | *(any)*            | `Managed (cloud)` · `Self-hosted` · `Embedded` |

**Event Catalog** *(repeatable)*

| Field             | Input                                                      |
|-------------------|------------------------------------------------------------|
| Event Name        | e.g., `order.placed`, `user.registered`                    |
| Publisher Service | Select from defined service units                          |
| Consumer Service  | Select from defined service units                          |
| DTO               | Select from defined DTOs — the event payload type          |
| Description       | When/why this event fires                                  |

---

### 1.7 API Gateway Sub Tab

> Auto-suggested for Microservices and Hybrid patterns.

| Field              | Options / Input                                                                          |
|--------------------|------------------------------------------------------------------------------------------|
| Environment        | Select from environments defined in **Infrastructure → Environments**                     |
| Gateway Technology | Context-aware — filtered by orchestrator + cloud provider; see table below                |
| Routing Strategy   | `Path-based` · `Header-based` · `Domain-based`                                           |
| Features           | Multi-select: `Rate limiting` · `JWT validation` · `SSL termination` · `Load balancing` · `Request caching` · `Logging & tracing` · `Request transformation` · `CORS handling` · `IP allowlist/blocklist` · `Circuit breaking` · `Health checks` |
| Endpoints          | Multi-select from defined endpoints (Contracts tab)                                      |

Gateway technology options by environment:

| Orchestrator / Cloud | Technologies                                             |
|----------------------|----------------------------------------------------------|
| Kubernetes / K3s     | `Kong` · `Traefik` · `NGINX Ingress` · `Envoy` · `Custom (specify)` · `None` |
| Docker Compose       | `Traefik` · `NGINX` · `Custom (specify)` · `None`       |
| AWS                  | `AWS API Gateway` · `Kong` · `Custom (specify)` · `None`|
| GCP                  | `Cloudflare Workers` · `Custom (specify)` · `None`      |
| *(default)*          | `Kong` · `Traefik` · `NGINX` · `Envoy` · `AWS API Gateway` · `Cloudflare Workers` · `Custom (specify)` · `None` |

---

### 1.8 Jobs Sub Tab

> Background/scheduled job queue configuration. *(repeatable — add multiple queues)*

| Field          | Options / Input                                                                          |
|----------------|------------------------------------------------------------------------------------------|
| Name           | Free text identifier for this queue                                                      |
| Description    | Free text description of what jobs this queue handles                                    |
| Stack Config   | *(non-monolith with configs defined)* Select from Stack Configs                          |
| Technology     | Dynamically filtered by backend language — see table below                               |
| Concurrency    | Free text (default: `10`)                                                                |
| Max Retries    | Free text (default: `3`)                                                                 |
| Retry Policy   | `Exponential backoff` · `Fixed interval` · `Linear backoff` · `None`                     |
| Dead Letter Q  | `false` · `true`                                                                         |
| Worker Service | Select from defined service units — which service hosts this worker                      |
| Payload DTO    | Select from defined DTOs — the job payload type                                          |

Job queue technology options per backend language:

| Language        | Technologies                                    |
|-----------------|-------------------------------------------------|
| Go              | `Asynq` · `River` · `Temporal` · `Faktory` · `Custom` |
| TypeScript/Node | `BullMQ` · `Temporal` · `Custom`               |
| Python          | `Celery` · `Temporal` · `Custom`               |
| Ruby            | `Sidekiq` · `Temporal` · `Custom`              |
| Java            | `Temporal` · `Custom`                          |
| Kotlin          | `Temporal` · `Custom`                          |
| C#/.NET         | `Hangfire` · `Temporal` · `Custom`             |
| Rust            | `Temporal` · `Custom`                          |
| PHP             | `Laravel Queues` · `Temporal` · `Custom`       |
| Elixir          | `Oban` · `Temporal` · `Custom`                 |
| *(no language)* | `Temporal` · `BullMQ` · `Sidekiq` · `Celery` · `Faktory` · `Asynq` · `River` · `Custom` |

**Cron Jobs** *(repeatable within each queue)*

| Field    | Options / Input                                                              |
|----------|------------------------------------------------------------------------------|
| Name     | Free text identifier (e.g., `nightly-cleanup`)                               |
| Schedule | Cron expression (e.g., `0 2 * * *`)                                         |
| Handler  | Free text — handler function / method name                                   |
| Timeout  | Free text — maximum execution time (e.g., `30s`, `5m`)                      |

---

### 1.9 Security Sub Tab

> WAF provider options are narrowed based on the cloud provider configured in **Infrastructure → Environments**.

| Field               | Options / Input                                                                                 |
|---------------------|-------------------------------------------------------------------------------------------------|
| WAF Provider        | `Cloudflare WAF` · `AWS WAF` · `Cloud Armor` · `Azure WAF` · `ModSecurity` · `NGINX ModSec` · `None` |
| WAF Ruleset         | `OWASP Core Rule Set` · `Managed rules` · `Custom` · `None`                                     |
| CAPTCHA             | `hCaptcha` · `reCAPTCHA v2` · `reCAPTCHA v3` · `Cloudflare Turnstile` · `None`                 |
| Bot Protection      | `Cloudflare Bot Management` · `Imperva` · `DataDome` · `Custom` · `None`                        |
| Rate Limit Strategy | `Token bucket (Redis)` · `Sliding window` · `Fixed window` · `Leaky bucket` · `API Gateway` · `None` |
| Rate Limit Backend  | `Redis` · `Memcached` · `In-memory` · `None`                                                    |
| DDoS Protection     | `CDN-level (Cloudflare)` · `Provider-managed` · `None`                                          |
| Internal mTLS       | `Enabled` · `Disabled` — mutual TLS for internal service-to-service communication               |

> **Rate Limit Backend** and **Rate Limit Strategy** are hidden when strategy is `None` or `API Gateway`.  
> **Internal mTLS** is hidden when Rate Limit Strategy is `None` or `API Gateway`.

WAF provider options by cloud provider:

| Cloud Provider | WAF Options                                                      |
|----------------|------------------------------------------------------------------|
| GCP            | `Cloud Armor` · `Cloudflare WAF` · `ModSecurity` · `NGINX ModSec` · `None` |
| Azure          | `Azure WAF` · `Cloudflare WAF` · `ModSecurity` · `NGINX ModSec` · `None`   |
| *(default)*    | `Cloudflare WAF` · `AWS WAF` · `Cloud Armor` · `Azure WAF` · `ModSecurity` · `NGINX ModSec` · `None` |

---

### 1.10 Auth & Identity Sub Tab

| Field                  | Options / Input                                                                               |
|------------------------|-----------------------------------------------------------------------------------------------|
| Auth Strategy          | Multi-select: `JWT (stateless)` · `Session-based` · `OAuth 2.0 / OIDC` · `API Keys` · `mTLS` · `None` |
| Identity Provider      | `Self-managed` · `Auth0` · `Clerk` · `Supabase Auth` · `Firebase Auth` · `Keycloak` · `AWS Cognito` · `Other` |
| Service Unit           | *(Self-managed or Keycloak only)* Select from defined service units — which service handles auth |
| Authorization Model    | `RBAC` · `ABAC` · `ACL` · `ReBAC` · `Policy-based (OPA/Cedar)` · `Custom`                    |
| Token Storage (client) | Multi-select (filtered by selected strategies): `HttpOnly cookie` · `Authorization header (Bearer)` · `WebSocket protocol header` · `Other` |
| Session Mgmt           | *(Session-based strategy only)* `Stateless (JWT only)` · `Server-side sessions (Redis)` · `Database sessions` · `None` |
| Refresh Token          | `None` · `Rotating` · `Non-rotating` · `Sliding window`                                       |
| MFA Support            | `None` · `TOTP` · `SMS` · `Email` · `Passkeys/WebAuthn`                                       |

> **Token Storage** options are derived from the active strategies:  
> - `JWT (stateless)` → HttpOnly cookie · Bearer header · Other  
> - `Session-based` → HttpOnly cookie  
> - `OAuth 2.0 / OIDC` → HttpOnly cookie · Bearer header

**Permissions** *(repeatable — define named permission strings before defining roles)*

| Field       | Input                                                              |
|-------------|--------------------------------------------------------------------|
| Name        | e.g., `users:read`, `orders:write`, `reports:export`               |
| Description | What this permission allows                                        |

**Roles** *(repeatable — full CRUD role editor)*

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
| Strategy      | Multi-select: `Cache-aside` · `Read-through` · `Write-through` · `Write-behind` · `CDN purge` |
| Invalidation  | `TTL-based` · `Event-driven` · `Manual` · `Hybrid`                           |
| TTL           | `30s` · `1m` · `5m` · `15m` · `1h` · `24h` · `Custom`                       |
| Entities      | Multi-select from domains (populated dynamically)                            |

---

### 2.4 File / Object Storage Sub Tab

> *(repeatable — add multiple storage buckets)*

| Field         | Options / Input                                                                           |
|---------------|-------------------------------------------------------------------------------------------|
| Technology    | `S3` · `GCS` · `Azure Blob` · `MinIO` · `Cloudflare R2` · `Local disk`                   |
| Environment   | Select from environments defined in **Infrastructure → Environments**                     |
| Purpose       | Free text (e.g., "User avatars", "Document uploads", "Backups")                           |
| Access        | `Public (CDN-fronted)` · `Private (signed URLs)` · `Internal only`                        |
| Max Size      | `1 MB` · `5 MB` · `10 MB` · `25 MB` · `50 MB` · `100 MB` · `500 MB` · `1 GB` · `Unlimited` |
| Domains       | Multi-select from domains (which domains store files here)                                |
| TTL Minutes   | `30` · `60` · `1440` · `10080` · `Custom` (for signed URL expiry)                        |
| Allowed Types | Multi-select: `image/*` · `application/pdf` · `video/*` · `audio/*` · `text/*` · `application/json` |

---

### 2.5 Governance Sub Tab

*(repeatable — add multiple named governance policies, each applying to a set of databases)*

| Field               | Options / Input                                                                               |
|---------------------|-----------------------------------------------------------------------------------------------|
| Name                | Free text identifier (e.g., `primary-policy`, `audit-retention`)                             |
| Applies To          | Multi-select from defined databases                                                           |
| Migration Tool      | Dynamically filtered by backend language — see table below                                    |
| Backup Strategy     | `Automated daily` · `Point-in-time recovery` · `Manual snapshots` · `Managed provider` · `None` |
| Search Tech         | `Elasticsearch` · `Meilisearch` · `Algolia` · `Typesense` · `None`                           |
| Retention Policy    | `30 days` · `90 days` · `1 year` · `3 years` · `7 years` · `Indefinite` · `Custom`           |
| Delete Strategy     | `Soft-delete` · `Hard-delete` · `Archival` · `Soft + periodic purge`                          |
| PII Encryption      | `Field-level AES-256` · `Full database encryption` · `Application-level` · `None`             |
| Compliance          | Multi-select: `GDPR` · `HIPAA` · `SOC2 Type II` · `PCI-DSS` · `ISO-27001` · `CCPA` · `PIPEDA` |
| Data Residency      | `US` · `EU` · `APAC` · `US + EU` · `Global` · `Custom`                                       |
| Archival Storage    | Cloud-provider-aware — `S3 Glacier` (AWS) · `GCS Archive` (GCP) · `Azure Archive` (Azure) · `On-premise` · `None` |

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

---

## 3 · Contracts Tab

### 3.1 DTOs Sub Tab

#### Adding a DTO

| Field            | Input                                                                                     |
|------------------|-------------------------------------------------------------------------------------------|
| Name             | e.g., `CreateUserRequest`, `OrderSummaryResponse`, `UserRegisteredEvent`                  |
| Category         | `Request` · `Response` · `Event Payload` · `Shared/Common`                                |
| Source Domain(s) | Multi-select from **Data → Domains**                                                      |
| Protocol         | `REST/JSON` · `Protobuf` · `Avro` · `MessagePack` · `Thrift` · `FlatBuffers` · `Cap'n Proto` |
| Description      | What this DTO represents and when it's used                                               |

Protocol-specific DTO fields:

| Protocol     | Extra Fields                                                                             |
|--------------|------------------------------------------------------------------------------------------|
| Protobuf     | Package name · Syntax (`proto2` / `proto3`) · Options (free text)                        |
| Avro         | Namespace · Schema Registry (free text)                                                  |
| Thrift       | Namespace · Target Language                                                              |
| FlatBuffers  | Namespace                                                                                |
| Cap'n Proto  | Namespace                                                                                |

#### DTO Fields *(repeatable)*

| Field          | Options / Input                                                                                                       |
|----------------|-----------------------------------------------------------------------------------------------------------------------|
| Name           | e.g., `email`, `order_items`, `total`                                                                                 |
| Type           | `string` · `int` · `float` · `boolean` · `datetime` · `uuid` · `enum(values)` · `array(type)` · `nested(DTO)` · `map(key,value)` |
| Required       | `false` · `true`                                                                                                      |
| Nullable       | `false` · `true`                                                                                                      |
| Validation     | Multi-select: `required` · `min_length` · `max_length` · `min_value` · `max_value` · `email` · `url` · `regex` · `uuid` · `enum` · `phone` · `pattern` · `custom` |
| Default        | *(optional)* Default value expression                                                                                 |
| Notes          | *(optional)* Mapping notes, transformation hints                                                                      |
| Field Number   | *(Protobuf only)* Integer field number                                                                               |
| Proto Modifier | *(Protobuf only)* `optional` · `required` · `repeated`                                                               |
| JSON Name      | *(Protobuf only)* JSON field name override                                                                            |
| Field ID       | *(Thrift / Cap'n Proto only)* Integer field ID                                                                       |
| Thrift Modifier| *(Thrift only)* `optional` · `required`                                                                              |
| Deprecated     | *(FlatBuffers only)* `false` · `true`                                                                                 |

---

### 3.2 Endpoints / Operations Sub Tab

#### Adding an Endpoint

| Field           | Options / Input                                                             |
|-----------------|-----------------------------------------------------------------------------|
| Service Unit    | Select which backend unit exposes this                                      |
| Name / Path     | e.g., `POST /api/v1/users`, `getUser`, `UserService.Create`                 |
| Protocol        | `REST` · `GraphQL` · `gRPC` · `WebSocket message` · `Event` *(filtered by service technologies)* |
| Auth Required   | `false` · `true`                                                            |
| Auth Roles      | Multi-select from roles defined in **Backend → Auth**                       |
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

> Versioning strategy is now **per protocol** — each active endpoint protocol can have its own versioning approach.

| Field               | Options / Input                                                                |
|---------------------|--------------------------------------------------------------------------------|
| REST Strategy       | `URL path (/v1/)` · `Header (Accept-Version)` · `Query param` · `None`         |
| GraphQL Strategy    | `Schema versioning` · `Field deprecation` · `None`                             |
| gRPC Strategy       | `Package versioning` · `None`                                                  |
| WebSocket Strategy  | `URL path (/v1/)` · `Header` · `None`                                          |
| Event Strategy      | `Topic versioning` · `Schema registry versioning` · `None`                     |
| Current Version     | Free text (e.g., `v1`)                                                         |
| Deprecation Policy  | `None` · `Sunset header` · `Versioned removal notice` · `Changelog entry` · `Custom` |
| Pagination Strategy | `Cursor-based` · `Offset/limit` · `Keyset` · `Page number` · `None`            |

---

### 3.4 External APIs Sub Tab

> *(repeatable — define each third-party API dependency)*

| Field            | Options / Input                                                                               |
|------------------|-----------------------------------------------------------------------------------------------|
| Provider         | Free text (e.g., `Stripe`, `SendGrid`, `Twilio`)                                              |
| Responsibility   | Free text — what this integration does                                                        |
| Protocol         | `REST` · `GraphQL` · `gRPC` · `WebSocket` · `Webhook` · `SOAP`                               |
| Auth Mechanism   | `API Key` · `OAuth2 Client Credentials` · `OAuth2 PKCE` · `Bearer Token` · `Basic Auth` · `mTLS` · `None` |
| Failure Strategy | `Circuit Breaker` · `Retry with backoff` · `Fallback value` · `Timeout + fail` · `None`       |
| Base URL         | *(REST / general HTTP)* Free text                                                             |
| Rate Limit       | *(REST)* Free text (e.g., `100 req/min`)                                                      |
| Webhook Path     | *(REST / Webhook)* Free text (your inbound webhook endpoint)                                  |
| TLS Mode         | *(gRPC)* `Insecure` · `TLS` · `mTLS`                                                         |
| WS Subprotocol   | *(WebSocket)* Free text                                                                       |
| Message Format   | *(WebSocket)* Free text                                                                       |
| HMAC Header      | *(Webhook)* Free text — header containing the HMAC signature                                  |
| Retry Policy     | *(Webhook)* `Exponential backoff` · `Fixed interval` · `None`                                 |
| SOAP Version     | *(SOAP)* `1.1` · `1.2`                                                                        |

**Interactions** *(repeatable within each External API — one entry per API call/operation)*

| Field          | Input                                                                 |
|----------------|-----------------------------------------------------------------------|
| Name           | Free text operation name                                              |
| Path           | Free text (e.g., `/v1/charges`)                                       |
| Request DTO    | Select from DTOs filtered by protocol                                 |
| Response DTO   | Select from DTOs filtered by protocol                                 |
| HTTP Method    | *(REST)* `GET` · `POST` · `PUT` · `PATCH` · `DELETE`                 |
| GQL Operation  | *(GraphQL)* `Query` · `Mutation` · `Subscription`                    |
| Stream Type    | *(gRPC)* `Unary` · `Server stream` · `Client stream` · `Bidirectional` |
| WS Direction   | *(WebSocket)* `Client→Server` · `Server→Client` · `Bidirectional`   |

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
| Font          | Free text (font family name or stack)                                   |
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
| I18N Library        | *(filtered by framework — see table below)*                                              |
| Timezone Handling   | `UTC always` · `User preference` · `Auto-detect` · `Manual`                          |

I18N library options per framework:

| Framework  | I18N Libraries                                                                |
|------------|-------------------------------------------------------------------------------|
| React      | `react-i18next` · `next-intl` · `LinguiJS` · `i18next` · `Custom` · `None`  |
| Vue        | `vue-i18n` · `i18next` · `Custom` · `None`                                   |
| Svelte     | `svelte-i18n` · `i18next` · `Custom` · `None`                                |
| Angular    | `@angular/localize` · `ngx-translate` · `Custom` · `None`                    |
| Solid / Qwik | `i18next` · `Custom` · `None`                                              |
| All others | `i18next` · `Custom` · `None`                                                |

---

### 4.6 A11Y / SEO Sub Tab

| Field            | Options / Input                                                                             |
|------------------|---------------------------------------------------------------------------------------------|
| WCAG Level       | `A` · `AA` · `AAA` · `None`                                                                 |
| SEO Rendering    | `SSR` · `SSG` · `ISR` · `Prerender` · `None`                                                |
| Sitemap          | `false` · `true`                                                                            |
| Meta Tags        | `Manual` · `Framework-native` · `None`                                                      |
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

> Sub-tabs: **ENVIRONMENTS** · **NETWORKING** · **OBSERVABILITY** · **CI/CD**

### 5.1 Environments Sub Tab

> *(repeatable — define one entry per deployment environment, e.g. `dev`, `staging`, `prod`)* Environments defined here are referenced by services, messaging brokers, API gateways, and databases throughout the other tabs.

#### Adding an Environment

| Field            | Options / Input                                                                               |
|------------------|-----------------------------------------------------------------------------------------------|
| Name             | Free text (e.g., `dev`, `staging`, `prod`)                                                    |
| Compute Env      | `Bare Metal` · `VM` · `Containers (Docker)` · `Kubernetes` · `Serverless (FaaS)` · `PaaS`    |
| Cloud Provider   | `AWS` · `GCP` · `Azure` · `Cloudflare` · `Hetzner` · `Self-hosted` · `Other (specify)`        |
| Orchestrator     | Dynamically filtered by Compute Env — see table below                                         |
| Regions          | Multi-select: `us-east-1` · `us-east-2` · `us-west-1` · `us-west-2` · `eu-west-1` · `eu-west-2` · `eu-central-1` · `ap-southeast-1` · `ap-southeast-2` · `ap-northeast-1` · `sa-east-1` · `ca-central-1` · `af-south-1` |

Orchestrator options per compute environment:

| Compute Env        | Orchestrators                                              |
|--------------------|------------------------------------------------------------|
| Bare Metal         | `Docker Compose` · `K3s` · `Nomad` · `None`               |
| VM                 | `Docker Compose` · `K3s` · `Nomad` · `None`               |
| Containers (Docker)| `Docker Compose` · `K3s` · `K8s (managed)` · `Nomad` · `ECS` · `None` |
| Kubernetes         | `K3s` · `K8s (managed)`                                    |
| Serverless (FaaS)  | `Cloud Run` · `None`                                       |
| PaaS               | `Docker Compose` · `K3s` · `K8s (managed)` · `Nomad` · `ECS` · `Cloud Run` · `None` |

---

### 5.2 Networking Sub Tab

> Many options below are narrowed based on the **Cloud Provider** configured in §5.1 Environments.

| Field           | Options / Input                                                                                 |
|-----------------|-------------------------------------------------------------------------------------------------|
| DNS Provider    | Provider-aware: `Cloudflare` · `Route53` (AWS) · `Cloud DNS` (GCP) · `Azure DNS` (Azure) · `Other` |
| TLS/SSL         | `Let's Encrypt` · `Cloudflare` · `ACM` · `Manual` · `None (dev)`                               |
| Reverse Proxy   | Context-aware — see table below                                                                 |
| CDN             | Provider-aware: `Cloudflare` · `CloudFront` (AWS) · `Cloud CDN` (GCP) · `Azure CDN` · `Fastly` · `Vercel Edge` · `None` |
| Primary Domain  | Free text (e.g., `myapp.com`)                                                                   |
| Domain Strategy | `Subdomain per service` · `Path-based routing` · `Single domain` · `Custom`                     |
| CORS Enforced   | `Reverse proxy (Nginx/Caddy)` · `Application-level` · `CDN/WAF` · `Both`                       |
| CORS Strategy   | `Permissive` · `Strict allowlist` · `Same-origin`                                               |
| CORS Origins    | Free text                                                                                       |
| SSL Cert Mgmt   | Provider-aware: `Auto-renew (certbot/ACME)` · `ACM` (AWS) · `GCP-managed` · `Azure-managed` · `Cloudflare proxy` · `Manual` |

Reverse proxy options are narrowed by orchestrator/environment:

| Context                     | Reverse Proxy Options                                  |
|-----------------------------|--------------------------------------------------------|
| Kubernetes/K3s              | `NGINX Ingress` · `Traefik` · `Envoy`                  |
| Docker Compose              | `Nginx` · `Caddy` · `Traefik`                          |
| Cloud Run / Serverless      | `Cloudflare Tunnel` · `Cloud LB`                       |
| *(default / unset)*         | `Nginx` · `Caddy` · `Traefik` · `Cloudflare Tunnel` · `Cloud LB` |

---

### 5.3 Observability Sub Tab

> Alerting and Tracing options are narrowed based on the selected **Metrics** backend.

| Field          | Options / Input                                                                                          |
|----------------|----------------------------------------------------------------------------------------------------------|
| Logging        | Provider-aware: `Loki + Grafana` · `ELK Stack` · `CloudWatch` (AWS) · `Cloud Logging` (GCP) · `Azure Monitor` · `Datadog` · `Stdout/file` |
| Metrics        | Provider-aware: `Prometheus + Grafana` · `Datadog` · `CloudWatch` (AWS) · `Cloud Monitoring` (GCP) · `Azure Monitor` · `New Relic` · `None` |
| Tracing        | Filtered by Metrics — see table below                                                                    |
| Error Tracking | `Sentry` · `Datadog` · `Rollbar` · `Built-in` · `None`                                                   |
| Health Checks  | `false` · `true` — auto-generate `/health` and `/ready` endpoints per service unit                       |
| Alerting       | Filtered by Metrics — see table below                                                                    |
| Log Retention  | `7 days` · `30 days` · `90 days` · `1 year` · `Indefinite`                                               |

Tracing options per metrics backend:

| Metrics Backend      | Tracing Options                                               |
|----------------------|---------------------------------------------------------------|
| Prometheus + Grafana | `OpenTelemetry + Jaeger` · `OpenTelemetry + Tempo` · `None`  |
| Datadog              | `Datadog APM` · `OpenTelemetry + Jaeger` · `None`            |
| CloudWatch           | `AWS X-Ray` · `OpenTelemetry + Jaeger` · `None`              |
| Cloud Monitoring     | `Cloud Trace` · `OpenTelemetry + Jaeger` · `None`            |
| Azure Monitor        | `Azure App Insights` · `OpenTelemetry + Jaeger` · `None`     |
| New Relic            | `New Relic Distributed Tracing` · `OpenTelemetry + Jaeger` · `None` |
| *(default)*          | `OpenTelemetry + Jaeger` · `OpenTelemetry + Tempo` · `Datadog APM` · `AWS X-Ray` · `Cloud Trace` · `Azure App Insights` · `New Relic Distributed Tracing` · `None` |

Alerting options per metrics backend:

| Metrics Backend      | Alerting Options                                                    |
|----------------------|---------------------------------------------------------------------|
| Prometheus + Grafana | `Grafana Alerting` · `PagerDuty` · `OpsGenie` · `None`             |
| Datadog              | `Datadog Monitors` · `PagerDuty` · `OpsGenie` · `None`             |
| CloudWatch           | `CloudWatch Alarms` · `PagerDuty` · `OpsGenie` · `None`            |
| Cloud Monitoring     | `Cloud Monitoring Alerting` · `PagerDuty` · `OpsGenie` · `None`    |
| Azure Monitor        | `Azure Monitor Alerts` · `PagerDuty` · `OpsGenie` · `None`         |
| New Relic            | `New Relic Alerts` · `PagerDuty` · `OpsGenie` · `None`             |
| *(default)*          | All of the above                                                    |

---

### 5.4 CI/CD Sub Tab

> Container registry, IaC tool, and secrets management options are narrowed based on the **Cloud Provider** configured in §5.1 Environments. Container runtime is narrowed by backend language.

| Field              | Options / Input                                                                                    |
|--------------------|-----------------------------------------------------------------------------------------------------|
| CI/CD Platform     | `GitHub Actions` · `GitLab CI` · `Jenkins` · `CircleCI` · `ArgoCD` · `Tekton`                      |
| Container Registry | Provider-aware — see table below                                                                    |
| Deploy Strategy    | Filtered by Orchestrator — see table below                                                          |
| IaC Tool           | Provider-aware — see table below                                                                    |
| Secrets Mgmt       | Provider-aware — see table below                                                                    |
| Container Runtime  | Language-filtered — see table below                                                                 |
| Backup/DR          | `Cross-region replication` · `Daily snapshots` · `Managed provider DR` · `None`                    |

Container registry options per cloud provider:

| Cloud Provider | Registry Options                                |
|----------------|-------------------------------------------------|
| AWS            | `ECR` · `GHCR` · `Docker Hub`                  |
| GCP            | `GCR` · `Artifact Registry` · `GHCR`           |
| Azure          | `ACR` · `GHCR`                                  |
| Cloudflare / Hetzner / Self-hosted | `GHCR` · `Docker Hub` · `Self-hosted` |
| *(default)*    | `Docker Hub` · `GHCR` · `ECR` · `GCR` · `Artifact Registry` · `ACR` · `Self-hosted` |

Deploy strategy options per orchestrator:

| Orchestrator         | Deploy Strategies                              |
|----------------------|------------------------------------------------|
| Docker Compose / None| `Recreate`                                     |
| K3s / K8s (managed)  | `Rolling` · `Blue-green` · `Canary` · `Recreate` |
| ECS                  | `Rolling` · `Blue-green` · `Canary`            |
| Cloud Run            | `Rolling` · `Canary`                           |
| Nomad                | `Rolling` · `Blue-green` · `Canary`            |
| *(default)*          | `Rolling` · `Blue-green` · `Canary` · `Recreate` |

IaC tool options per cloud provider:

| Cloud Provider | IaC Tools                                              |
|----------------|--------------------------------------------------------|
| AWS            | `Terraform` · `Pulumi` · `CloudFormation` · `CDK` · `None` |
| GCP            | `Terraform` · `Pulumi` · `None`                        |
| Azure          | `Terraform` · `Pulumi` · `Bicep` · `ARM Templates` · `None` |
| Cloudflare     | `Terraform` · `Pulumi` · `Wrangler` · `None`           |
| Hetzner / Self-hosted | `Terraform` · `Pulumi` · `Ansible` · `None`      |
| *(default)*    | `Terraform` · `Pulumi` · `CloudFormation` · `CDK` · `Bicep` · `ARM Templates` · `Wrangler` · `Ansible` · `None` |

Secrets management options per cloud provider:

| Cloud Provider | Secrets Options                                                   |
|----------------|-------------------------------------------------------------------|
| AWS            | `AWS Secrets Manager` · `HashiCorp Vault` · `GitHub Secrets`     |
| GCP            | `GCP Secret Manager` · `HashiCorp Vault` · `GitHub Secrets`      |
| Azure          | `Azure Key Vault` · `HashiCorp Vault` · `GitHub Secrets`         |
| Others         | `HashiCorp Vault` · `GitHub Secrets`                             |
| *(default)*    | `GitHub Secrets` · `HashiCorp Vault` · `AWS Secrets Manager` · `GCP Secret Manager` · `Azure Key Vault` · `None` |

Container runtime options per backend language:

| Language        | Container Runtimes                                          |
|-----------------|-------------------------------------------------------------|
| Go              | `scratch` · `distroless` · `alpine`                         |
| TypeScript/Node | `node:alpine` · `node:slim` · `distroless/nodejs`           |
| Python          | `python:slim` · `python:alpine` · `distroless/python3`      |
| Java            | `eclipse-temurin:alpine` · `distroless/java` · `amazoncorretto` |
| Kotlin          | `eclipse-temurin:alpine` · `distroless/java` · `amazoncorretto` |
| C#/.NET         | `mcr.microsoft.com/dotnet/aspnet` · `alpine`                |
| Rust            | `scratch` · `distroless` · `alpine`                         |
| Ruby            | `ruby:slim` · `ruby:alpine`                                 |
| PHP             | `php:fpm-alpine` · `php:cli-alpine`                         |
| Elixir          | `elixir:alpine` · `elixir:slim`                             |

---

## 6 · Cross-Cutting Concerns Tab

### 6.1 Testing Strategy Sub Tab

> Options are dynamically filtered by backend languages, backend architecture pattern, communication protocols, and frontend tech.

| Field             | Options / Input                                                                  |
|-------------------|----------------------------------------------------------------------------------|
| Unit Testing      | Dynamically filtered by backend language — see table below                       |
| Integration Tests | Filtered by arch pattern — see table below                                       |
| E2E Testing       | Dynamically filtered by frontend platform/framework — see table below            |
| FE Testing        | Filtered by frontend language — see table below                                  |
| API Testing       | Filtered by communication protocols — see table below                            |
| Load Testing      | `k6` · `Artillery` · `JMeter` · `None` *(+ `Locust` when Python is a backend language)* |
| Contract Testing  | Filtered by arch pattern — see table below                                       |

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

Integration test options per architecture pattern:

| Architecture Pattern            | Integration Tools                                           |
|---------------------------------|-------------------------------------------------------------|
| Microservices / Event-Driven / Hybrid | `Testcontainers` · `Docker Compose` · `None`          |
| Monolith / Modular Monolith     | `In-memory fakes` · `Docker Compose` · `Testcontainers` · `None` |

E2E testing options per frontend platform/framework:

| Platform / Framework         | E2E Tools                                       |
|------------------------------|-------------------------------------------------|
| Web (any framework)          | `Playwright` · `Cypress` · `Selenium` · `None`  |
| Flutter / Dart               | `Flutter Driver` · `Integration Test` · `None`  |
| Jetpack Compose / Android    | `Espresso` · `UI Automator` · `None`            |
| SwiftUI / UIKit / iOS        | `XCUITest` · `EarlGrey` · `None`                |
| *(no frontend configured)*   | `None`                                          |

Frontend testing options per frontend language:

| Language        | FE Testing Tools                                                      |
|-----------------|-----------------------------------------------------------------------|
| TypeScript/JS   | `Vitest` · `Jest` · `Testing Library` · `Storybook` · `None`         |
| All others      | `None`                                                                |

API testing options per communication protocols:

| Protocols in Use    | API Testing Tools                                       |
|---------------------|---------------------------------------------------------|
| REST only           | `Bruno` · `Hurl` · `Postman/Newman` · `REST Client` · `None` |
| GraphQL only        | `Bruno` · `Postman/Newman` · `GraphQL Playground` · `None` |
| gRPC only           | `grpcurl` · `Postman/Newman` · `BloomRPC` · `None`      |
| Multiple protocols  | `Bruno` · `Postman/Newman` · `None`                    |
| *(no protocols)*    | `Bruno` · `Hurl` · `Postman/Newman` · `REST Client` · `None` |

Contract testing options per architecture pattern:

| Architecture Pattern     | Contract Tools                                |
|--------------------------|-----------------------------------------------|
| Microservices / Hybrid   | `Pact` · `Schemathesis` · `Dredd` · `None`    |
| Event-Driven             | `Pact` · `AsyncAPI validator` · `None`        |
| Monolith / Modular       | `None` · `Schemathesis`                       |

---

### 6.2 Documentation Sub Tab

> Documentation format is now **per protocol** — each active endpoint protocol gets its own format selection.

| Field           | Options / Input                                                        |
|-----------------|------------------------------------------------------------------------|
| REST Docs       | `OpenAPI/Swagger` · `None`                                             |
| GraphQL Docs    | `GraphQL Playground` · `GraphQL SDL` · `None`                          |
| gRPC Docs       | `gRPC reflection` · `Protobuf docs (buf.build)` · `None`               |
| WebSocket Docs  | `AsyncAPI` · `None`                                                    |
| Event Docs      | `AsyncAPI` · `CloudEvents spec` · `None`                               |
| Auto-generation | `false` · `true` — generate specs from code annotations                |
| Changelog       | `Conventional Commits` · `Manual` · `None`                             |

---

### 6.3 Standards Sub Tab

| Field              | Options / Input                                                                  |
|--------------------|----------------------------------------------------------------------------------|
| Dependency Updates | `Dependabot` · `Renovate` · `Manual` · `None`                                   |
| Feature Flags      | `LaunchDarkly` · `Unleash` · `Flagsmith` · `Custom (env vars)` · `None`          |
| Backend Linter     | Dynamically filtered by backend language — see table below                       |
| Frontend Linter    | Dynamically filtered by frontend language — see table below                      |

Backend linter options per language:

| Language        | Linters                                                   |
|-----------------|-----------------------------------------------------------|
| Go              | `golangci-lint` · `staticcheck` · `go vet` · `None`       |
| TypeScript/Node | `ESLint` · `Biome` · `TSLint (legacy)` · `None`           |
| Python          | `Ruff` · `Flake8` · `Pylint` · `mypy` · `None`            |
| Java            | `Checkstyle` · `SpotBugs` · `PMD` · `SonarLint` · `None`  |
| Kotlin          | `ktlint` · `detekt` · `SonarLint` · `None`                |
| C#/.NET         | `Roslyn Analyzers` · `StyleCop` · `SonarLint` · `None`    |
| Rust            | `Clippy` · `cargo-audit` · `None`                          |
| Ruby            | `RuboCop` · `StandardRB` · `None`                          |
| PHP             | `PHP-CS-Fixer` · `PHPStan` · `Psalm` · `None`             |
| Elixir          | `Credo` · `Dialyxir` · `None`                             |
| Other           | `Custom` · `None`                                          |
| *(no language)* | `golangci-lint` · `ESLint` · `Ruff` · `Checkstyle` · `Clippy` · `None` |

Frontend linter options per frontend language:

| Language           | Linters                                                              |
|--------------------|----------------------------------------------------------------------|
| TypeScript / JavaScript | `ESLint + Prettier` · `Biome` · `oxlint` · `Stylelint` · `Custom` · `None` |
| Dart / Kotlin / Swift | `Custom` · `None`                                                 |

> **Uptime SLO** and **Latency P99** are serialized in the manifest (`CrossCutPillar`) but are not currently editable from this tab — configure them directly in `manifest.json` if needed.

---

## 7 · Realize Tab

> Configure code generation options before running the realization engine. The tab is split into two groups: **App Settings** (output configuration) and **Provider** (model tier assignment).

### 7.1 App Settings

| Field           | Options / Input                                                                                                   |
|-----------------|-------------------------------------------------------------------------------------------------------------------|
| App Name        | Free text (e.g., `my-app`) — used as the output project name                                                      |
| Output Dir      | Free text path (default: `.`)                                                                                     |
| Concurrency     | `1` · `2` · `4` · `8` — max parallel tasks (default: `4`)                                                        |
| Verify          | `true` · `false` — run language verifier after each generated file (default: `true`)                              |
| Dry Run         | `false` · `true` — print task plan without invoking agents (default: `false`)                                     |

### 7.2 Provider & Tier Assignment

> The provider selector is populated from providers configured via the **Provider Menu** (`Shift+M`). Tier model selectors update dynamically when the provider changes.

| Field           | Options / Input                                                                                                   |
|-----------------|-------------------------------------------------------------------------------------------------------------------|
| Provider        | Select from configured providers (populated from Provider Menu — see §7.3)                                        |
| Tier Fast       | Select model for low-complexity tasks — options depend on selected provider (see tier table below)                 |
| Tier Medium     | Select model for medium-complexity tasks — options depend on selected provider                                     |
| Tier Slow       | Select model for high-complexity / escalation tasks — options depend on selected provider                          |

Tier model options per provider:

| Provider | Fast (Tier Fast)                                  | Medium (Tier Medium)                              | Slow (Tier Slow)                    |
|----------|---------------------------------------------------|---------------------------------------------------|-------------------------------------|
| Claude   | `claude-haiku-4-5-20251001`                       | `claude-sonnet-4-6`                               | `claude-opus-4-6`                   |
| ChatGPT  | `gpt-4o-mini` · `o3-mini`                         | `gpt-4o` · `gpt-4o-2024-11-20`                    | `o1` · `o1-preview`                 |
| Gemini   | `gemini-2.0-flash` · `gemini-1.5-flash`           | `gemini-2.0-pro-exp` · `gemini-1.5-pro`           | `gemini-ultra`                      |
| Mistral  | `open-mistral-nemo`                               | `mistral-small-2409` · `mistral-small-2402`        | `mistral-large-2411` · `mistral-large-2407` |
| Llama    | `llama-3.2-8b-preview` · `llama-3.1-8b-instant`   | `llama-3.3-70b-versatile` · `llama-3.1-70b-versatile` | `llama-3.1-405b-reasoning`     |

> When no provider is selected, the `—` (unset) sentinel is shown for all tier fields. The orchestrator falls back to Claude via the `ANTHROPIC_API_KEY` environment variable.

> **Legacy fields:** `model` and `section_models` still exist in the manifest JSON for backward compatibility but are **not editable** from the Realize tab UI.

### 7.3 Provider Menu *(Shift+M — configure LLM providers)*

Each provider can be independently configured with its own auth method and credential. Configured providers become available in the Realize tab's provider selector.

| Provider | Auth Methods          | Tiers                        |
|----------|-----------------------|------------------------------|
| Claude   | `API Key`             | `Haiku` · `Sonnet` · `Opus`  |
| ChatGPT  | `API Key`             | `Mini` · `4o` · `o1`         |
| Gemini   | `API Key` · `OAuth`   | `Flash` · `Pro` · `Ultra`    |
| Mistral  | `API Key`             | `Nemo` · `Small` · `Large`   |
| Llama    | `API Key`             | `8B` · `70B` · `405B`        |
| Custom   | `API Key`             | `Custom`                     |

> Credentials are persisted to the OS keyring / config file and loaded on startup. The Gemini provider supports OAuth 2.0 PKCE flow in addition to API key authentication.

---

## Dependency & Reference Graph

The tabs form a directed dependency graph — this is the intended order of definition:

```
Description (free-text project overview)
    ↓
Infrastructure → Environments (named envs referenced by all other pillars)
    ↓
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains; Stack Configs; defines Auth Roles)
    ↓
Contracts (DTOs reference Domains; Endpoints reference Service Units + Auth Roles)
    ↓
Frontend (Pages reference Endpoints + DTOs + Auth Roles from Backend)
    ↓
Infrastructure → Networking / Observability / CI/CD (references all deployable units)
    ↓
Cross-Cutting (Testing filtered by Backend langs + Frontend tech; Docs per-protocol)
    ↓
Realize (Code generation — orchestrates multi-provider generation for all pillars)
```

> **However**, the UI allows **non-linear editing** — users can start anywhere and link entities later. Empty references show as "unlinked" placeholders that can be resolved.
