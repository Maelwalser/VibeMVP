# VibeMenu — Project Description & Engineering Standards

## 1. Project Overview

**VibeMenu** is an interactive Terminal User Interface (TUI) CLI tool for declaratively specifying a complete software system architecture. It implements a vim-inspired editor that lets developers and architects define comprehensive system manifests across 8 sections — a free-form description editor plus 7 structured pillars (backend, data, contracts, frontend, infrastructure, cross-cutting concerns, and code generation configuration).

The resulting manifest is serialized to `manifest.json` and intended for downstream consumption by code-generation agents or tooling via the `cmd/realize` pipeline.

**Key design principles:**
- Vim-modal editing (Normal / Insert / Command modes)
- Tokyo Night dark theme throughout
- Non-linear editing — users can fill any tab in any order
- Pillar-based dependency graph: Description → Data → Backend → Contracts → Frontend → Infrastructure → Cross-Cutting → Realize

---

## 2. Technology Stack

| Concern | Choice |
|---------|--------|
| Language | Go 1.26.1 |
| TUI framework | `github.com/charmbracelet/bubbletea` v1.3.10 |
| TUI components | `github.com/charmbracelet/bubbles` v1.0.0 (textarea, textinput) |
| Styling/layout | `github.com/charmbracelet/lipgloss` v1.1.0 |
| Claude SDK | `github.com/anthropics/anthropic-sdk-go` v1.28.0 |
| TUI entry point | `cmd/agent/main.go` |
| Realize entry point | `cmd/realize/main.go` |
| Manifest types | `internal/manifest/` (9 files, split by pillar) |
| UI components | `internal/ui/` (56 files, ~24,076 lines) |
| Code generation engine | `internal/realize/` (DAG, agent, skills, verifiers, orchestrator, memory) |
| Default realize model | `claude-sonnet-4-6` (tier-dependent; see Section 6.3) |

---

## 3. Project Structure

