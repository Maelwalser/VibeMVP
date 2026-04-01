# vibeMVP — Project Description & Engineering Standards

## 1. Project Overview

**vibeMVP** is an interactive Terminal User Interface (TUI) CLI tool for declaratively specifying a complete software system architecture. It implements a vim-inspired editor that lets developers and architects define comprehensive system manifests across 6 architectural pillars — backend, data, contracts, frontend, infrastructure, and cross-cutting concerns.

The resulting manifest is serialized to `manifest.json` and intended for downstream consumption by code-generation agents or tooling.

**Key design principles:**
- Vim-modal editing (Normal / Insert / Command modes)
- Tokyo Night dark theme throughout
- Non-linear editing — users can fill any tab in any order
- Pillar-based dependency graph: Data → Backend → Contracts → Frontend → Infrastructure → Cross-Cutting

---

## 2. Technology Stack

| Concern | Choice |
|---------|--------|
| Language | Go 1.26.1 |
| TUI framework | `github.com/charmbracelet/bubbletea` v1.3.10 |
| TUI components | `github.com/charmbracelet/bubbles` v1.0.0 (textarea, textinput) |
| Styling/layout | `github.com/charmbracelet/lipgloss` v1.1.0 |
| TUI entry point | `cmd/agent/main.go` |
| Realize entry point | `cmd/realize/main.go` |
| Manifest types | `internal/manifest/manifest.go` |
| UI components | `internal/ui/` (12 files, ~11,700 lines) |
| Code generation engine | `internal/realize/` (DAG, agent, skills, verifiers, orchestrator) |
| Claude model | `claude-opus-4-6` (realize default) |

---

## 3. Project Structure

```
vibeMVP/
├── cmd/
│   ├── agent/
│   │   └── main.go              # TUI entry point — sets up save callback, runs Bubble Tea program
│   └── realize/
│       └── main.go              # Code-gen entry point — CLI flags, runs orchestrator
├── internal/
│   ├── manifest/
│   │   └── manifest.go          # All data model types (~745 lines); JSON serialization
│   ├── ui/
│   │   ├── model.go             # Root TUI model, vim modes, tab routing (~651 lines)
│   │   ├── styles.go            # Tokyo Night palette, all lipgloss styles (~145 lines)
│   │   ├── sections.go          # Section/field definitions, FieldKind enum (~189 lines)
│   │   ├── render_helpers.go    # Shared rendering utilities (~449 lines)
│   │   ├── backend_editor.go    # Backend tab — env, services, comm, messaging, gateway, auth (~2,418 lines)
│   │   ├── data_tab_editor.go   # Data tab — databases, domains, caching, file storage (~1,561 lines)
│   │   ├── data_editor.go       # Entity/column schema editor (~1,179 lines)
│   │   ├── db_editor.go         # Database source editor (~533 lines)
│   │   ├── contracts_editor.go  # DTOs, endpoints, versioning (~1,492 lines)
│   │   ├── frontend_editor.go   # Tech stack, theming, pages, navigation (~1,256 lines)
│   │   ├── infra_editor.go      # Networking, CI/CD, observability (~585 lines)
│   │   └── crosscut_editor.go   # Testing, documentation (~520 lines)
│   └── realize/
│       ├── agent/
│       │   ├── agent.go         # Claude API client, tool-use loop (~134 lines)
│       │   ├── context.go       # Agent context struct (~13 lines)
│       │   └── prompt.go        # System/user prompt builders (~118 lines)
│       ├── dag/
│       │   ├── dag.go           # DAG struct, topological sort, cycle detection (~149 lines)
│       │   ├── builder.go       # Manifest → DAG task graph construction (~399 lines)
│       │   └── payload.go       # Task payload types (~39 lines)
│       ├── orchestrator/
│       │   ├── orchestrator.go  # Config, entrypoint, task dispatch (~202 lines)
│       │   └── runner.go        # Per-task runner: agent call + verify + retry (~103 lines)
│       ├── output/
│       │   └── writer.go        # File output writer (~74 lines)
│       ├── skills/
│       │   ├── registry.go      # In-memory skill registry (~17 lines)
│       │   └── loader.go        # Load skill markdown files from disk (~239 lines)
│       └── verify/
│           ├── verifier.go      # Verifier interface + factory (~130 lines)
│           ├── go_verifier.go   # go build + go vet verifier (~113 lines)
│           ├── ts_verifier.go   # tsc verifier (~59 lines)
│           ├── python_verifier.go # python -m py_compile verifier (~63 lines)
│           ├── tf_verifier.go   # terraform validate verifier (~64 lines)
│           └── null_verifier.go # No-op verifier for unknown languages (~13 lines)
├── system-declaration-menu.md   # Full specification: all options for every field
├── go.mod / go.sum
└── LICENSE
```

