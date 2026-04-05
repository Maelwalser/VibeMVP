# VibeMenu
What vibe is on the menu today?

<img width="1900" height="1140" alt="Pasted image" src="https://github.com/user-attachments/assets/4c636af3-6a08-4acb-b7f1-07e6e4965c7e" />


A vim-inspired TUI for declaratively specifying a complete software system architecture. Define your stack across 8 structured sections, then generate a `manifest.json` for downstream code generation via the `realize` pipeline.

> Still in development — not yet production-stable.

## Table of Contents

- [Installation](#installation)
- [The TUI Editor](#the-tui-editor)
  - [Key Bindings](#key-bindings)
  - [Sections Overview](#sections-overview)
- [The Architecture Overview](#architecture-diagram-overview)
- [manifest.json Reference](#manifestjson-reference)
- [Provider Configuration](#provider-configuration)
- [Code Generation (`realize`)](#code-generation-realize)
  - [Model Tiering](#model-tiering)
  - [Verification](#verification)
- [Skills System](#skills-system)

---

## Installation

### macOS / Linux — install script

```bash
curl -fsSL https://raw.githubusercontent.com/vibe-menu/vibemenu/main/install.sh | bash
```

Installs `vibemenu` and `realize` to `/usr/local/bin` (override with `INSTALL_DIR`).

### Specific version

```bash
VIBEMENU_VERSION=v1.0.0 bash install.sh
```

### Manual download

Pre-built binaries for every platform are attached to each [GitHub Release](https://github.com/vibe-menu/vibemenu/releases):

| Platform | Archive |
|----------|---------|
| Linux x86-64 | `vibemenu-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `vibemenu-<version>-linux-arm64.tar.gz` |
| macOS x86-64 | `vibemenu-<version>-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `vibemenu-<version>-darwin-arm64.tar.gz` |
| Windows x86-64 | `vibemenu-<version>-windows-amd64.zip` |

Each archive contains two binaries: `vibemenu` (TUI editor) and `realize` (code generation).

### Build from source

```bash
git clone https://github.com/vibe-menu/vibemenu
cd vibemenu
go build -o vibemenu ./cmd/agent
go build -o realize  ./cmd/realize
```

Requires Go 1.26+.

### Skills (bundled — no extra setup needed)

Skill files are **embedded in the `realize` binary**. On first run, they are automatically extracted to `.vibemenu/skills/` in the current directory:

```
realize --manifest manifest.json
# realize: extracting bundled skills to .vibemenu/skills
```

Existing files are never overwritten, so you can safely customise the skills directory. Point `realize` at a different location with `--skills`:

```bash
realize --skills /path/to/custom/skills --manifest manifest.json
```

---

## The TUI Editor

VibeMenu uses a vim-modal editing system with three modes:

| Mode | How to enter | Purpose |
|------|-------------|---------|
| **Normal** | `Esc` | Navigate sections, lists, tabs |
| **Insert** | `i` | Type into text fields |
| **Command** | `:` | Save, quit, jump to section |

### Key Bindings

#### Global (Normal Mode)

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next / previous main section |
| `j` / `k` | Move down / up within section |
| `Space` | Cycle a select field |
| `i` | Enter insert mode |
| `:` | Enter command mode |
| `Ctrl+S` | Save manifest |
| `Shift+M` | Open Provider Menu modal |
| `Ctrl+C` | Quit |

#### Command Mode

| Command | Action |
|---------|--------|
| `:w` / `:write` | Save |
| `:q` / `:quit` | Quit without save |
| `:wq` / `:x` | Save and quit |
| `:tabn` / `:bn` | Next section |
| `:tabp` / `:bp` | Previous section |
| `:1`–`:8` | Jump to section N |

#### Sub-Editor (varies by tab)

| Key | Action |
|-----|--------|
| `a` | Add item (list view) |
| `d` | Delete item (list view) |
| `Enter` / `i` | Edit selected item |
| `h` / `l` | Switch sub-tab |
| `b` / `Esc` | Back to parent / exit insert |
| `F` | Drill into nested fields (DTOs) |
| `A` | Drill into attributes (Domains) |

## Architecture Diagram Overview
`P` Open architecture diagram overview

<img width="1900" height="1138" alt="image" src="https://github.com/user-attachments/assets/333e5fac-cf88-4490-918a-ddba848df91d" />



### Sections Overview

Sections can be filled in **any order**. The dependency graph for downstream code generation is:

```
Description → Data → Backend → Contracts → Frontend → Infrastructure → Cross-Cutting → Realize
```

| # | Section | Sub-tabs |
|---|---------|---------|
| 0 | **Description** | Free-text project overview |
| 1 | **Backend** | Env · Services · Stack Config · Communication · Messaging · API Gateway · Jobs · Security · Auth |
| 2 | **Data** | Databases · Domains · Caching · File Storage |
| 3 | **Contracts** | DTOs · Endpoints · API Versioning · External APIs |
| 4 | **Frontend** | Tech · Theme · Pages · Navigation · i18n · A11y/SEO |
| 5 | **Infrastructure** | Networking · CI/CD · Observability |
| 6 | **Cross-Cutting** | Testing · Docs |
| 7 | **Realize** | Code generation configuration |

#### Section 0 — Description

Free-text textarea for describing the project in natural language before filling in structured pillars.

#### Section 1 — Backend

- **Env**: Architecture pattern (Monolith / Modular Monolith / Microservices / Event-Driven / Hybrid) — conditionally shows/hides sub-tabs
- **Services**: Name, responsibility, language, framework (dynamically filtered by language), pattern tag
- **Stack Config**: Reusable language/framework combinations for multi-language setups
- **Communication**: Service-to-service links with protocol, direction, trigger, sync/async, resilience patterns
- **Messaging**: Broker config and repeatable event catalog
- **API Gateway**: Technology, routing rules, features
- **Jobs**: Background job queues and cron job definitions
- **Security**: WAF config, CORS settings, session management
- **Auth**: Strategy, identity provider, role-based access control (RBAC with inheritance), permission definitions, authorization model, token storage, MFA

#### Section 2 — Data

- **Databases**: Alias, category, technology (filtered by category), hosting, HA mode — with type-conditional fields (SSL mode, eviction policy, replication factor, etc.)
- **Domains**: Bounded contexts with repeatable attributes (name, type, constraints, default, sensitive, validation) and relationships (type, FK field, cascade)
- **Caching**: Cache layer configuration
- **File Storage**: Object/file storage config

#### Section 3 — Contracts

- **DTOs**: Name, category (Request/Response/Event Payload/Shared), source domain, serialization protocol (REST/JSON, Protobuf, Avro, MessagePack, Thrift, FlatBuffers, Cap'n Proto) with protocol-specific fields
- **Endpoints**: Service unit, name/path, protocol (REST/GraphQL/gRPC/WebSocket/Event), auth roles (multi-select from backend roles), request/response DTOs — with protocol-conditional fields
- **API Versioning**: Strategy (URL path, header, query param, none), current version, deprecation policy
- **External APIs**: Third-party integrations with protocol-specific auth and failure strategy configuration

#### Section 4 — Frontend

- **Tech**: Language, platform, framework (filtered by language+platform), meta-framework, package manager, styling, component library, state management, data fetching, form handling, PWA, realtime, auth flow, bundle optimization, testing, linting
- **Theme**: Dark mode strategy, border radius, spacing, elevation, motion, vibe, colors
- **Pages**: Route, auth_required, layout, loading strategy, error handling, auth roles (multi-select), component actions (12+ action types: Fetch, Submit, Download, Upload, Delete, Refresh, Export, Navigate, Toast, State, Custom)
- **Navigation**: Nav type, breadcrumbs, auth-aware navigation
- **i18n**: Internationalization config
- **A11y/SEO**: Accessibility and SEO configuration
- **Assets**: Design assets (images, icons, fonts, videos, mockups) with usage classification (project or inspiration)

#### Section 5 — Infrastructure

- **Networking**: DNS, TLS, reverse proxy, CDN
- **CI/CD**: Platform, container registry, deploy strategy, IaC tool, secrets management
- **Observability**: Logging, metrics, tracing, error tracking, health checks, alerting
- Server environments: Named environments with compute/cloud/orchestrator settings

#### Section 6 — Cross-Cutting

- **Testing**: Framework selections dynamically filtered by backend languages and frontend tech (unit, integration, E2E, API, load, contract)
- **Docs**: API doc format, auto-generation toggle, changelog strategy

#### Section 7 — Realize

Code generation configuration: app name, output directory, global model, concurrency, verify toggle, dry run, per-section model overrides, and provider assignments.

---

## manifest.json Reference

Saved on `:w` or `Ctrl+S`. Unconfigured pillars are omitted automatically.

```json
{
  "created_at": "2026-01-01T00:00:00Z",
  "description": "Free-text project description",

  "backend": {
    "arch_pattern": "Microservices",
    "services": [],
    "stack_configs": [],
    "auth": { "strategy": "JWT", "roles": [] },
    "waf": {},
    "job_queues": [],
    "cron_jobs": []
  },

  "data": {
    "databases": [],
    "domains": [],
    "cachings": [],
    "file_storages": []
  },

  "contracts": {
    "dtos": [],
    "endpoints": [],
    "versioning": {},
    "external_apis": []
  },

  "frontend": {
    "tech": {},
    "theme": {},
    "pages": [],
    "navigation": {},
    "i18n": {},
    "a11y_seo": {},
    "assets": []
  },

  "infrastructure": {
    "networking": {},
    "cicd": {},
    "observability": {},
    "environments": []
  },

  "cross_cutting": {
    "testing": {},
    "docs": {}
  },

  "realize": {
    "app_name": "my-app",
    "output_dir": "output",
    "model": "claude-sonnet-4-6",
    "concurrency": 4,
    "verify": true,
    "dry_run": false,
    "section_models": {
      "backend": "Claude · Sonnet",
      "data": "Claude · Sonnet",
      "contracts": "Claude · Haiku",
      "frontend": "Claude · Sonnet",
      "infra": "Claude · Haiku",
      "crosscut": "Claude · Haiku"
    }
  },

  "configured_providers": {}
}
```

### `realize` Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `app_name` | string | — | Application name for generated code |
| `output_dir` | string | `output` | Directory where generated files are written |
| `model` | string | `claude-sonnet-4-6` | Global LLM model for code generation |
| `concurrency` | int | `1` | Max parallel tasks during code generation |
| `verify` | bool | `true` | Run language verifiers (build, vet, tsc) after generation |
| `dry_run` | bool | `false` | Print task plan without calling any LLM agents |
| `section_models` | object | — | Per-pillar model override in `"Provider · Tier"` format |

---

## Provider Configuration

Open the **Provider Menu** with `Shift+M` to configure LLM providers interactively.

Supported providers and their model tiers:

| Provider | Fast | Medium | Slow |
|----------|------|--------|------|
| **Claude** | Haiku (`claude-haiku-4-5-20251001`) | Sonnet (`claude-sonnet-4-6`) | Opus (`claude-opus-4-6`) |
| **ChatGPT** | o3-mini | 4o (`gpt-4o`) | o1 |
| **Gemini** | Flash (`gemini-2.0-flash`) | Pro (`gemini-2.0-pro-exp`) | Ultra (`gemini-ultra`) |
| **Mistral** | Nemo (`open-mistral-nemo`) | Small (`mistral-small-2409`) | Large (`mistral-large-2411`) |
| **Llama** | 8B (`llama-3.2-8b-preview`) | 70B (`llama-3.3-70b-versatile`) | 405B (`llama-3.1-405b-reasoning`) |

Authentication is configured per provider via API key or OAuth 2.0 PKCE flow. Credentials are stored in `manifest.json` under `configured_providers`.

Per-section overrides in `section_models` use the format `"Provider · Tier"` (e.g. `"Claude · Sonnet"`). Sections without an override inherit the global model selection.

**Environment variable fallback:**

| Provider | Environment Variable |
|----------|---------------------|
| Claude | `ANTHROPIC_API_KEY` |
| ChatGPT | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Llama (Groq) | `GROQ_API_KEY` |

---

## Code Generation (`realize`)

The `realize` binary reads `manifest.json` and drives a parallel, agentic code-generation pipeline.

```
manifest.json
    ↓
DAG construction     → execution graph with dependency edges
    ↓
Orchestrator         → parallel task dispatch (bounded by --parallel)
    ↓  (per task)
Runner               → deterministic fixes → agent call → verify → retry (escalating tier)
    ↓
Shared memory        → stores completed outputs; downstream agents read upstream signatures
    ↓
File writer          → writes generated files to --output directory
```

### CLI Flags

```
--manifest   path to manifest.json       (default: manifest.json)
--output     output directory            (default: output)
--skills     skills directory            (default: .vibemenu/skills)
--retries    max retry attempts per task (default: 3)
--parallel   max concurrent tasks        (default: 1)
--dry-run    print task plan, no agents
--verbose    print token usage + logs
```

### Model Tiering

Tasks are automatically assigned a model tier based on complexity. On verification failure, the engine escalates to the next tier for the retry:

| Tier | Task Kinds | Claude / OpenAI / Gemini |
|------|-----------|--------------------------|
| **Fast** | Contracts, docs, Docker, CI/CD | Haiku 4.5 / o3-mini / Flash |
| **Medium** | Services, auth, data, frontend, Terraform, testing | Sonnet 4.6 / 4o / Pro |
| **Slow** | Escalation fallback | Opus 4.6 / o1 / Ultra |

### Verification

After each task, generated code is checked by a language-specific verifier. Failed tasks apply deterministic fixes (formatting, unused imports) before retrying with an escalated model tier.

| Language | Check |
|----------|-------|
| Go | `go build` + `go vet` |
| TypeScript | `tsc --noEmit` |
| Python | `python -m py_compile` |
| Terraform | `terraform validate` |

---

## Skills System

Skills are markdown files that inject domain-specific guidance into agent prompts.

**Location:** `.vibemenu/skills/` (override with `--skills`)

Each `.md` file defines a named skill. Technology aliases automatically map framework names (e.g. `nextjs`) to the relevant skill file. Universal skills apply to all tasks of a given kind regardless of tech stack.

```
.vibemenu/
└── skills/
    ├── nextjs.md
    ├── postgres.md
    └── terraform-aws.md
```