```
VibeMenu/
├── cmd/
│   ├── agent/
│   │   └── main.go              # TUI entry point — sets up save callback, runs Bubble Tea program (50 lines)
│   └── realize/
│       └── main.go              # Code-gen entry point — CLI flags, runs orchestrator (40 lines)
├── internal/
│   ├── manifest/
│   │   ├── manifest.go          # Root Manifest struct + Save() + RealizeOptions + ProviderAssignments (113 lines)
│   │   ├── manifest_enums.go    # All enum type declarations (186 lines)
│   │   ├── manifest_data.go     # DataPillar, DBSourceDef, DomainDef, caching types (174 lines)
│   │   ├── manifest_backend.go  # BackendPillar, ServiceDef, CommLink, AuthConfig, RoleDef, PermissionDef, WAFConfig, JobQueueDef, CronJobDef (179 lines)
│   │   ├── manifest_contracts.go # ContractsPillar, DTODef, EndpointDef, APIVersioning, ExternalAPIDef, ExternalAPIInteraction (131 lines)
│   │   ├── manifest_frontend.go  # FrontendPillar, FrontendTechConfig, FrontendTheme, PageDef, PageComponentDef, ComponentActionDef, NavigationConfig, I18nConfig, A11ySEOConfig, AssetDef (159 lines)
│   │   ├── manifest_infra.go     # InfraPillar, NetworkingConfig, CICDConfig, ObservabilityConfig, ServerEnvironmentDef (56 lines)
│   │   ├── manifest_crosscut.go  # CrossCutPillar, TestingConfig, DocsConfig (37 lines)
│   │   └── recent.go            # Recent manifest tracking (69 lines)
│   ├── ui/
│   │   ├── model.go             # Root TUI model, vim modes, Update + command dispatch (537 lines)
│   │   ├── model_sections.go    # Section registry: editor getters + update closures (one entry per pillar) (150 lines)
│   │   ├── model_view.go        # Root View() and all render helpers (319 lines)
│   │   ├── editor.go            # Editor interface (Mode, HintLine, View) (22 lines)
│   │   ├── editor_state.go      # DropdownState shared struct (9 lines)
│   │   ├── nav.go               # NavigateTab(), VimNav struct — shared navigation helpers (148 lines)
│   │   ├── styles.go            # Tokyo Night palette, all lipgloss styles (196 lines)
│   │   ├── sections.go          # Section/field definitions, FieldKind enum (374 lines)
│   │   ├── render_helpers.go    # Shared rendering utilities (fillTildes, renderFormFields, …) (783 lines)
│   │   ├── field_options.go     # Shared field option slices (OptionsOnOff, OptionsOffOn) (9 lines)
│   │   ├── animation.go         # Animation utilities (32 lines)
│   │   ├── app.go               # App initialization and setup (105 lines)
│   │   ├── lang_versions.go     # Language version matrices for tech filtering (264 lines)
│   │   ├── description_editor.go # Pillar 0: free-text project description textarea (130 lines)
│   │   ├── backend_editor.go    # Backend: struct, init, ToManifest, Update dispatcher (1,068 lines — approaching split threshold)
│   │   ├── backend_fields.go    # Backend: default field constructors and service/comm/auth form helpers (889 lines)
│   │   ├── backend_services.go  # Backend: service list/form + comm list/form + messaging update handlers (692 lines)
│   │   ├── backend_update.go    # Backend: dropdown, insert update handlers (602 lines)
│   │   ├── backend_view.go      # Backend: HintLine, View, all sub-tab render functions (634 lines)
│   │   ├── backend_auth_security.go # Backend: auth + security update handlers and view (770 lines)
│   │   ├── backend_config.go    # Backend: stack config list/form handlers (222 lines)
│   │   ├── backend_jobs.go      # Backend: jobs list/form handlers (218 lines)
│   │   ├── data_tab_editor.go   # Data: struct, init, ToManifest, Update dispatcher, View (685 lines)
│   │   ├── data_tab_fields.go   # Data: database/domain/caching/fs field constructors + helpers (860 lines)
│   │   ├── data_domains.go      # Data: domain list/form + attr/rel update handlers + viewDomains (549 lines)
│   │   ├── data_domain_fields.go # Data: domain form field constructors (223 lines)
│   │   ├── data_caching_storage.go # Data: caching/governance/file-storage update handlers + views (463 lines)
│   │   ├── data_editor.go       # Entity/column schema editor: struct, init, Mode, HintLine, Update (143 lines)
│   │   ├── data_editor_fields.go # Entity editor: column + entity settings form helpers (275 lines)
│   │   ├── data_editor_update.go # Entity editor: all update handlers (449 lines)
│   │   ├── data_editor_view.go  # Entity editor: all view functions (414 lines)
│   │   ├── db_editor.go         # Database source editor (498 lines)
│   │   ├── db_editor_fields.go  # Database form field constructors (197 lines)
│   │   ├── contracts_editor.go  # Contracts: struct, init, ToManifest, Update dispatcher (634 lines)
│   │   ├── contracts_fields.go  # Contracts: DTO/endpoint/versioning/external field constructors (907 lines)
│   │   ├── contracts_dtos.go    # Contracts: DTO list/form + field drill-down update + viewDTOs (521 lines)
│   │   ├── contracts_endpoints.go # Contracts: endpoint/versioning/external update + views (723 lines)
│   │   ├── frontend_editor.go   # Frontend: struct, init, ToManifest, Update dispatcher (808 lines)
│   │   ├── frontend_fields.go   # Frontend: compatibility maps + default field constructors (1,020 lines)
│   │   ├── frontend_update.go   # Frontend: tech/theme update handlers + View (679 lines)
│   │   ├── frontend_pages_update.go # Frontend: page/navigation update handlers (216 lines)
│   │   ├── frontend_i18n_a11y.go # Frontend: i18n + a11y/SEO update handlers + viewPages (301 lines)
│   │   ├── frontend_action_fields.go # Frontend: component action field constructors (265 lines)
│   │   ├── frontend_assets.go   # Frontend asset management (241 lines)
│   │   ├── infra_editor.go      # Infra: struct, init, ToManifest, Update dispatcher, View (870 lines)
│   │   ├── infra_fields.go      # Infra: provider maps, deploy strategies, default field constructors (638 lines)
│   │   ├── crosscut_editor.go   # Crosscut: struct, init, ToManifest, Update dispatcher, View (532 lines)
│   │   ├── crosscut_fields.go   # Crosscut: testing/docs/standards field constructors (488 lines)
│   │   ├── realize_editor.go    # Realize: code generation configuration form (445 lines)
│   │   ├── provider_menu.go     # Provider modal: types, struct, init, state helpers (317 lines)
│   │   ├── provider_menu_oauth.go # Provider modal: OAuth 2.0 PKCE flow + credential step (306 lines)
│   │   ├── provider_menu_update.go # Provider modal: Update handler (225 lines)
│   │   ├── provider_menu_view.go # Provider modal: View + all render functions (428 lines)
│   │   ├── realization_screen.go # Code generation output screen (317 lines)
│   │   ├── welcome.go           # Welcome/initialization screen (265 lines)
│   │   ├── description_editor.go # Free-text project description area (130 lines)
│   │   └── open_file_modal.go   # File open dialog (stub)
│   └── realize/
│       ├── agent/
│       │   ├── agent.go         # Agent interface + ClaudeAgent implementation with streaming (161 lines)
│       │   ├── openai_agent.go  # OpenAI-compatible agent (ChatGPT, Mistral, Llama via Groq) (144 lines)
│       │   ├── gemini_agent.go  # Google Gemini agent implementation (135 lines)
│       │   ├── context.go       # Agent context struct (26 lines)
│       │   ├── prompt.go        # System/user prompt builders (265 lines)
│       │   └── roles.go         # taskRoleDescriptions map — detailed role instructions per TaskKind (159 lines)
│       ├── dag/
│       │   ├── dag.go           # DAG struct, topological sort, TaskKind enums (160 lines)
│       │   ├── builder.go       # Manifest → DAG task graph construction (547 lines)
│       │   └── payload.go       # Task payload types (51 lines)
│       ├── config/
│       │   └── defaults.go      # Tunable constants (DefaultModel, DefaultMaxTokens, MaxSkillBytes, …) (20 lines)
│       ├── orchestrator/
│       │   ├── orchestrator.go  # Config, entrypoint, task dispatch (336 lines)
│       │   ├── models.go        # Provider model registry for all 6 providers, resolveModelID() (184 lines)
│       │   ├── runner.go        # Per-task runner: agent call + verify + retry (389 lines)
│       │   └── tier.go          # defaultTierForKind map + escalateModel() (69 lines)
│       ├── memory/
│       │   ├── memory.go        # SharedMemory — stores completed task outputs for downstream agents (202 lines)
│       │   ├── sigscan.go       # Signature extraction from generated source files (189 lines)
│       │   └── memory_test.go   # Unit tests for SharedMemory (175 lines)
│       ├── state/
│       │   └── state.go         # State tracking during code generation (76 lines)
│       ├── output/
│       │   └── writer.go        # File output writer (74 lines)
│       ├── skills/
│       │   ├── registry.go      # In-memory skill registry (17 lines)
│       │   ├── aliases.go       # Technology alias map + universal skills per task kind (158 lines)
│       │   └── loader.go        # Load skill markdown files from disk (109 lines)
│       ├── deps/
│       │   ├── resolver_interface.go # LanguageResolver interface + ResolverRegistry (34 lines)
│       │   ├── resolver.go      # deps.Agent — runs package manager to lock deps (246 lines)
│       │   ├── modules.go       # Shared: ResolvedDeps types, LibraryAPIDocs, PromptContext, Save/Load (231 lines)
│       │   ├── go_modules.go    # Go-specific: WellKnownGoModules, GoModForService, ValidateGoMod (401 lines)
│       │   └── npm_modules.go   # npm-specific: WellKnownNpmPackages, resolveNpmVersion (94 lines)
│       └── verify/
│           ├── verifier.go      # Verifier interface + Registry + ForTask() (134 lines)
│           ├── go_verifier.go   # go build + go vet verifier (169 lines)
│           ├── ts_verifier.go   # tsc verifier (59 lines)
│           ├── python_verifier.go # python -m py_compile verifier (63 lines)
│           ├── tf_verifier.go   # terraform validate verifier (64 lines)
│           ├── null_verifier.go # No-op verifier for unknown languages (13 lines)
│           ├── fixer.go         # Deterministic fixes (gofmt, unused imports) before retry (99 lines)
│           └── deterministic_fixes.go # Advanced error pattern matching and fix application (303 lines)
├── system-declaration-menu.md   # Full specification: all options for every field
├── go.mod / go.sum
└── LICENSE
```