File size budget: **800 lines max** per file. Extract utilities if approaching this limit.

> Several UI files already exceed 800 lines due to accumulated features. When touching these files, extract helpers to `render_helpers.go` or split sub-tab logic into dedicated files.

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

### 4.2 Section Delegation Pattern

Each of the 6 main tabs is a self-contained sub-editor. The root model delegates:
- `Update(msg)` — event routing
- `View(w, h)` — rendering
- `Mode()` — current mode (root uses this for status line)
- `HintLine()` — bottom keybinding hints
- `ToManifest[X]Pillar()` — serializes editor state to manifest types

The **KindDataModel** sentinel field in `sections.go` signals full delegation to the sub-editor.

### 4.3 List+Form Pattern (used in most sub-editors)

```
SubView: List → user presses Enter → SubView: Form → Esc → SubView: List
```

Lists show items with `j/k` navigation. `a` adds, `d` deletes, `Enter`/`i` edits. Forms use unified `renderFormFields()` from `render_helpers.go`.

### 4.4 Manifest Builder Pattern

Each sub-editor implements `ToManifest[X]Pillar()` converting in-memory form state to the canonical manifest structs. `BuildManifest()` in `model.go` calls all six to assemble the final `manifest.Manifest`.

### 4.5 Rendering Layout

All form fields use a consistent vim-style layout via `renderFormFields()`:
```
[LineNo] [Label          ] = [Value]
   3          14            3    (remaining width)
```

Tab bars use `renderSubTabBar()`. Bottom hints use `hintBar()`.

---

## 5. The 6 Architectural Pillars

### Pillar 1 — Backend (`BackendEditor`)
Sub-tabs: **Env** · **Services** · **Communication** · **Messaging** · **API Gateway** · **Auth**

- Architecture pattern selector (Monolith / Modular Monolith / Microservices / Event-Driven / Hybrid) conditionally shows/hides sub-tabs
- Services list with per-service: name, responsibility, language, framework (dynamically filtered by language), pattern tag
- Communication links: from/to service, protocol, direction, trigger, sync/async, resilience patterns
- Messaging: broker config + repeatable event catalog
- Auth: strategy, identity provider, authorization model, token storage, MFA

### Pillar 2 — Data (`DataTabEditor` + `DBEditor` + `DataEditor`)
Sub-tabs: **Databases** · **Domains** · **Caching** · **File Storage**

- Databases: alias, category, technology (filtered by category), hosting, HA mode — with type-conditional fields (SSL mode, eviction policy, replication factor, etc.)
- Domains: bounded contexts with repeatable attributes (name, type, constraints, default, sensitive, validation) and relationships (type, FK field, cascade)
- Entities (legacy model): similar to domains but in separate `data_editor.go`
- Caching layer config; File/object storage config

### Pillar 3 — Contracts (`ContractsEditor`)
Sub-tabs: **DTOs** · **Endpoints** · **Versioning**

- DTOs: name, category (Request/Response/Event Payload/Shared), source domain, nested fields list with per-field type/validation
- Endpoints: protocol-specific forms — REST (method, path params, query params, pagination), GraphQL (operation type), gRPC (service, method, stream type), WebSocket (channel, client/server events)
- Versioning: strategy, current version, deprecation policy

### Pillar 4 — Frontend (`FrontendEditor`)
Sub-tabs: **Tech** · **Theme** · **Pages** · **Navigation**

- Tech: language, platform, framework (filtered by language+platform), meta-framework, styling, component library, state management, data fetching, form handling, validation
- Theme: dark mode strategy, border radius, spacing scale, elevation, motion
- Pages: route, auth required, layout, core actions, loading/error strategy
- Navigation: nav type, breadcrumbs, auth-aware toggle

