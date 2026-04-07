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
| Manifest types | `internal/manifest/` (11 files, split by pillar) |
| UI components | `internal/ui/` (14 subdirectories, 87 files, ~31,400 lines) |
| Code generation engine | `internal/realize/` (DAG, agent, skills, verifiers, orchestrator, memory) |
| Bundled resources | `internal/bundled/` (skills markdown + loader patches) |
| Default realize model | `claude-sonnet-4-6` (tier-dependent; see Section 6.3) |

---

## 3. Project Structure

```
VibeMenu/
├── cmd/
│   ├── agent/
│   │   └── main.go                  # TUI entry point — sets up save callback, runs Bubble Tea program (50 lines)
│   └── realize/
│       └── main.go                  # Code-gen entry point — CLI flags, runs orchestrator (55 lines)
├── internal/
│   ├── bundled/
│   │   ├── bundled.go               # Bundled resource embedding (20 lines)
│   │   ├── skills/                   # Bundled skill markdown files
│   │   └── loader-patches/           # Loader patch files per language/platform
│   ├── manifest/
│   │   ├── manifest.go              # Root Manifest struct + Save() + RealizeOptions (222 lines)
│   │   ├── manifest_enums.go        # All enum type declarations (186 lines)
│   │   ├── manifest_data.go         # DataPillar, DBSourceDef, DomainDef, caching types (175 lines)
│   │   ├── manifest_backend.go      # BackendPillar, ServiceDef, CommLink, AuthConfig, RoleDef, PermissionDef, WAFConfig, JobQueueDef, CronJobDef (214 lines)
│   │   ├── manifest_contracts.go    # ContractsPillar, DTODef, EndpointDef, APIVersioning, ExternalAPIDef, ExternalAPIInteraction (140 lines)
│   │   ├── manifest_frontend.go     # FrontendPillar, FrontendTechConfig, FrontendTheme, PageDef, PageComponentDef, ComponentActionDef, NavigationConfig, I18nConfig, A11ySEOConfig, AssetDef (159 lines)
│   │   ├── manifest_infra.go        # InfraPillar, NetworkingConfig, CICDConfig, ObservabilityConfig, ServerEnvironmentDef (56 lines)
│   │   ├── manifest_crosscut.go     # CrossCutPillar, TestingConfig, DocsConfig (37 lines)
│   │   ├── providers.go             # Provider configuration types (56 lines)
│   │   ├── recent.go                # Recent manifest tracking (69 lines)
│   │   └── manifest_test.go         # Manifest unit tests (232 lines)
│   ├── ui/                          # No files in root — all organized into subdirectories
│   │   ├── app/                     # Core application model (1,181 lines)
│   │   │   ├── app.go               # App initialization and setup (106 lines)
│   │   │   ├── model.go             # Root TUI model, vim modes, Update + command dispatch (566 lines)
│   │   │   ├── model_sections.go    # Section registry: editor getters + update closures (161 lines)
│   │   │   └── model_view.go        # Root View() and all render helpers (448 lines)
│   │   ├── arch/                    # Architecture visualization (1,933 lines)
│   │   │   ├── arch_canvas.go       # Canvas rendering primitives (192 lines)
│   │   │   ├── arch_graph.go        # Graph data structures and layout (389 lines)
│   │   │   ├── arch_info.go         # Architecture info panel content (619 lines)
│   │   │   ├── arch_render.go       # Architecture diagram rendering (907 lines)
│   │   │   └── arch_screen.go       # Architecture screen model + update (426 lines)
│   │   ├── backend/                 # Backend pillar editor (8,432 lines)
│   │   │   ├── backend_editor.go    # Struct, init, Update dispatcher (389 lines)
│   │   │   ├── backend_manifest.go  # ToManifest conversion (473 lines)
│   │   │   ├── backend_fields.go    # Default field constructors (585 lines)
│   │   │   ├── backend_env_fields.go # Environment field constructors (414 lines)
│   │   │   ├── backend_services.go  # Service list/form + comm list/form + messaging handlers (744 lines)
│   │   │   ├── backend_update.go    # Dropdown, insert update handlers (660 lines)
│   │   │   ├── backend_view.go      # HintLine, View, sub-tab render functions (754 lines)
│   │   │   ├── backend_auth.go      # Auth + security update handlers and view (737 lines)
│   │   │   ├── backend_security.go  # Security-specific handlers (211 lines)
│   │   │   ├── backend_config.go    # Stack config list/form handlers (238 lines)
│   │   │   ├── backend_jobs.go      # Jobs list/form handlers (228 lines)
│   │   │   ├── backend_repos.go     # Repository pattern support (495 lines)
│   │   │   ├── backend_repos_ops.go # Repository operations handlers (415 lines)
│   │   │   ├── backend_repos_view.go # Repository view rendering (163 lines)
│   │   │   └── backend_setters.go   # Field setter utilities (540 lines)
│   │   ├── contracts/               # Contracts pillar editor (2,909 lines)
│   │   │   ├── contracts_editor.go  # Struct, init, ToManifest, Update dispatcher (686 lines)
│   │   │   ├── contracts_fields.go  # DTO field constructors (233 lines)
│   │   │   ├── contracts_fields_endpoints.go # Endpoint/versioning/external field constructors (707 lines)
│   │   │   ├── contracts_dtos.go    # DTO list/form + field drill-down + viewDTOs (529 lines)
│   │   │   └── contracts_endpoints.go # Endpoint/versioning/external update + views (754 lines)
│   │   ├── core/                    # Shared utilities and types (4,096 lines)
│   │   │   ├── editor.go            # Editor interface + Mode enum (43 lines)
│   │   │   ├── editor_state.go      # DropdownState shared struct (20 lines)
│   │   │   ├── nav.go               # NavigateTab(), VimNav struct — shared navigation helpers (148 lines)
│   │   │   ├── nav_test.go          # Navigation unit tests (315 lines)
│   │   │   ├── sections.go          # Section/field definitions, FieldKind enum (374 lines)
│   │   │   ├── styles.go            # Tokyo Night palette, all lipgloss styles (314 lines)
│   │   │   ├── render_helpers.go    # Shared rendering utilities (renderFormFields, fillTildes, …) (495 lines)
│   │   │   ├── render_util.go       # Additional rendering utilities (405 lines)
│   │   │   ├── field_options.go     # Shared field option slices (OptionsOnOff, OptionsOffOn) (9 lines)
│   │   │   ├── field_descriptions.go # Field description text (491 lines)
│   │   │   ├── field_descriptions_contracts.go # Contract-specific field descriptions (396 lines)
│   │   │   ├── field_descriptions_infra.go # Infrastructure-specific field descriptions (315 lines)
│   │   │   ├── animation.go         # Animation utilities (69 lines)
│   │   │   ├── lang_versions.go     # Language version matrices for tech filtering (264 lines)
│   │   │   └── undo.go              # Generic UndoStack[T] + snapshot types (76 lines)
│   │   ├── crosscut/                # Cross-cutting concerns editor (1,113 lines)
│   │   │   ├── crosscut_editor.go   # Struct, init, ToManifest, Update dispatcher, View (570 lines)
│   │   │   └── crosscut_fields.go   # Testing/docs/standards field constructors (543 lines)
│   │   ├── data/                    # Data pillar editor (6,333 lines)
│   │   │   ├── data_tab_editor.go   # Struct, init, ToManifest, Update dispatcher, View (790 lines)
│   │   │   ├── data_tab_fields.go   # Database/domain/fs field constructors (526 lines)
│   │   │   ├── data_tab_fields_cache.go # Caching field constructors (400 lines)
│   │   │   ├── data_domains.go      # Domain list/form + attr/rel handlers + viewDomains (517 lines)
│   │   │   ├── data_domain_fields.go # Domain form field constructors (221 lines)
│   │   │   ├── data_caching_storage.go # Caching/governance/file-storage handlers + views (495 lines)
│   │   │   ├── data_editor.go       # Entity/column schema editor: struct, init, Mode, HintLine (143 lines)
│   │   │   ├── data_editor_fields.go # Entity editor: column + entity form helpers (277 lines)
│   │   │   ├── data_editor_update.go # Entity editor: all update handlers (449 lines)
│   │   │   ├── data_editor_view.go  # Entity editor: all view functions (414 lines)
│   │   │   ├── db_editor.go         # Database source editor (543 lines)
│   │   │   └── db_editor_fields.go  # Database form field constructors (198 lines)
│   │   ├── description/             # Description editor (128 lines)
│   │   │   └── description_editor.go # Pillar 0: free-text project description textarea (128 lines)
│   │   ├── frontend/                # Frontend pillar editor (4,050 lines)
│   │   │   ├── frontend_editor.go   # Struct, init, ToManifest, Update dispatcher (492 lines)
│   │   │   ├── frontend_editor_input.go # Insert-mode input handling (397 lines)
│   │   │   ├── frontend_fields.go   # Compatibility maps + default field constructors (504 lines)
│   │   │   ├── frontend_fields_pages.go # Page/component field constructors (524 lines)
│   │   │   ├── frontend_update.go   # Tech/theme update handlers + View (685 lines)
│   │   │   ├── frontend_pages_update.go # Page/navigation update handlers (226 lines)
│   │   │   ├── frontend_i18n_a11y.go # i18n + a11y/SEO update handlers + viewPages (308 lines)
│   │   │   ├── frontend_action_fields.go # Component action field constructors (268 lines)
│   │   │   └── frontend_assets.go   # Frontend asset management (251 lines)
│   │   ├── infra/                   # Infrastructure pillar editor (1,620 lines)
│   │   │   ├── infra_editor.go      # Struct, init, ToManifest, Update dispatcher (623 lines)
│   │   │   ├── infra_fields.go      # Provider maps, deploy strategies, field constructors (654 lines)
│   │   │   └── infra_view.go        # View rendering functions (343 lines)
│   │   ├── provider/                # Provider modal (1,153 lines)
│   │   │   ├── provider_menu.go     # Types, struct, init, state helpers (243 lines)
│   │   │   ├── provider_menu_oauth.go # OAuth 2.0 PKCE flow + credential step (304 lines)
│   │   │   ├── provider_menu_update.go # Update handler (167 lines)
│   │   │   ├── provider_menu_view.go # View + all render functions (339 lines)
│   │   │   └── credentials.go       # Credential management utilities (110 lines)
│   │   ├── realization/             # Realization screen (450 lines)
│   │   │   └── realization_screen.go # Code generation output screen (450 lines)
│   │   ├── realize_cfg/             # Realize configuration editor (499 lines)
│   │   │   └── realize_editor.go    # Code generation configuration form (499 lines)
│   │   └── welcome/                 # Welcome screen (333 lines)
│   │       └── welcome.go           # Welcome/initialization screen (333 lines)
│   └── realize/
│       ├── agent/
│       │   ├── agent.go             # Agent interface + ClaudeAgent implementation with streaming (161 lines)
│       │   ├── openai_agent.go      # OpenAI-compatible agent (ChatGPT, Mistral, Llama via Groq) (144 lines)
│       │   ├── gemini_agent.go      # Google Gemini agent implementation (135 lines)
│       │   ├── context.go           # Agent context struct (37 lines)
│       │   ├── prompt.go            # System/user prompt builders (548 lines)
│       │   └── roles.go             # taskRoleDescriptions map — detailed role instructions per TaskKind (467 lines)
│       ├── dag/
│       │   ├── dag.go               # DAG struct, topological sort, TaskKind enums (172 lines)
│       │   ├── builder.go           # Manifest → DAG task graph construction (661 lines)
│       │   ├── builder_helpers.go   # Builder helper functions (240 lines)
│       │   ├── builder_test.go      # Builder unit tests (247 lines)
│       │   ├── dag_test.go          # DAG unit tests (178 lines)
│       │   └── payload.go           # Task payload types (66 lines)
│       ├── config/
│       │   └── defaults.go          # Tunable constants (DefaultModel, DefaultMaxTokens, MaxSkillBytes, …) (43 lines)
│       ├── orchestrator/
│       │   ├── orchestrator.go      # Config, entrypoint, task dispatch (598 lines)
│       │   ├── models.go            # Provider model registry for all 6 providers, resolveModelID() (212 lines)
│       │   ├── runner.go            # Per-task runner: agent call + verify + retry (514 lines)
│       │   ├── reconcile.go         # Post-generation reconciliation logic (177 lines)
│       │   ├── repair.go            # Automated repair strategies (158 lines)
│       │   └── tier.go              # defaultTierForKind map + escalateModel() (87 lines)
│       ├── memory/
│       │   ├── memory.go            # SharedMemory — stores completed task outputs for downstream agents (338 lines)
│       │   ├── sigscan.go           # Signature extraction from generated source files (473 lines)
│       │   ├── memory_test.go       # Unit tests for SharedMemory (224 lines)
│       │   └── sigscan_test.go      # Unit tests for signature scanning (271 lines)
│       ├── state/
│       │   └── state.go             # State tracking during code generation (76 lines)
│       ├── output/
│       │   └── writer.go            # File output writer (100 lines)
│       ├── skills/
│       │   ├── registry.go          # In-memory skill registry (17 lines)
│       │   ├── aliases.go           # Technology alias map + universal skills per task kind (273 lines)
│       │   ├── aliases_test.go      # Alias unit tests (100 lines)
│       │   ├── extract.go           # Skill extraction utilities (47 lines)
│       │   └── loader.go            # Load skill markdown files from disk (108 lines)
│       ├── deps/
│       │   ├── resolver_interface.go # LanguageResolver interface + ResolverRegistry (34 lines)
│       │   ├── resolver.go          # deps.Agent — runs package manager to lock deps (255 lines)
│       │   ├── modules.go           # Shared: ResolvedDeps types, PromptContext, Save/Load (247 lines)
│       │   ├── library_docs.go      # Library API documentation fetching (294 lines)
│       │   ├── go_modules.go        # Go-specific: WellKnownGoModules, GoModForService, ValidateGoMod (463 lines)
│       │   └── npm_modules.go       # npm-specific: WellKnownNpmPackages, resolveNpmVersion (215 lines)
│       └── verify/
│           ├── verifier.go          # Verifier interface + Registry + ForTask() (134 lines)
│           ├── verifier_test.go     # Verifier unit tests (248 lines)
│           ├── go_verifier.go       # go build + go vet verifier (169 lines)
│           ├── ts_verifier.go       # tsc verifier (59 lines)
│           ├── python_verifier.go   # python -m py_compile verifier (97 lines)
│           ├── tf_verifier.go       # terraform validate verifier (64 lines)
│           ├── null_verifier.go     # No-op verifier for unknown languages (13 lines)
│           ├── fixer.go             # Deterministic fixes (gofmt, unused imports) before retry (99 lines)
│           ├── deterministic_fixes.go # Shared error pattern matching and fix dispatch (255 lines)
│           ├── deterministic_fixes_go.go # Go-specific deterministic fixes (713 lines)
│           ├── deterministic_fixes_ts.go # TypeScript-specific deterministic fixes (354 lines)
│           ├── deterministic_fixes_py.go # Python-specific deterministic fixes (161 lines)
│           ├── deterministic_fixes_helpers.go # Shared fix helper functions (52 lines)
│           ├── import_validator.go   # Import path validation (277 lines)
│           └── integration_verifier.go # Cross-file integration verification (179 lines)
├── system-declaration-menu.md       # Full specification: all options for every field
├── go.mod / go.sum
└── LICENSE
```

