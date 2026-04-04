package skills

import "github.com/vibe-menu/internal/realize/dag"

// aliasMap normalizes manifest technology strings to skill file base names.
// Add new technology aliases here without touching loader.go.
var aliasMap = map[string]string{
	// Go frameworks
	"Go":    "golang-patterns",
	"Fiber": "go-fiber",
	"Gin":   "go-gin",
	"Echo":  "go-echo-chi",
	"Chi":   "go-echo-chi",
	// TypeScript/Node frameworks
	"TypeScript": "coding-standards",
	"JavaScript": "coding-standards",
	"Express":    "node-express",
	"Fastify":    "node-fastify",
	"NestJS":     "node-nestjs",
	"Hono":       "node-hono-elysia",
	// Python frameworks
	"Python":  "python-patterns",
	"FastAPI": "python-fastapi",
	"Django":  "python-django",
	"Flask":   "python-flask-litestar",
	// Java / Spring Boot
	"Java":        "java-coding-standards",
	"Spring Boot": "java-spring-boot",
	"JPA":         "jpa-patterns",
	// Kotlin / Android / KMP
	"Kotlin":               "kotlin-patterns",
	"Ktor":                 "kotlin-ktor",
	"Android":              "android-clean-architecture",
	"Kotlin Multiplatform": "compose-multiplatform",
	// Swift / iOS
	"Swift":   "swiftui",
	"SwiftUI": "swiftui",
	"iOS":     "swiftui",
	// Native
	"Perl": "perl-patterns",
	"C++":  "cpp-coding-standards",
	"C":    "cpp-coding-standards",
	// Rust
	"Rust":      "rust-axum",
	"Axum":      "rust-axum",
	"Actix-web": "rust-actix-web",
	"Rocket":    "rust-rocket-warp",
	"Warp":      "rust-rocket-warp",
	// PHP frameworks
	"PHP":     "php-laravel-symfony",
	"Laravel": "php-laravel-symfony",
	"Symfony": "php-laravel-symfony",
	"Slim":    "php-slim",
	"Laminas": "php-laminas",
	// Ruby frameworks
	"Ruby":    "ruby-rails-sinatra",
	"Rails":   "ruby-rails-sinatra",
	"Sinatra": "ruby-rails-sinatra",
	"Hanami":  "ruby-hanami",
	"Roda":    "ruby-roda",
	// Elixir frameworks
	"Elixir":  "elixir-phoenix",
	"Phoenix": "elixir-phoenix",
	"Plug":    "elixir-plug-bandit",
	"Bandit":  "elixir-plug-bandit",
	// Desktop
	"Tauri":    "desktop-tauri-electron",
	"Electron": "desktop-tauri-electron",
	// Frontend frameworks
	"React":        "frontend-patterns",
	"Next.js":      "react-nextjs",
	"Vue":          "frontend-patterns",
	"Nuxt.js":      "web-vue-nuxt",
	"Svelte":       "frontend-patterns",
	"SvelteKit":    "web-svelte-sveltekit",
	"Angular":      "frontend-patterns",
	"Flutter":      "mobile-flutter",
	"React Native": "mobile-react-native-expo",
	// Databases
	"PostgreSQL": "postgres-patterns",
	"Postgres":   "postgres-patterns",
	"MySQL":      "db-mysql-mariadb",
	"MongoDB":    "db-mongodb-couchdb",
	"Redis":      "db-redis-memcached",
	"DynamoDB":   "db-dynamodb",
	"SQLite":     "db-sqlite",
	// Message brokers
	"Kafka":       "broker-kafka",
	"RabbitMQ":    "broker-rabbitmq",
	"NATS":        "broker-nats",
	"AWS SQS/SNS": "broker-cloud",
	// Job queues
	"BullMQ":        "jobs-bullmq",
	"Bull":          "jobs-bullmq",
	"Temporal":      "jobs-temporal",
	"Sidekiq":       "jobs-sidekiq-celery",
	"Celery":        "jobs-sidekiq-celery",
	"Dramatiq":      "jobs-sidekiq-celery",
	"Faktory":       "jobs-faktory-asynq-river",
	"Asynq":         "jobs-faktory-asynq-river",
	"River":         "jobs-faktory-asynq-river",
	"Hangfire":      "jobs-hangfire",
	"Laravel Queues": "jobs-laravel-queues",
	"Oban":          "jobs-oban",
	// Styling
	"Tailwind CSS": "tailwind",
	"Tailwind":     "tailwind",
	// Infrastructure
	"Docker":          "docker-patterns",
	"Terraform":       "iac-terraform-pulumi",
	"Terraform (AWS)": "iac-terraform-pulumi",
	"Pulumi":          "iac-terraform-pulumi",
	"GitHub Actions":  "github-actions",
	"GitLab CI":       "cicd-pipelines",
	// Auth / Identity providers
	"Auth0":   "idp-integrations",
	"Clerk":   "idp-integrations",
	"Cognito": "idp-integrations",
	// Go database drivers — map to targeted skill for pgx interface + version pinning
	"pgx":     "go-pgx-repository",
	"pgxv5":   "go-pgx-repository",
	"pgxmock": "go-pgx-repository",

	// Auth strategy aliases (AuthConfig.Strategy values)
	"JWT":            "auth-jwt-stateless",
	"Session-based":  "auth-session-based",
	"SAML":           "auth-oauth2-oidc",
	"OIDC":           "auth-oauth2-oidc",
	"API Key":        "auth-apikey",
	"mTLS":           "auth-mtls",

	// File storage provider aliases (FileStorages[i].Technology values)
	"S3":         "storage-s3-gcs",
	"GCS":        "storage-s3-gcs",
	"MinIO":      "storage-minio",
	"Azure Blob": "storage-azure-blob",
	"R2":         "storage-r2",

	// Cache DB aliases (CachingConfig.CacheDB values not yet aliased)
	// Note: "Redis" is already aliased above
	"Valkey":    "db-valkey",
	"Memcached": "db-redis-memcached",

	// Testing framework aliases (TestingConfig field values)
	"Jest":                   "test-unit",
	"Vitest":                 "test-unit",
	"pytest":                 "test-unit",
	"JUnit":                  "test-unit",
	"xUnit":                  "test-unit",
	"Go testing":             "test-unit",
	"RSpec":                  "test-unit",
	"Playwright":             "test-playwright-cypress",
	"Cypress":                "test-playwright-cypress",
	"Selenium":               "test-selenium",
	"Nightwatch":             "test-playwright-cypress",
	"k6":                     "test-load",
	"Locust":                 "test-load",
	"Apache JMeter":          "test-load",
	"Pact":                   "test-contract",
	"Spring Cloud Contract":  "test-contract",
	"REST Assured":           "test-api",
	"Postman":                "test-api",
	"Supertest":              "test-api",
	"Testcontainers":         "test-integration-containers",

	// CI platform aliases (CIPlatform values not yet aliased)
	// Note: "GitHub Actions" and "GitLab CI" are already aliased above
	"CircleCI": "cicd-pipelines",
	"Jenkins":  "cicd-pipelines",
	"Tekton":   "cicd-tekton",

	// Frontend feature aliases (RealtimeStrategy, BundleOptimization, ErrorBoundary)
	// Note: "WebSocket" is aliased below under Protocol aliases → protocol-websockets
	"Server-Sent Events": "web-realtime",
	"Pusher":             "web-realtime",
	"Ably":               "web-realtime",
	"Long Polling":       "web-realtime",
	"Code Splitting":     "frontend-bundle-optimization",
	"Error Boundaries":   "frontend-error-boundaries",
	"PWA":                "web-pwa",
	"next/image":         "web-image-optimization",

	// Docs format aliases (DocsConfig.Changelog values)
	"Keep a Changelog":     "contracts-changelog",
	"Conventional Commits": "contracts-changelog",

	// Protocol aliases (Endpoint/DTO protocol values)
	"GraphQL":   "protocol-graphql",
	"gRPC":      "protocol-grpc",
	"WebSocket": "protocol-websockets",
}