File size budget: **800 lines max** per file. Extract utilities if approaching this limit.

> Several files currently exceed or approach the limit (`backend_editor.go` 1,068 lines, `frontend_fields.go` 1,020 lines, `backend_fields.go` 889 lines, `contracts_fields.go` 907 lines, `render_helpers.go` 783 lines). When adding features to these files, extract helpers to dedicated files first.

---

## 4. Architecture

### 4.1 Vim Modal System

The root `Model` (`model.go`) owns three modes:

```go
type Mode int
const (
    ModeNormal   // Navigation: Tab/Shift-Tab between sections, j/k within
    ModeInsert   // Text input: i to enter, Esc to exit
    ModeCommand  // :w :q :wq :tabn :tabp :1-6 :help
)
```

### 4.2 Editor Interface + Polymorphic Dispatch

All sub-editors implement the `Editor` interface defined in `editor.go`:

```go
type Editor interface {
    Mode() Mode
    HintLine() string
    View(w, h int) string
}
```

The root `Model` uses `activeEditor() Editor` and `delegateUpdate()` — both dispatch through `sectionRegistry` in `model_sections.go` rather than switch statements. Adding a new pillar requires one `sectionEntry` registration in `buildSectionRegistry()`; no other files need changing.

Each sub-editor also implements:
- `ToManifest[X]Pillar()` — serializes editor state to manifest types