File size budget: **800 lines max** per file. Extract utilities if approaching this limit.

> A few files approach the limit (`arch_render.go` 907 lines, `data_tab_editor.go` 790 lines, `backend_view.go` 754 lines, `backend_services.go` 744 lines, `backend_auth.go` 737 lines, `deterministic_fixes_go.go` 713 lines, `contracts_fields_endpoints.go` 707 lines). When adding features to these files, extract helpers to dedicated files first.

---

## 4. Architecture

### 4.1 Vim Modal System

The root `Model` (`app/model.go`) owns three modes, defined in `core/editor.go`:

```go
type Mode int
const (
    ModeNormal   // Navigation: Tab/Shift-Tab between sections, j/k within
    ModeInsert   // Text input: i to enter, Esc to exit
    ModeCommand  // :w :q :wq :tabn :tabp :1-6 :help
)
```

### 4.2 Editor Interface + Polymorphic Dispatch

All sub-editors implement the `Editor` interface defined in `core/editor.go`:

```go
type Editor interface {
    Mode() Mode
    HintLine() string
    View(w, h int) string
}
```

The root `Model` uses `activeEditor() Editor` and `delegateUpdate()` — both dispatch through `sectionRegistry` in `app/model_sections.go` rather than switch statements. Adding a new pillar requires one `sectionEntry` registration in `buildSectionRegistry()`; no other files need changing.

