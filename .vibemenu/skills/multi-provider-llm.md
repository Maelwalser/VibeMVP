---
name: multi-provider-llm
description: Multi-provider LLM patterns — portable prompts, cost-aware routing, JSON structured output, fallback chains, token budgets, rate limiting, streaming, and provider feature matrix.
origin: vibemenu
---

# Multi-Provider LLM Patterns

Production patterns for systems that use Claude, OpenAI GPT, Gemini, Mistral, and Llama. Covers prompt portability, cost-aware routing, fallback chains, and token management.

## When to Activate

- `realize` pipeline in VibeMenu (all agent tasks)
- Building AI features that need to work across multiple LLM providers
- Implementing fallback or cost-optimization routing between models
- Generating structured JSON output that must be validated

---

## Portable System Prompt Patterns

Design prompts that work correctly across Claude, GPT, Gemini, and Mistral without model-specific behavior cues.

**Structure (always follow this order):**
```
1. Role      — who the model is
2. Context   — what it knows
3. Task      — what to do
4. Output    — exact format required
5. Constraints — what not to do
```

```go
// agent/prompt.go — portable system prompt template
const systemPromptTemplate = `You are a {{.Role}} specialized in {{.Domain}}.

Context:
{{.Context}}

Task:
{{.Task}}

Output format:
Return ONLY valid JSON matching this schema — no markdown wrapper, no explanation:
{{.Schema}}

Constraints:
- Do not include any text outside the JSON object
- Do not add comments inside the JSON
- If a field is unknown, use null rather than omitting it
- Return an error object {"error": "reason"} if you cannot complete the task`
```

**Avoid these Claude-specific constructs in shared prompts:**
```
# BAD — only works in Claude
<thinking>...</thinking>
<tool_use>...</tool_use>

# BAD — GPT-specific
{"role": "system", "content": "You are..."}  // format, not content issue

# GOOD — neutral, works everywhere
"You are a Go backend engineer. Return only valid JSON."
```

**Test on all target models before shipping.** Models interpret the same instruction differently — always validate output format compliance on each model.

---

## Cost-Aware Model Routing

Route tasks to the cheapest model that can reliably handle the complexity. Escalate only on failure.

```go
// orchestrator/tier.go — default tier assignment per task kind
type ModelTier int

const (
    TierFast   ModelTier = iota // Haiku / Flash / o3-mini — cheapest
    TierMedium                  // Sonnet / Pro / 4o — balanced
    TierSlow                    // Opus / Ultra / o1 — most capable, most expensive
)

var defaultTierForKind = map[dag.TaskKind]ModelTier{
    // Fast tier: simple, low-creativity generation
    dag.TaskContracts:   TierFast,
    dag.TaskDocs:        TierFast,
    dag.TaskDockerfile:  TierFast,
    dag.TaskCICD:        TierFast,

    // Medium tier: requires reasoning, architectural decisions
    dag.TaskService:     TierMedium,
    dag.TaskAuth:        TierMedium,
    dag.TaskData:        TierMedium,
    dag.TaskFrontend:    TierMedium,
    dag.TaskTerraform:   TierMedium,
    dag.TaskTesting:     TierMedium,

    // Slow tier: escalation fallback only — never use as default
    // dag.TaskEscalation: TierSlow,
}
```

```go
// escalateModel returns the next tier up on verification failure.
// Called in runner.go after a failed verification attempt.
func escalateModel(current ModelTier) (ModelTier, bool) {
    switch current {
    case TierFast:
        return TierMedium, true
    case TierMedium:
        return TierSlow, true
    default:
        return TierSlow, false // already at max
    }
}
```

**Cost log per task** — emit structured logs for cost attribution:
```go
slog.Info("agent call",
    "task_id", taskID,
    "task_kind", kind,
    "provider", provider,
    "model", model,
    "tier", tier,
    "input_tokens", usage.InputTokens,
    "output_tokens", usage.OutputTokens,
    "attempt", attempt,
)
```

---

## Structured Output — JSON Mode

Each provider handles structured output differently. Always validate the result against your schema regardless of the mode used.

### Claude (Anthropic SDK)

No native JSON mode. Use system prompt instructions + validation.