// universalSkillsForKind lists skill keys always injected for a task kind,
// regardless of which specific technologies appear in the manifest payload.
// Add new task kinds or universal skills here without touching loader.go.
var universalSkillsForKind = map[dag.TaskKind][]string{
	// Plan task: architect phase needs pgx interface guidance.
	dag.TaskKindServicePlan: {
		"backend-patterns", "go-pgx-repository",
	},
	// Service layers: each has a focused skill set matching its responsibility.
	dag.TaskKindServiceRepository: {
		"backend-patterns", "coding-standards", "go-pgx-repository",
		"go-caching-impl", // Redis/Valkey caching layer
	},
	dag.TaskKindServiceLogic: {
		"backend-patterns", "coding-standards",
		"go-background-workers", // goroutine worker pools
	},
	dag.TaskKindServiceHandler: {
		"backend-patterns", "security-review", "api-design", "coding-standards",
		"pagination-impl",      // every list endpoint needs pagination
		"api-versioning-impl",  // versioning strategy
	},
	dag.TaskKindServiceBootstrap: {
		"backend-patterns", "coding-standards",
		"file-storage-patterns", // multipart upload / presigned URLs
	},
	// Auth: always security-first
	dag.TaskKindAuth: {
		"security-review", "coding-standards", "security-scan",
	},
	// Data schemas: migrations guide
	dag.TaskKindDataSchemas: {
		"database-migrations",
		"multi-tenancy", // RLS + tenant context
	},
	// Migrations: migrations guide + postgres patterns (covers SQL DDL for any SQL DB)
	dag.TaskKindDataMigrations: {
		"database-migrations", "postgres-patterns",
		"multi-tenancy", // RLS + tenant context
	},
	// Messaging: backend patterns for broker wiring
	dag.TaskKindMessaging: {
		"backend-patterns", "coding-standards",
	},
	// Gateway: security + backend patterns
	dag.TaskKindGateway: {
		"backend-patterns", "security-review", "api-design",
		"grpc-gateway", // gRPC-JSON transcoding
	},
	// Contracts: API design is the primary skill here
	dag.TaskKindContracts: {
		"api-design", "coding-standards",
	},
	// Frontend: frontend patterns + coding standards
	dag.TaskKindFrontend: {
		"frontend-patterns", "coding-standards",
		"frontend-bundle-optimization", // code splitting + tree-shaking
		"frontend-error-boundaries",    // React/Next.js error handling
		"frontend-realtime-client",     // WebSocket / SSE hooks
	},
	// Docker: docker patterns + deployment
	dag.TaskKindInfraDocker: {
		"docker-patterns", "deployment-patterns",
	},
	// Terraform/IaC: deployment patterns
	dag.TaskKindInfraTerraform: {
		"deployment-patterns",
	},
	// CI/CD: deployment + verification loop
	dag.TaskKindInfraCI: {
		"deployment-patterns", "verification-loop",
	},
	// Testing: full TDD suite + e2e + verification
	dag.TaskKindCrossCutTesting: {
		"tdd-workflow", "e2e-testing", "coding-standards", "verification-loop",
	},
	// Docs: API design standards
	dag.TaskKindCrossCutDocs: {
		"api-design", "coding-standards",
	},
}