Each sub-editor also implements:
- `ToManifest[X]Pillar()` — serializes editor state to manifest types

The **KindDataModel** sentinel field in `core/sections.go` signals full delegation to the sub-editor.

### 4.3 Shared Navigation Utilities (`core/nav.go`)

Two reusable helpers replace duplicated navigation code across all sub-editors:

**`NavigateTab(key, active, maxTabs int) int`** — handles `h`/`l`/`left`/`right` tab switching. Used in every editor that has sub-tabs.

**`VimNav` struct** — stateful count-prefix + vim motion handler:
```go
type VimNav struct { CountBuf string; GBuf bool }
// Handle returns (newIdx, consumed). consumed=false for enter/space/i/a (caller handles those).
func (v *VimNav) Handle(key string, idx, n int) (int, bool)
func (v *VimNav) Reset()
```
Handles: digit accumulation, `j`/`k` with count multiplier, `gg` (top), `G` (bottom).

### 4.4 Undo System (`core/undo.go`)

Generic `UndoStack[T]` with bounded depth (50 entries). Provides `Push`, `Pop`, `Len`. Includes helper functions `CopySlice` and `CopyFieldItems` for creating safe snapshots, plus typed snapshot structs (`SvcSnapshot`, `CommSnapshot`, `EventSnapshot`) for editors that maintain parallel Field-items and manifest slices.