### Pillar 5 — Infrastructure (`InfraEditor`)
Sub-tabs: **Networking** · **CI/CD** · **Observability**

- Networking: DNS, TLS, reverse proxy, CDN
- CI/CD: platform, container registry, deploy strategy, IaC tool, secrets management
- Observability: logging, metrics, tracing, error tracking, health checks, alerting

### Pillar 6 — Cross-Cutting (`CrosscutEditor`)
Sub-tabs: **Testing** · **Docs**

- Testing: unit, integration, E2E, API, load, contract testing tool selections
- Docs: API doc format, auto-generation toggle, changelog strategy

---

## 6. Realize Engine (Code Generation)

`cmd/realize` is the downstream consumer of `manifest.json`. It drives an agentic code-generation pipeline.

### 6.1 Pipeline Overview

```
manifest.json
    ↓
dag.Builder.Build()   → execution DAG (tasks with dependency edges)
    ↓
orchestrator.Run()    → parallel task dispatch (bounded by --parallel flag)
    ↓  (per task)
runner.Run()          → agent.Call() → verify.Check() → retry up to MaxRetries
    ↓
output.Writer         → writes generated files under --output directory
```

### 6.2 DAG Task IDs

Tasks follow a naming convention derived from manifest entries:

| Pattern | Example |
|---------|---------|
| `data.<alias>` | `data.postgres` |
| `svc.<name>` | `svc.api-gateway` |
| `contracts` | `contracts` |
| `frontend` | `frontend` |
| `infra.<component>` | `infra.networking` |
| `crosscut.<component>` | `crosscut.testing` |

### 6.3 Skills System

Skills are markdown files in `.vibemvp/skills/` (configurable via `--skills`). Each file defines a named generation skill. The `skills.Loader` reads them at startup; the `skills.Registry` makes them available to the agent prompt builder.

### 6.4 Verifiers

After each agent call, a language-appropriate verifier checks the output:

| Language | Verifier | Check |
|----------|----------|-------|
| Go | `go_verifier` | `go build` + `go vet` |
| TypeScript | `ts_verifier` | `tsc --noEmit` |
| Python | `python_verifier` | `python -m py_compile` |
| Terraform | `tf_verifier` | `terraform validate` |
| Other | `null_verifier` | always passes |

### 6.5 CLI Flags

```
--manifest  path to manifest.json      (default: manifest.json)
--output    output directory            (default: output)
--skills    skills directory            (default: .vibemvp/skills)
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
  "backend":    { "arch_pattern": "...", "services": [...], ... },
  "data":       { "databases": [...], "domains": [...], ... },
  "contracts":  { "dtos": [...], "endpoints": [...], ... },
  "frontend":   { "tech": {...}, "pages": [...], ... },
  "infrastructure": { "networking": {...}, "cicd": {...}, ... },
  "cross_cutting":  { "testing": {...}, "docs": {...} },
  "entities":   [...],
  "databases":  [...]
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
| `Ctrl+C` | Quit |

### Command Mode
| Command | Action |
|---------|--------|
| `:w` / `:write` | Save |
| `:q` / `:quit` | Quit without save |
| `:wq` / `:x` | Save and quit |
| `:tabn` / `:bn` | Next section |
| `:tabp` / `:bp` | Previous section |
| `:1`–`:6` | Jump to section N |

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
- **Style constants:** All colors and styles live in `styles.go`. Do not inline lipgloss colors elsewhere.
- **Shared rendering:** Add new rendering helpers to `render_helpers.go`, not inline in sub-editors.
- **Field abstraction:** New form fields use the `Field` struct with `KindText`, `KindSelect`, or `KindTextArea`. Never render raw text inputs directly in sub-editors.

---

## 10. Specification Reference

`system-declaration-menu.md` is the canonical specification for all menu options, field names, and valid values across all 6 pillars. When adding or modifying any editor field, cross-reference this document to ensure alignment.

The dependency graph for non-linear resolution:
```
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains)
    ↓
Contracts (DTOs reference Domains; Endpoints reference Service Units)
    ↓
Frontend (Pages reference Endpoints + DTOs)
    ↓
Infrastructure (references all deployable units)
    ↓
Cross-Cutting (references everything)
```

Empty references show as "unlinked" placeholders — the UI must allow editing in any order.