The **KindDataModel** sentinel field in `sections.go` signals full delegation to the sub-editor.

### 4.3 Shared Navigation Utilities (`nav.go`)

Two reusable helpers replace duplicated navigation code across all sub-editors:

**`NavigateTab(key, active, maxTabs int) int`** — handles `h`/`l`/`left`/`right` tab switching. Used in every editor that has sub-tabs.

**`VimNav` struct** — stateful count-prefix + vim motion handler:
```go
type VimNav struct { CountBuf string; GBuf bool }
// Handle returns (newIdx, consumed). consumed=false for enter/space/i/a (caller handles those).
func (v *VimNav) Handle(key string, idx, n int) (int, bool)
func (v *VimNav) Reset()
```
Handles: digit accumulation, `j`/`k` with count multiplier, `gg` (top), `G` (bottom). Used by `InfraEditor` and `CrossCutEditor`.

### 4.4 List+Form Pattern (used in most sub-editors)

```
SubView: List → user presses Enter → SubView: Form → Esc → SubView: List
```

Lists show items with `j/k` navigation. `a` adds, `d` deletes, `Enter`/`i` edits. Forms use unified `renderFormFields()` from `render_helpers.go`.

### 4.5 Manifest Builder Pattern

Each sub-editor implements `ToManifest[X]Pillar()` converting in-memory form state to the canonical manifest structs. `BuildManifest()` in `model.go` calls all seven to assemble the final `manifest.Manifest`.

### 4.6 Model Sub-Structs

The root `Model` struct groups related fields into sub-structs to reduce coupling:

```go
type cmdState    struct { buffer, status string; isErr bool }
type modalState  struct { open bool; menu ProviderMenu }
type realizeState struct { screen RealizationScreen; show, triggered bool }
```

### 4.7 Rendering Layout

All form fields use a consistent vim-style layout via `renderFormFields()`:
```
[LineNo] [Label          ] = [Value]
   3          14            3    (remaining width)
```

Tab bars use `renderSubTabBar()`. Bottom hints use `hintBar()`.

### 4.8 UI Components

- **DescriptionEditor** (`description_editor.go`): Pillar 0 — free-text textarea for project overview before filling structured pillars.
- **RealizeEditor** (`realize_editor.go`): Configuration form for code generation with provider selection, per-tier model assignment, concurrency, and verification settings.
- **ProviderMenu** (`provider_menu.go` + 3 sibling files): Interactive modal (Shift+M) for selecting and configuring LLM providers (Claude, ChatGPT, Gemini, Mistral, Llama, Custom) with tier selection and OAuth 2.0 PKCE flow.
- **RealizationScreen** (`realization_screen.go`): Display for code generation progress and output status.
- **WelcomeScreen** (`welcome.go`): Initial welcome/tutorial screen and manifest initialization.
- **FrontendAssets** (`frontend_assets.go`): Asset management utilities for frontend design assets.
- **Animation** (`animation.go`): Reusable animation primitives for TUI effects.
- **App** (`app.go`): High-level app initialization and lifecycle management.
- **LangVersions** (`lang_versions.go`): Language version matrices used for dynamic framework/version filtering across pillars.