### 4.5 List+Form Pattern (used in most sub-editors)

```
SubView: List → user presses Enter → SubView: Form → Esc → SubView: List
```

Lists show items with `j/k` navigation. `a` adds, `d` deletes, `Enter`/`i` edits. Forms use unified `renderFormFields()` from `core/render_helpers.go`.

### 4.6 Manifest Builder Pattern

Each sub-editor implements `ToManifest[X]Pillar()` converting in-memory form state to the canonical manifest structs. `BuildManifest()` in `app/model.go` calls all seven to assemble the final `manifest.Manifest`.

### 4.7 Model Sub-Structs

The root `Model` struct in `app/model.go` groups related fields into sub-structs to reduce coupling:

```go
type cmdState    struct { buffer, status string; isErr bool }
type modalState  struct { open bool; menu ProviderMenu }
type realizeState struct { screen RealizationScreen; show, triggered bool }
```

### 4.8 Rendering Layout

All form fields use a consistent vim-style layout via `renderFormFields()` in `core/render_helpers.go`:
```
[LineNo] [Label          ] = [Value]
   3          14            3    (remaining width)
```

Tab bars use `renderSubTabBar()`. Bottom hints use `hintBar()`. Additional rendering utilities live in `core/render_util.go`.

### 4.9 UI Package Organization

