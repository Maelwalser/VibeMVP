# VibeMenu
                                                                            
What vibe is on the menu today?

A TUI for declaratively specifying a complete software system architecture. Define your stack across 8 structured sections, then generate a `manifest.json` for downstream code generation via the `realize` pipeline.

(Still in development, may have issues; TODO Final task in flow needs fixing)

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [The TUI Editor](#the-tui-editor)
  - [Key Bindings](#key-bindings)
  - [Sections Overview](#sections-overview)
- [manifest.json Reference](#manifestjson-reference)
- [Provider Configuration](#provider-configuration)
- [Code Generation (`realize`)](#code-generation-realize)
  - [CLI Flags](#cli-flags)
  - [Model Tiering](#model-tiering)
- [Skills System](#skills-system)

---

## Installation

```

---
```

---

## The TUI Editor

### Sections Overview

Sections can be filled in any order. The dependency graph for code generation is:

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
    "concurrency": 4,
    "verify": true,
    "dry_run": false,
    "provider": "Claude",
    "tier_fast": "claude-haiku-4-5-20251001",
    "tier_medium": "claude-sonnet-4-6",
    "tier_slow": "claude-opus-4-6",
    "section_models": {
      "backend": "Claude · Sonnet",
      "data": "Gemini · Flash"
    }
  },

  "configured_providers": {
    "Claude":  { "provider": "Claude",  "model": "Sonnet", "auth": "api_key", "credential": "..." },
    "Gemini":  { "provider": "Gemini",  "model": "Pro",    "auth": "oauth",   "credential": "..." }
  }
}
```

### `realize` Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `app_name` | string | — | Application name for generated code |
| `output_dir` | string | `output` | Directory where generated files are written |
| `concurrency` | int | `1` | Max parallel tasks during code generation |
| `verify` | bool | `true` | Run language verifiers (build, vet, tsc) after generation |
| `dry_run` | bool | `false` | Print task plan without calling any LLM agents |
| `provider` | string | — | Default LLM provider: `Claude`, `Gemini`, `ChatGPT`, `Mistral`, `Llama` |
| `tier_fast` | string | — | Model ID override for low-complexity tasks |
| `tier_medium` | string | — | Model ID override for medium-complexity tasks |
| `tier_slow` | string | — | Model ID override for escalation / high-complexity tasks |
| `section_models` | object | — | Per-pillar provider override in `"Provider · Tier"` format |

---

## Provider Configuration

Open the Provider Menu with `Shift+M` to configure LLM providers interactively.

Supported providers and their model tiers:

| Provider | Fast | Medium | Slow |
|----------|------|--------|------|
| **Claude** | Haiku (`claude-haiku-4-5-20251001`) | Sonnet (`claude-sonnet-4-6`) | Opus (`claude-opus-4-6`) |
| **ChatGPT** | Mini (`gpt-4o-mini`) | 4o (`gpt-4o`) | o1 (`o1`) |
| **Gemini** | Flash (`gemini-2.0-flash`) | Pro (`gemini-2.0-pro-exp`) | Ultra (`gemini-ultra`) |
| **Mistral** | Nemo (`open-mistral-nemo`) | Small (`mistral-small-2409`) | Large (`mistral-large-2411`) |
| **Llama** | 8B (`llama-3.2-8b-preview`) | 70B (`llama-3.3-70b-versatile`) | 405B (`llama-3.1-405b-reasoning`) |

Authentication is configured per provider via API key or OAuth 2.0 PKCE flow. Credentials are stored in `manifest.json` under `configured_providers`.

Per-section overrides in `section_models` use the format `"Provider · Tier"` (e.g. `"Claude · Sonnet"`). Sections without an override use the default provider and tier for that task kind.

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

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--manifest` | `manifest.json` | Path to the manifest file |
| `--output` | `output` | Directory for generated code |
| `--skills` | `.vibemenu/skills` | Directory for skill markdown files |
| `--retries` | `3` | Max verification retry attempts per task |
| `--parallel` | `1` | Max concurrent tasks |
| `--dry-run` | `false` | Print task plan without running agents |
| `--verbose` | `false` | Print token usage and thinking logs |
| `--provider` | — | Default LLM provider (overrides manifest) |
| `--api-key` | — | API key for the default provider (overrides env var) |

**Example:**

```bash
./realize \
  --manifest manifest.json \
  --output ./my-project \
  --parallel 4 \
  --provider Claude \
  --retries 5 \
  --verbose
```

### Model Tiering

Tasks are automatically assigned a model tier based on complexity. On verification failure, the engine escalates to the next tier for the retry:

| Tier | Task Kinds | Rationale |
|------|-----------|-----------|
| **Fast** | Contracts, docs, Docker, CI/CD | Straightforward generation |
| **Medium** | Services, auth, data, frontend, Terraform, testing | Moderate complexity |
| **Slow** | Escalation fallback | Verification failures, complex reasoning |

### Verification

After each task, the generated code is checked by a language-specific verifier:

| Language | Check |
|----------|-------|
| Go | `go build` + `go vet` |
| TypeScript | `tsc --noEmit` |
| Python | `python -m py_compile` |
| Terraform | `terraform validate` |

Failed tasks apply deterministic fixes (formatting, unused imports) before retrying with an escalated model tier.

---

## Skills System

Skills are markdown files that inject domain-specific guidance into agent prompts.

**Location:** `.vibemenu/skills/` (override with `--skills`)

Each `.md` file in the skills directory defines a named skill. Technology aliases in the engine automatically map framework names (e.g. `nextjs`) to the relevant skill file. Universal skills apply to all tasks of a given kind regardless of tech stack.

```
.vibemenu/
└── skills/
    ├── nextjs.md
    ├── postgres.md
    └── terraform-aws.md
```