---

## 5. The 8 Sections (Description + 7 Pillars)

### Section 0 — Description (`DescriptionEditor`)

Free-text project description textarea. Allows users to describe the project in natural language before filling in the structured pillars. Content is saved to `manifest.json` under the description field.

### Pillar 1 — Backend (`BackendEditor`)
Sub-tabs: **Env** · **Services** · **Stack Config** · **Communication** · **Messaging** · **API Gateway** · **Jobs** · **Security** · **Auth**

- Architecture pattern selector (Monolith / Modular Monolith / Microservices / Event-Driven / Hybrid) conditionally shows/hides sub-tabs
- Services list with per-service: name, responsibility, language, framework (dynamically filtered by language), pattern tag
- Stack Config: reusable language/framework combinations for multi-language services
- Communication links: from/to service, protocol, direction, trigger, sync/async, resilience patterns
- Messaging: broker config + repeatable event catalog
- API Gateway: technology, routing, features
- Jobs: background job queues (`JobQueueDef`) and cron jobs (`CronJobDef`) configuration
- Security: WAF configuration (`WAFConfig`), CORS settings, session management
- Auth: strategy, identity provider (with `RoleDef` list for authorization roles), permission definitions, authorization model, token storage, MFA
  - Supports role-based access control (RBAC) with role inheritance
  - Roles can be referenced in endpoint auth_required fields and frontend page access control

### Pillar 2 — Data (`DataTabEditor` + `DBEditor` + `DataEditor`)
Sub-tabs: **Databases** · **Domains** · **Caching** · **File Storage**

- Databases: alias, category, technology (filtered by category), hosting, HA mode — with type-conditional fields (SSL mode, eviction policy, replication factor, etc.)
- Domains: bounded contexts with repeatable attributes (name, type, constraints, default, sensitive, validation) and relationships (type, FK field, cascade)
- Entities (legacy model): similar to domains but in separate `data_editor.go`
- Caching layer config; File/object storage config

### Pillar 3 — Contracts (`ContractsEditor`)
Sub-tabs: **DTOs** · **Endpoints** · **API Versioning** · **External APIs**