The `internal/ui/` package is organized into 14 subdirectories with no files in the root:

| Directory | Purpose | Files | Lines |
|-----------|---------|-------|-------|
| `app/` | Root TUI model, lifecycle, view dispatch | 4 | 1,181 |
| `arch/` | Architecture visualization diagrams | 5 | 1,933 |
| `backend/` | Backend pillar editor (services, auth, jobs, repos) | 15 | 8,432 |
| `contracts/` | Contracts pillar editor (DTOs, endpoints, external APIs) | 5 | 2,909 |
| `core/` | Shared types, styles, rendering, navigation, undo | 15 | 4,096 |
| `crosscut/` | Cross-cutting concerns editor (testing, docs) | 2 | 1,113 |
| `data/` | Data pillar editor (databases, domains, entities, caching) | 12 | 6,333 |
| `description/` | Free-text description editor | 1 | 128 |
| `frontend/` | Frontend pillar editor (tech, theme, pages, i18n, a11y) | 9 | 4,050 |
| `infra/` | Infrastructure pillar editor (networking, CI/CD, observability) | 3 | 1,620 |
| `provider/` | Provider modal + credential management | 5 | 1,153 |
| `realization/` | Code generation progress screen | 1 | 450 |
| `realize_cfg/` | Realize configuration form | 1 | 499 |
| `welcome/` | Welcome/initialization screen | 1 | 333 |

### 4.10 Field Descriptions