```go
// agent/agent.go
resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
    Model:     anthropic.F(anthropic.ModelClaude4Sonnet20251101),
    MaxTokens: anthropic.F(int64(4096)),
    System: anthropic.F([]anthropic.TextBlockParam{
        {Type: anthropic.F(anthropic.TextBlockParamTypeText),
         Text: anthropic.F(systemPrompt)},
    }),
    Messages: anthropic.F([]anthropic.MessageParam{
        anthropic.UserMessageParam(anthropic.NewTextBlock(userPrompt)),
    }),
})

raw := resp.Content[0].Text
// Validate against expected schema
var result MySchema
if err := json.Unmarshal([]byte(raw), &result); err != nil {
    return nil, fmt.Errorf("claude returned invalid JSON: %w\nraw: %s", err, raw)
}
```

### OpenAI (GPT-4o)

Use `response_format` JSON mode — still validate after.

```go
resp, err := openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model: openai.F(openai.ChatModelGPT4o),
    ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
        openai.ResponseFormatJSONObjectParam{Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject)},
    ),
    Messages: openai.F(messages),
    MaxTokens: openai.F(int64(4096)),
})
raw := resp.Choices[0].Message.Content
// Always validate — JSON mode doesn't guarantee schema compliance
```

### Gemini

Use `responseSchema` for the strongest structured output enforcement.

```go
resp, err := geminiModel.GenerateContent(ctx, genai.Text(prompt))
// Or with schema enforcement:
geminiModel.GenerationConfig.ResponseMIMEType = "application/json"
geminiModel.GenerationConfig.ResponseSchema = &genai.Schema{
    Type: genai.TypeObject,
    Properties: map[string]*genai.Schema{
        "service_name": {Type: genai.TypeString},
        "endpoints":    {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeObject}},
    },
    Required: []string{"service_name"},
}
```

**Ground truth is schema validation — always run it regardless of provider mode:**
```go
func validateJSON[T any](raw string) (T, error) {
    var result T
    if err := json.Unmarshal([]byte(raw), &result); err != nil {
        return result, fmt.Errorf("JSON parse failed: %w\nraw output:\n%s", err, raw)
    }
    // Optionally run deeper validation (required fields, enum values, etc.)
    if err := validate(result); err != nil {
        return result, fmt.Errorf("schema validation failed: %w", err)
    }
    return result, nil
}
```

---

## Fallback Chain Implementation

Try providers in order; move to next on error or failed validation.

```go
// orchestrator/runner.go
type AgentProvider interface {
    Name() string
    Call(ctx context.Context, prompt AgentPrompt) (string, error)
}

func callWithFallback(
    ctx context.Context,
    providers []AgentProvider,
    prompt AgentPrompt,
    verify func(string) error,
) (string, error) {
    var lastErr error

    for attempt, provider := range providers {
        result, err := provider.Call(ctx, prompt)
        if err != nil {
            slog.Warn("provider call failed",
                "provider", provider.Name(),
                "attempt", attempt,
                "error", err,
            )
            lastErr = fmt.Errorf("provider %s: %w", provider.Name(), err)
            continue
        }

        if verifyErr := verify(result); verifyErr != nil {
            slog.Warn("provider output failed verification",
                "provider", provider.Name(),
                "attempt", attempt,
                "error", verifyErr,
            )
            lastErr = fmt.Errorf("provider %s verification: %w", provider.Name(), verifyErr)
            continue
        }

        slog.Info("provider succeeded", "provider", provider.Name(), "attempt", attempt)
        return result, nil
    }

    return "", fmt.Errorf("all providers exhausted: %w", lastErr)
}
```

**Provider chain composition:**
```go
// Build the fallback chain from provider assignments in the manifest
func buildProviders(cfg ProviderConfig) []AgentProvider {
    providers := []AgentProvider{
        newProvider(cfg.Primary),    // e.g., ClaudeAgent (Sonnet)
        newProvider(cfg.Secondary),  // e.g., GeminiAgent (Pro)
        newProvider(cfg.Fallback),   // e.g., ClaudeAgent (Opus)
    }
    // Remove nils (unconfigured providers)
    result := providers[:0]
    for _, p := range providers {
        if p != nil {
            result = append(result, p)
        }
    }
    return result
}
```

---

## Token Budget Awareness

Context window limits differ significantly. Trim context to fit.