- DTOs: name, category (Request/Response/Event Payload/Shared), source domain, protocol (REST/JSON, Protobuf, Avro, MessagePack, Thrift, FlatBuffers, Cap'n Proto), nested fields with protocol-specific types and validation
  - Protocol-specific fields: Protobuf (package, syntax, options), Avro (namespace, schema registry), Thrift (namespace, language), FlatBuffers/Cap'n Proto (namespace)
- Endpoints: service unit, name/path, protocol (REST/GraphQL/gRPC/WebSocket/Event), auth_required, auth_roles (multi-select from backend roles), request/response DTOs
  - Protocol-specific: HTTP method + pagination strategy (REST), operation type (GraphQL), stream type (gRPC), direction (WebSocket)
- API Versioning: strategy (URL path, header, query param, none), current version, deprecation policy
- External APIs: integration with third-party services with protocol-specific configurations (`ExternalAPIDef` + `ExternalAPIInteraction`)
  - Provider, protocol (REST/GraphQL/gRPC/WebSocket/Webhook/SOAP), auth mechanism (API Key, OAuth2, Bearer, Basic, mTLS, None), failure strategy
  - Protocol-conditional fields: REST (base URL, HTTP method, content type, rate limit, webhook endpoint), GraphQL (operation type), gRPC (stream type, TLS mode), WebSocket (subprotocol, message format), Webhook (HMAC header, retry policy), SOAP (version)
  - Request/response DTOs filtered by protocol (backwards compatible with untagged DTOs)

### Pillar 4 — Frontend (`FrontendEditor`)
Sub-tabs: **Tech** · **Theme** · **Pages** · **Navigation** · **i18n** · **A11y/SEO**

- Tech: language, platform, framework (filtered by language+platform), meta-framework, package manager, styling, component library, state management, data fetching, form handling, validation, PWA support, realtime strategy, image optimization, auth flow type, error boundary, bundle optimization, frontend testing, frontend linter
- Theme: dark mode strategy, border radius, spacing scale, elevation, motion, vibe, colors, description
- Pages: route, auth_required, layout, core actions, loading strategy, error handling, auth_roles (multi-select from backend roles for role-based page access), linked pages
  - Pages can define `PageComponentDef` entries with `ComponentActionDef` — 12+ action types (Fetch, Submit, Download, Upload, Delete, Refresh, Export, Navigate, Toast, State, Custom)
- Navigation: nav type (sidebar, top bar, etc.), breadcrumbs toggle, auth-aware navigation toggle
- Assets: frontend design assets (images, icons, fonts, videos, mockups, etc.) with usage classification (project or inspiration) via `AssetDef`
- i18n: internationalization settings (`I18nConfig`)
- A11y/SEO: accessibility and SEO configuration (`A11ySEOConfig`)

### Pillar 5 — Infrastructure (`InfraEditor`)
Sub-tabs: **Networking** · **CI/CD** · **Observability**

- Networking: DNS, TLS, reverse proxy, CDN
- CI/CD: platform, container registry, deploy strategy, IaC tool, secrets management
- Observability: logging, metrics, tracing, error tracking, health checks, alerting (`ObservabilityConfig`)
- Server environments: named environments with compute/cloud/orchestrator settings (`ServerEnvironmentDef`)

### Pillar 6 — Cross-Cutting (`CrosscutEditor`)
Sub-tabs: **Testing** · **Docs**

- Testing: testing framework selections dynamically filtered by backend languages and frontend tech choices
  - Unit: language-specific test framework (Jest, Vitest for JavaScript/TypeScript; pytest, Go testing, JUnit, xUnit for others)
  - Integration: integration test framework
  - E2E: end-to-end test tool (Playwright, Cypress, Nightwatch, Selenium, etc.)
  - API: API testing tool (REST, GraphQL, gRPC specific)
  - Load: load testing tool (k6, Locust, Apache JMeter, etc.)
  - Contract: contract testing tool (Pact, Spring Cloud Contract)
- Docs: API doc format (OpenAPI/Swagger, GraphQL schema doc, AsyncAPI, etc.), auto-generation toggle, changelog strategy

### Pillar 7 — Realize (`RealizeEditor`)
Configuration tab for downstream code generation pipeline:

- app_name: application name for generated code
- output_dir: destination directory for generated files
- concurrency: parallel task execution limit (1, 2, 4, 8)
- verify: enable/disable code verification after generation (default: true)
- dry_run: print task plan without executing agent calls
- provider: select from configured providers (populated from Provider Menu)
- tier_fast / tier_medium / tier_slow: model ID assignment per complexity tier (options depend on selected provider)
- Provider assignments (`ProviderAssignments`): which LLM provider to use per section

---

## 6. Realize Engine (Code Generation)

`cmd/realize` is the downstream consumer of `manifest.json`. It drives an agentic, multi-provider code-generation pipeline.

### 6.1 Pipeline Overview

```
manifest.json
    ↓
dag.Builder.Build()       → execution DAG (tasks with dependency edges)
    ↓
orchestrator.Run()        → parallel task dispatch (bounded by --parallel flag)
    ↓  (per task)
runner.Run()              → deterministic fixes → agent.Call() → verify.Check() → retry (escalating model tier)
    ↓
memory.SharedMemory       → stores completed outputs; downstream agents read upstream signatures
    ↓
output.Writer             → writes generated files under --output directory
```

### 6.2 Multi-Provider Agent System

Three agent implementations in `internal/realize/agent/`:

| File | Providers |
|------|-----------|
| `agent.go` (ClaudeAgent) | Claude Haiku, Sonnet, Opus |
| `openai_agent.go` (OpenAIAgent) | ChatGPT o3-mini/4o/o1, Mistral Nemo/Small/Large, Llama 8B/70B/405B via Groq |
| `gemini_agent.go` (GeminiAgent) | Gemini Flash, Pro, Ultra |

Provider and model selection is configured via `ProviderAssignments` in the manifest's realize section, and managed through the interactive **Provider Menu** (`Shift+M`).

`roles.go` contains `taskRoleDescriptions` — a per-`TaskKind` map of detailed role instructions injected into every system prompt, providing context-specific guidance for each code generation task.

### 6.3 Model Tiering (`orchestrator/tier.go`)

Tasks are assigned default model tiers by kind. On verification failure, `escalateModel()` automatically promotes to a higher tier for the retry:

| Tier | Default kinds | Models (Claude / OpenAI / Gemini) |
|------|--------------|-----------------------------------|
| Haiku/fast | contracts, docs, docker, CI | Haiku 4.5 / o3-mini / Flash |
| Sonnet/medium | services, auth, data, frontend, terraform, testing | Sonnet 4.6 / 4o / Pro |
| Opus/slow | escalation fallback | Opus 4.6 / o1 / Ultra |

### 6.4 Shared Memory (`realize/memory/`)

After each task completes, its output is stored in `SharedMemory`. Downstream agents receive a prompt context that includes exported signatures (function names, types, interfaces) extracted by `sigscan.go` from upstream files. This reduces duplication and enables consistent cross-service references without re-reading the full output.

### 6.5 DAG Task IDs

Tasks follow a naming convention derived from manifest entries:

| Pattern | Example |
|---------|---------|
| `data.<alias>` | `data.postgres` |
| `svc.<name>` | `svc.api-gateway` |
| `contracts` | `contracts` |
| `frontend` | `frontend` |
| `infra.<component>` | `infra.networking` |
| `crosscut.<component>` | `crosscut.testing` |

### 6.6 Skills System

Skills are markdown files in `.vibemenu/skills/` (configurable via `--skills`). Each file defines a named generation skill. The `skills.Loader` reads them at startup; the `skills.Registry` makes them available to the agent prompt builder. Technology aliases in `skills/aliases.go` map framework names to canonical skill file names.

### 6.7 Verifiers + Deterministic Fixes

Before each retry, `fixer.go` and `deterministic_fixes.go` apply zero-LLM deterministic fixes (gofmt formatting, unused import removal, common error patterns) to save token cost. After fixing, the language-appropriate verifier checks the output:

| Language | Verifier | Check |
|----------|----------|-------|
| Go | `go_verifier` | `go build` + `go vet` |
| TypeScript | `ts_verifier` | `tsc --noEmit` |
| Python | `python_verifier` | `python -m py_compile` |
| Terraform | `tf_verifier` | `terraform validate` |
| Other | `null_verifier` | always passes |

### 6.8 CLI Flags

```
--manifest  path to manifest.json      (default: manifest.json)
--output    output directory            (default: output)
--skills    skills directory            (default: .vibemenu/skills)
--retries   max retry attempts per task (default: 3)
--parallel  max concurrent tasks        (default: 1)
--dry-run   print task plan, no agents
--verbose   print token usage + thinking logs
```

---

## 7. Manifest Output

Saved to `manifest.json` on `:w` / `Ctrl+S`. Structure:

```json
{
  "created_at": "2026-...",
  "description": "...",
  "backend":    { "arch_pattern": "...", "services": [...], "stack_configs": [...], "auth": { "roles": [...], ... }, "waf": {...}, "job_queues": [...], "cron_jobs": [...], ... },
  "data":       { "databases": [...], "domains": [...], ... },
  "contracts":  { "dtos": [...], "endpoints": [...], "versioning": {...}, "external_apis": [...], ... },
  "frontend":   { "tech": {...}, "theme": {...}, "pages": [...], "navigation": {...}, "i18n": {...}, "a11y_seo": {...}, "assets": [...] },
  "infrastructure": { "networking": {...}, "cicd": {...}, "observability": {...}, "environments": [...] },
  "cross_cutting":  { "testing": {...}, "docs": {...} },
  "realize":    { "app_name": "...", "output_dir": "...", "concurrency": 4, "verify": true, "dry_run": false, "provider": "...", "tier_fast": "...", "tier_medium": "...", "tier_slow": "..." },
  "configured_providers": { ... }
}
```

---

## 8. Key Bindings Reference

### Global (Normal Mode)
| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next / previous main section |
| `j` / `k` | Navigate within section |
| `Space` | Cycle select field |
| `i` | Enter insert mode |
| `:` | Enter command mode |
| `Ctrl+S` | Save manifest |
| `Shift+M` | Open Provider Menu modal |
| `Ctrl+C` | Quit |

### Command Mode
| Command | Action |
|---------|--------|
| `:w` / `:write` | Save |
| `:q` / `:quit` | Quit without save |
| `:wq` / `:x` | Save and quit |
| `:tabn` / `:bn` | Next section |
| `:tabp` / `:bp` | Previous section |
| `:1`–`:8` | Jump to section N |

### Sub-Editor (varies by tab)
| Key | Action |
|-----|--------|
| `a` | Add item (list view) |
| `d` | Delete item (list view) |
| `Enter` / `i` | Edit / insert mode |
| `h` / `l` | Switch sub-tab |
| `b` / `Esc` | Back to parent / exit insert |
| `F` | Drill into nested fields (DTOs) |
| `A` | Drill into attributes (Domains) |

---

## 9. Go Engineering Standards

- **Error handling:** Never swallow errors. Use `fmt.Errorf("context: %w", err)` for wrapping.
- **Immutability:** Favor passing structs by value. Return new copies rather than mutating in place.
- **File size:** 200–400 lines typical, 800 lines hard max. Split by feature/domain. Several files already exceed this — do not add to them without first extracting helpers.
- **Formatting:** `gofmt` enforced. Run `go vet` before committing.
- **No cobra/viper:** This project uses raw `bubbletea` — do not add cobra or viper unless adding a non-interactive CLI mode.
- **Style constants:** All colors and styles live in `styles.go`. Do not inline lipgloss colors elsewhere.
- **Shared rendering:** Add new rendering helpers to `render_helpers.go`, not inline in sub-editors.
- **Field abstraction:** New form fields use the `Field` struct with `KindText`, `KindSelect`, or `KindTextArea`. Never render raw text inputs directly in sub-editors.
- **Tab navigation:** Use `NavigateTab()` from `nav.go` for `h`/`l` sub-tab switching — do not duplicate this switch in new editors.
- **Vim list navigation:** Use `VimNav` from `nav.go` for `j`/`k`/`gg`/`G`/count-prefix in any new editor with a navigable list.
- **Editor interface:** New sub-editors must implement the `Editor` interface (`Mode()`, `HintLine()`, `View()`). Register them in `buildSectionRegistry()` in `model_sections.go` — add one `sectionEntry` with `editor` and `update` closures, and add the section ID to `sectionOrder`.
- **Manifest types:** Add new pillar types to the appropriate `manifest_*.go` file, not to `manifest.go`. Only the root `Manifest` struct and `Save()` belong in `manifest.go`.
- **Model registry:** Add new AI providers or model tiers to `providerModels` in `orchestrator/models.go`. Do not add new switch cases to `resolveAgent()`.
- **Skill aliases:** Add new technology aliases to `aliasMap` in `skills/aliases.go`. Universal skills for a task kind go in `universalSkillsForKind` in the same file.
- **Task roles:** Add role-specific prompt instructions for new `TaskKind` values in `taskRoleDescriptions` in `agent/roles.go`.
- **Model tiering:** Add default tier assignments for new `TaskKind` values in `defaultTierForKind` in `orchestrator/tier.go`.

---

## 10. Specification Reference

`system-declaration-menu.md` is the canonical specification for all menu options, field names, and valid values across all 7 pillars. When adding or modifying any editor field, cross-reference this document to ensure alignment.

The dependency graph for non-linear resolution:
```
Description (free-text project overview)
    ↓
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains; defines Auth Roles; Stack Configs)
    ↓
Contracts (DTOs reference Domains; Endpoints reference Service Units + Auth Roles; External APIs)
    ↓
Frontend (Pages reference Endpoints + DTOs + Auth Roles from Backend; Components + Assets + i18n + A11y)
    ↓
Infrastructure (references all deployable units; named environments)
    ↓
Cross-Cutting (Testing frameworks filtered by Backend languages + Frontend tech; Docs formats)
    ↓
Realize (Code generation config — orchestrates multi-provider generation for all pillars)
```

Empty references show as "unlinked" placeholders — the UI must allow editing in any order.