Context-sensitive field descriptions are split across three files in `core/`:
- `field_descriptions.go` — general and backend field descriptions (491 lines)
- `field_descriptions_contracts.go` — contracts-specific field descriptions (396 lines)
- `field_descriptions_infra.go` — infrastructure-specific field descriptions (315 lines)

---

## 5. The 8 Sections (Description + 7 Pillars)

### Section 0 — Description (`description/DescriptionEditor`)

Free-text project description textarea. Allows users to describe the project in natural language before filling in the structured pillars. Content is saved to `manifest.json` under the description field.

### Pillar 1 — Backend (`backend/BackendEditor`)
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
- Repository pattern support (`backend_repos.go`, `backend_repos_ops.go`, `backend_repos_view.go`)

### Pillar 2 — Data (`data/DataTabEditor` + `data/DBEditor` + `data/DataEditor`)
Sub-tabs: **Databases** · **Domains** · **Caching** · **File Storage**

- Databases: alias, category, technology (filtered by category), hosting, HA mode — with type-conditional fields (SSL mode, eviction policy, replication factor, etc.)
- Domains: bounded contexts with repeatable attributes (name, type, constraints, default, sensitive, validation) and relationships (type, FK field, cascade)
- Entities (legacy model): similar to domains but in separate `data/data_editor.go`
- Caching layer config; File/object storage config

### Pillar 3 — Contracts (`contracts/ContractsEditor`)
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

### Pillar 4 — Frontend (`frontend/FrontendEditor`)
Sub-tabs: **Tech** · **Theme** · **Pages** · **Navigation** · **i18n** · **A11y/SEO**

- Tech: language, platform, framework (filtered by language+platform), meta-framework, package manager, styling, component library, state management, data fetching, form handling, validation, PWA support, realtime strategy, image optimization, auth flow type, error boundary, bundle optimization, frontend testing, frontend linter
- Theme: dark mode strategy, border radius, spacing scale, elevation, motion, vibe, colors, description
- Pages: route, auth_required, layout, core actions, loading strategy, error handling, auth_roles (multi-select from backend roles for role-based page access), linked pages
  - Pages can define `PageComponentDef` entries with `ComponentActionDef` — 12+ action types (Fetch, Submit, Download, Upload, Delete, Refresh, Export, Navigate, Toast, State, Custom)
- Navigation: nav type (sidebar, top bar, etc.), breadcrumbs toggle, auth-aware navigation toggle
- Assets: frontend design assets (images, icons, fonts, videos, mockups, etc.) with usage classification (project or inspiration) via `AssetDef`
- i18n: internationalization settings (`I18nConfig`)
- A11y/SEO: accessibility and SEO configuration (`A11ySEOConfig`)

### Pillar 5 — Infrastructure (`infra/InfraEditor`)
Sub-tabs: **Networking** · **CI/CD** · **Observability**

- Networking: DNS, TLS, reverse proxy, CDN
- CI/CD: platform, container registry, deploy strategy, IaC tool, secrets management
- Observability: logging, metrics, tracing, error tracking, health checks, alerting (`ObservabilityConfig`)
- Server environments: named environments with compute/cloud/orchestrator settings (`ServerEnvironmentDef`)

### Pillar 6 — Cross-Cutting (`crosscut/CrosscutEditor`)
Sub-tabs: **Testing** · **Docs**

- Testing: testing framework selections dynamically filtered by backend languages and frontend tech choices
  - Unit: language-specific test framework (Jest, Vitest for JavaScript/TypeScript; pytest, Go testing, JUnit, xUnit for others)
  - Integration: integration test framework
  - E2E: end-to-end test tool (Playwright, Cypress, Nightwatch, Selenium, etc.)
  - API: API testing tool (REST, GraphQL, gRPC specific)
  - Load: load testing tool (k6, Locust, Apache JMeter, etc.)
  - Contract: contract testing tool (Pact, Spring Cloud Contract)
- Docs: API doc format (OpenAPI/Swagger, GraphQL schema doc, AsyncAPI, etc.), auto-generation toggle, changelog strategy

### Pillar 7 — Realize (`realize_cfg/RealizeEditor`)
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
reconcile + repair        → post-generation reconciliation and automated repair
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