| Model | Context Window | Recommendation |
|-------|---------------|----------------|
| Claude Sonnet/Opus | 200k tokens | Can include full upstream signatures |
| GPT-4o | 128k tokens | Include `SharedMemory` signatures only, not full files |
| Gemini 1.5 Pro | 1M tokens | Large context available; watch latency on > 200k |
| Mistral Large | 128k tokens | Same as GPT-4o |
| Llama 3 70B | 128k tokens | Same as GPT-4o |

```go
// agent/prompt.go — context trimming for smaller context windows
const (
    MaxContextTokensLarge  = 180_000 // Claude, Gemini — use full memory
    MaxContextTokensMedium = 100_000 // GPT-4o, Mistral — trim aggressively
    MaxContextTokensSmall  = 60_000  // Llama via Groq — summary only
)

func buildPromptContext(memory *SharedMemory, provider string) string {
    limit := contextLimitForProvider(provider)
    // Estimate: ~4 chars per token
    maxChars := limit * 4

    var b strings.Builder
    for _, sig := range memory.Signatures() {
        entry := formatSignature(sig)
        if b.Len()+len(entry) > maxChars {
            b.WriteString("\n[context truncated — remaining signatures omitted]\n")
            break
        }
        b.WriteString(entry)
    }
    return b.String()
}

func contextLimitForProvider(provider string) int {
    switch provider {
    case "claude":
        return MaxContextTokensLarge
    case "gemini":
        return MaxContextTokensLarge
    default: // openai, mistral, llama
        return MaxContextTokensMedium
    }
}
```

**Always set `max_tokens` explicitly.** A runaway generation on Opus can cost 10-50x a typical call.

---

## Temperature for Code Generation

Always use `temperature: 0` for code generation. Deterministic output is critical for reproducible builds and testability.

```go
// Claude
anthropic.MessageNewParams{
    Temperature: anthropic.F(0.0),
    // ...
}

// OpenAI
openai.ChatCompletionNewParams{
    Temperature: openai.F(0.0),
    // ...
}

// Gemini
model.GenerationConfig.Temperature = 0.0
```

**Exception:** For creative tasks (UI copy, descriptions, documentation) use `temperature: 0.3–0.7` to get less repetitive output.

---

## Rate Limiting and Retry

Each provider has different rate limits. Implement per-provider backoff.

```go
// agent/ratelimit.go
import "golang.org/x/time/rate"

// Per-provider rate limiters — adjust to your tier's limits
var providerLimiters = map[string]*rate.Limiter{
    "claude":  rate.NewLimiter(rate.Every(60*time.Second/500), 10),  // 500 RPM
    "openai":  rate.NewLimiter(rate.Every(60*time.Second/500), 10),  // 500 RPM
    "gemini":  rate.NewLimiter(rate.Every(60*time.Second/60), 5),    // 60 RPM
    "mistral": rate.NewLimiter(rate.Every(60*time.Second/100), 5),
}

func callWithRateLimit(ctx context.Context, provider string, call func() (string, error)) (string, error) {
    limiter := providerLimiters[provider]
    if limiter != nil {
        if err := limiter.Wait(ctx); err != nil {
            return "", fmt.Errorf("rate limiter wait: %w", err)
        }
    }

    for attempt := 0; attempt < 5; attempt++ {
        result, err := call()
        if err == nil {
            return result, nil
        }

        // Check for rate limit or server error
        if isRetryable(err) {
            delay := backoffDelay(attempt, err)
            slog.Warn("retryable error, backing off",
                "provider", provider, "attempt", attempt, "delay", delay, "error", err)
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return "", ctx.Err()
            }
            continue
        }
        return "", err // non-retryable error
    }
    return "", fmt.Errorf("max retries exceeded for provider %s", provider)
}

func isRetryable(err error) bool {
    msg := err.Error()
    return strings.Contains(msg, "429") ||
        strings.Contains(msg, "503") ||
        strings.Contains(msg, "rate limit") ||
        strings.Contains(msg, "overloaded")
}

func backoffDelay(attempt int, err error) time.Duration {
    // Respect Retry-After header if present in error (some SDKs expose it)
    base := time.Duration(1<<attempt) * time.Second
    if base > 60*time.Second {
        base = 60 * time.Second
    }
    return base + time.Duration(rand.Int63n(int64(base/2)))
}
```

---

## Streaming vs Non-Streaming

