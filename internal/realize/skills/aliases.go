package skills

import "github.com/vibe-menu/internal/realize/dag"

// aliasMap normalizes manifest technology strings to skill file base names.
// Add new technology aliases here without touching loader.go.
var aliasMap = map[string]string{
	// Go frameworks
	"Go":    "golang-patterns",
	"Fiber": "go-fiber",
	"Gin":   "go-gin",
	"Echo":  "go-echo",
	"Chi":   "go-chi",
	// TypeScript/Node frameworks
	"TypeScript": "coding-standards",
	"JavaScript": "coding-standards",
	"Express":    "typescript-express",
	"Fastify":    "typescript-fastify",
	"NestJS":     "typescript-nestjs",
	"Hono":       "typescript-hono",
	// Python frameworks
	"Python":  "python-patterns",
	"FastAPI": "python-fastapi",
	"Django":  "python-django",
	"Flask":   "python-flask",
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
	"Axum":      "rust-axum",
	"Actix-web": "rust-actix",
	// Frontend frameworks
	"React":        "frontend-patterns",
	"Next.js":      "react-nextjs",
	"Vue":          "frontend-patterns",
	"Nuxt.js":      "vue-nuxt",
	"Svelte":       "frontend-patterns",
	"SvelteKit":    "svelte-kit",
	"Angular":      "frontend-patterns",
	"Flutter":      "flutter",
	"React Native": "react-native",
	// Databases
	"PostgreSQL": "postgres-patterns",
	"Postgres":   "postgres-patterns",
	"MySQL":      "mysql",
	"MongoDB":    "mongodb",
	"Redis":      "redis",
	"DynamoDB":   "dynamodb",
	"SQLite":     "sqlite",
	// Message brokers
	"Kafka":       "kafka",
	"RabbitMQ":    "rabbitmq",
	"NATS":        "nats",
	"AWS SQS/SNS": "aws-sqs-sns",
	// Styling
	"Tailwind CSS": "tailwind",
	"Tailwind":     "tailwind",
	// Infrastructure
	"Docker":          "docker-patterns",
	"Terraform":       "terraform",
	"Terraform (AWS)": "terraform-aws",
	"Pulumi":          "pulumi",
	"GitHub Actions":  "github-actions",
	"GitLab CI":       "gitlab-ci",
	// Auth
	"Auth0":   "auth0",
	"Clerk":   "clerk",
	"Cognito": "aws-cognito",
}

// universalSkillsForKind lists skill keys always injected for a task kind,
// regardless of which specific technologies appear in the manifest payload.
// Add new task kinds or universal skills here without touching loader.go.
var universalSkillsForKind = map[dag.TaskKind][]string{
	// Backend service: quality patterns + security
	dag.TaskKindService: {
		"backend-patterns", "security-review", "coding-standards",
	},
	// Auth: always security-first
	dag.TaskKindAuth: {
		"security-review", "coding-standards", "security-scan",
	},
	// Data schemas: migrations guide
	dag.TaskKindDataSchemas: {
		"database-migrations",
	},
	// Migrations: migrations guide + postgres patterns (covers SQL DDL for any SQL DB)
	dag.TaskKindDataMigrations: {
		"database-migrations", "postgres-patterns",
	},
	// Messaging: backend patterns for broker wiring
	dag.TaskKindMessaging: {
		"backend-patterns", "coding-standards",
	},
	// Gateway: security + backend patterns
	dag.TaskKindGateway: {
		"backend-patterns", "security-review", "api-design",
	},
	// Contracts: API design is the primary skill here
	dag.TaskKindContracts: {
		"api-design", "coding-standards",
	},
	// Frontend: frontend patterns + coding standards
	dag.TaskKindFrontend: {
		"frontend-patterns", "coding-standards",
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