`roles.go` contains `taskRoleDescriptions` — a per-`TaskKind` map of detailed role instructions injected into every system prompt, providing context-specific guidance for each code generation task (467 lines).

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

Skills are markdown files in `.vibemenu/skills/` (configurable via `--skills`) and bundled in `internal/bundled/skills/`. Each file defines a named generation skill. The `skills.Loader` reads them at startup; the `skills.Registry` makes them available to the agent prompt builder. Technology aliases in `skills/aliases.go` map framework names to canonical skill file names. `skills/extract.go` provides skill extraction utilities.

### 6.7 Verifiers + Deterministic Fixes

Before each retry, `fixer.go` applies zero-LLM deterministic fixes to save token cost. Fixes are split by language:
- `deterministic_fixes.go` — shared dispatch and error pattern matching (255 lines)
- `deterministic_fixes_go.go` — Go-specific fixes: gofmt, unused imports, common error patterns (713 lines)
- `deterministic_fixes_ts.go` — TypeScript-specific fixes (354 lines)
- `deterministic_fixes_py.go` — Python-specific fixes (161 lines)
- `deterministic_fixes_helpers.go` — shared fix helper functions (52 lines)

Additional verification:
- `import_validator.go` — validates import paths across generated files (277 lines)
- `integration_verifier.go` — cross-file integration verification (179 lines)

After fixing, the language-appropriate verifier checks the output:

| Language | Verifier | Check |
|----------|----------|-------|
| Go | `go_verifier` | `go build` + `go vet` |
| TypeScript | `ts_verifier` | `tsc --noEmit` |
| Python | `python_verifier` | `python -m py_compile` |
| Terraform | `tf_verifier` | `terraform validate` |
| Other | `null_verifier` | always passes |

### 6.8 Orchestrator Extensions

- `orchestrator/reconcile.go` — post-generation reconciliation logic that ensures consistency across generated files (177 lines)
- `orchestrator/repair.go` — automated repair strategies for common generation failures (158 lines)

### 6.9 Dependency Resolution (`realize/deps/`)

- `library_docs.go` — fetches and caches library API documentation for agent context (294 lines)
- `go_modules.go` — Go module resolution with well-known module registry (463 lines)
- `npm_modules.go` — npm package resolution (215 lines)

### 6.10 CLI Flags

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
- **File size:** 200–400 lines typical, 800 lines hard max. Split by feature/domain.
- **Formatting:** `gofmt` enforced. Run `go vet` before committing.
- **No cobra/viper:** This project uses raw `bubbletea` — do not add cobra or viper unless adding a non-interactive CLI mode.
- **Style constants:** All colors and styles live in `core/styles.go`. Do not inline lipgloss colors elsewhere.
- **Shared rendering:** Add new rendering helpers to `core/render_helpers.go` or `core/render_util.go`, not inline in sub-editors.
- **Field abstraction:** New form fields use the `Field` struct (defined in `core/sections.go`) with `KindText`, `KindSelect`, or `KindTextArea`. Never render raw text inputs directly in sub-editors.
- **Field descriptions:** Add new field descriptions to the appropriate `core/field_descriptions*.go` file.
- **Tab navigation:** Use `NavigateTab()` from `core/nav.go` for `h`/`l` sub-tab switching — do not duplicate this switch in new editors.
- **Vim list navigation:** Use `VimNav` from `core/nav.go` for `j`/`k`/`gg`/`G`/count-prefix in any new editor with a navigable list.
- **Undo support:** Use `UndoStack[T]` from `core/undo.go` for editors that need undo functionality.
- **Editor interface:** New sub-editors must implement the `Editor` interface (`Mode()`, `HintLine()`, `View()`) from `core/editor.go`. Register them in `buildSectionRegistry()` in `app/model_sections.go` — add one `sectionEntry` with `editor` and `update` closures, and add the section ID to `sectionOrder`.
- **Package organization:** Each pillar editor lives in its own subdirectory under `internal/ui/`. Shared types and utilities live in `core/`. The root `internal/ui/` directory contains no files — only subdirectories.
- **Manifest types:** Add new pillar types to the appropriate `manifest_*.go` file, not to `manifest.go`. Only the root `Manifest` struct and `Save()` belong in `manifest.go`. Provider types live in `providers.go`.
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