```go
// Use streaming for outputs > 2k tokens — reduces perceived latency
// Use non-streaming for short structured JSON — easier to parse atomically

func (a *ClaudeAgent) Call(ctx context.Context, prompt AgentPrompt) (string, error) {
    estimatedOutputTokens := estimateOutputSize(prompt)

    if estimatedOutputTokens > 2000 {
        return a.callStreaming(ctx, prompt)
    }
    return a.callBlocking(ctx, prompt)
}

func (a *ClaudeAgent) callStreaming(ctx context.Context, prompt AgentPrompt) (string, error) {
    stream := a.client.Messages.NewStreaming(ctx, buildParams(prompt))
    var buf strings.Builder
    for stream.Next() {
        event := stream.Current()
        if delta, ok := event.Delta.(anthropic.ContentBlockDeltaEventDelta); ok {
            if delta.Type == anthropic.ContentBlockDeltaEventDeltaTypeTextDelta {
                buf.WriteString(delta.Text)
            }
        }
    }
    if err := stream.Err(); err != nil {
        return "", fmt.Errorf("streaming error: %w", err)
    }
    return buf.String(), nil
}
```

---

## Provider Feature Matrix

Document capability differences in your codebase — don't discover them at runtime.

| Feature | Claude (all) | GPT-4o | Gemini Pro | Mistral Large | Llama 3 70B |
|---------|-------------|--------|------------|---------------|-------------|
| JSON mode | Prompt only | `response_format` | `responseMimeType` | Prompt only | Prompt only |
| Function/tool calling | Yes | Yes | Yes | Yes | Limited |
| Vision/image input | Yes | Yes | Yes | No | No |
| Streaming | Yes | Yes | Yes | Yes | Yes |
| System prompt | Yes | Yes | Yes | Yes | Yes |
| Context window | 200k | 128k | 1M | 128k | 128k |
| Embeddings | Via Voyage AI | text-embedding-3 | text-embedding-004 | mistral-embed | No |

```go
// config/provider_features.go — checked at startup to prevent runtime surprises
type ProviderCapabilities struct {
    SupportsVision    bool
    SupportsTools     bool
    SupportsJSONMode  bool
    MaxContextTokens  int
    MaxOutputTokens   int
}

var providerCapabilities = map[string]ProviderCapabilities{
    "claude-haiku-4-5":     {SupportsVision: true, SupportsTools: true, MaxContextTokens: 200_000, MaxOutputTokens: 8192},
    "claude-sonnet-4-6":    {SupportsVision: true, SupportsTools: true, MaxContextTokens: 200_000, MaxOutputTokens: 8192},
    "gpt-4o":               {SupportsVision: true, SupportsTools: true, SupportsJSONMode: true, MaxContextTokens: 128_000, MaxOutputTokens: 16384},
    "gemini-1.5-pro":       {SupportsVision: true, SupportsTools: true, SupportsJSONMode: true, MaxContextTokens: 1_000_000, MaxOutputTokens: 8192},
    "mistral-large-latest": {SupportsTools: true, MaxContextTokens: 128_000, MaxOutputTokens: 4096},
}

func validateProviderForTask(provider string, task dag.TaskKind) error {
    caps, ok := providerCapabilities[provider]
    if !ok {
        return fmt.Errorf("unknown provider model: %s", provider)
    }
    if taskRequiresVision(task) && !caps.SupportsVision {
        return fmt.Errorf("provider %s does not support vision, required for task %s", provider, task)
    }
    return nil
}
```

---

## Anti-Patterns to Avoid

- **Hardcoding model IDs as string literals**: Use named constants. Model versions change and grep-replace is error-prone.
- **`temperature: 1.0` for code generation**: Non-deterministic code is impossible to test or reproduce.
- **Trusting JSON mode without schema validation**: Both GPT-4o and Gemini can return valid JSON that doesn't match your expected schema.
- **No `max_tokens` limit**: A single misbehaving prompt can generate 100k+ tokens and burn your budget.
- **Sharing a single HTTP client across providers**: Rate limit errors from one provider should not slow another. Use separate HTTP clients with independent timeouts.
- **Logging full prompt and response at INFO level**: System prompts and generated code can contain secrets or PII. Log at DEBUG level behind a `--verbose` flag.
- **Ignoring `Retry-After` headers**: Most providers return `Retry-After` on 429 — respecting it is faster and cheaper than binary exponential backoff.
