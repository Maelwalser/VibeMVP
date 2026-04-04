package orchestrator

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/dag"
)

// modelSpec holds the version-specific and default model IDs for one provider tier.
type modelSpec struct {
	// byVersion maps specific version strings to model IDs.
	byVersion map[string]string
	// fallback is used when version is empty or not found in byVersion.
	fallback string
}

// providerModels maps provider name → tier → modelSpec.
// Add new providers or tiers here without touching any other file.
var providerModels = map[string]map[string]modelSpec{
	"Claude": {
		"Haiku":  {fallback: "claude-haiku-4-5-20251001"},
		"Sonnet": {fallback: "claude-sonnet-4-6"},
		"Opus":   {fallback: "claude-opus-4-6"},
	},
	"ChatGPT": {
		"Mini": {byVersion: map[string]string{"o3-mini": "o3-mini"}, fallback: "gpt-4o-mini"},
		"4o":   {byVersion: map[string]string{"4o-2024": "gpt-4o-2024-11-20"}, fallback: "gpt-4o"},
		"o1":   {byVersion: map[string]string{"o1-preview": "o1-preview"}, fallback: "o1"},
	},
	"Gemini": {
		"Flash": {byVersion: map[string]string{"1.5": "gemini-1.5-flash"}, fallback: "gemini-2.0-flash"},
		"Pro":   {byVersion: map[string]string{"1.5": "gemini-1.5-pro"}, fallback: "gemini-2.0-pro-exp"},
		"Ultra": {fallback: "gemini-ultra"},
	},
	"Mistral": {
		"Nemo":  {fallback: "open-mistral-nemo"},
		"Small": {byVersion: map[string]string{"3.0": "mistral-small-2402"}, fallback: "mistral-small-2409"},
		"Large": {byVersion: map[string]string{"2.0": "mistral-large-2407"}, fallback: "mistral-large-2411"},
	},
	"Llama": {
		"8B":   {byVersion: map[string]string{"3.1": "llama-3.1-8b-instant"}, fallback: "llama-3.2-8b-preview"},
		"70B":  {byVersion: map[string]string{"3.1": "llama-3.1-70b-versatile"}, fallback: "llama-3.3-70b-versatile"},
		"405B": {fallback: "llama-3.1-405b-reasoning"},
	},
}

// providerDefaults maps each provider to its default model ID when no tier is matched.
var providerDefaults = map[string]string{
	"Claude":  defaultModel,
	"ChatGPT": "gpt-4o",
	"Gemini":  "gemini-2.0-flash",
	"Mistral": "mistral-large-2411",
	"Llama":   "llama-3.3-70b-versatile",
}

// providerTierNames maps each provider to [Fast, Medium, Slow] tier name strings.
// These names are the keys used in providerModels for that provider.
// Order must match ModelTier constants: TierFast=0, TierMedium=1, TierSlow=2.
var providerTierNames = map[string][3]string{
	"Claude":  {"Haiku", "Sonnet", "Opus"},
	"ChatGPT": {"Mini", "4o", "o1"},
	"Gemini":  {"Flash", "Pro", "Ultra"},
	"Mistral": {"Nemo", "Small", "Large"},
	"Llama":   {"8B", "70B", "405B"},
}

// resolveModelID returns the model ID string for a given provider, tier name, and version.
// Falls back to the provider default when the tier or version is not found.
func resolveModelID(provider, tier, version string) string {
	tiers, ok := providerModels[provider]
	if !ok {
		if d, ok := providerDefaults[provider]; ok {
			return d
		}
		return defaultModel
	}
	spec, ok := tiers[tier]
	if !ok {
		if d, ok := providerDefaults[provider]; ok {
			return d
		}
		return defaultModel
	}
	if id, ok := spec.byVersion[version]; ok {
		return id
	}
	return spec.fallback
}

// resolveModelIDForTier returns the model ID for a given provider, abstract ModelTier,
// and optional version string. Translates the abstract tier to a provider-specific tier
// name using providerTierNames before looking up in providerModels.
func resolveModelIDForTier(provider string, tier ModelTier, version string) string {
	names, ok := providerTierNames[provider]
	if !ok {
		return resolveModelID(provider, "", version)
	}
	return resolveModelID(provider, names[tier], version)
}

// buildAgentForTier constructs the appropriate agent for the given ProviderAssignment
// and ModelTier. For Claude with no Credential, falls back to the env-var API key.
// Add new providers to the switch without touching any other file.
func buildAgentForTier(pa manifest.ProviderAssignment, tier ModelTier, maxTokens int64, verbose bool) agent.Agent {
	model := resolveModelIDForTier(pa.Provider, tier, pa.Version)
	switch pa.Provider {
	case "Claude":
		if pa.Credential != "" {
			return agent.NewClaudeAgentWithKey(model, maxTokens, verbose, pa.Credential)
		}
		return agent.NewClaudeAgent(model, maxTokens, verbose)
	case "ChatGPT":
		return agent.NewOpenAIAgent("https://api.openai.com", pa.Credential, model, maxTokens, verbose)
	case "Mistral":
		return agent.NewOpenAIAgent("https://api.mistral.ai", pa.Credential, model, maxTokens, verbose)
	case "Llama":
		return agent.NewOpenAIAgent("https://api.groq.com/openai", pa.Credential, model, maxTokens, verbose)
	case "Gemini":
		return agent.NewGeminiAgent(pa.Credential, model, maxTokens, verbose)
	default:
		return agent.NewClaudeAgent(model, maxTokens, verbose)
	}
}

// ── Provider selection ────────────────────────────────────────────────────────

// resolveAgent returns a task-specific agent if the manifest has a provider
// assignment for the task's section, otherwise returns the default agent.
func resolveAgent(taskID string, providers manifest.ProviderAssignments, def agent.Agent, verbose bool) agent.Agent {
	pa, ok := providerFor(taskID, providers)
	if !ok || pa.Credential == "" {
		return def
	}
	model := resolveModelID(pa.Provider, pa.Model, pa.Version)
	switch pa.Provider {
	case "Claude":
		return agent.NewClaudeAgentWithKey(model, defaultMaxTokens, verbose, pa.Credential)
	case "ChatGPT":
		return agent.NewOpenAIAgent("https://api.openai.com", pa.Credential, model, defaultMaxTokens, verbose)
	case "Gemini":
		return agent.NewGeminiAgent(pa.Credential, model, defaultMaxTokens, verbose)
	case "Mistral":
		return agent.NewOpenAIAgent("https://api.mistral.ai", pa.Credential, model, defaultMaxTokens, verbose)
	case "Llama":
		return agent.NewOpenAIAgent("https://api.groq.com/openai", pa.Credential, model, defaultMaxTokens, verbose)
	default:
		return def
	}
}

// providerFor returns the ProviderAssignment for the section that owns taskID.
// Task IDs follow "<section>.<name>" or just "<section>".
func providerFor(taskID string, providers manifest.ProviderAssignments) (manifest.ProviderAssignment, bool) {
	if providers == nil {
		return manifest.ProviderAssignment{}, false
	}
	sectionID := taskID
	if dot := strings.Index(taskID, "."); dot >= 0 {
		sectionID = taskID[:dot]
	}
	pa, ok := providers[sectionID]
	return pa, ok
}

// describeProvider returns a human-readable model label for dry-run output.
// For manifest-configured providers it shows the provider name/tier; for
// default-agent tasks it shows the model ID resolved from defaultPA.
func describeProvider(taskID string, providers manifest.ProviderAssignments, kind dag.TaskKind, defaultPA manifest.ProviderAssignment) string {
	if kind == dag.TaskKindDependencyResolution {
		return "(package manager — no LLM)"
	}
	pa, ok := providerFor(taskID, providers)
	if !ok || pa.Credential == "" {
		return resolveModelIDForTier(defaultPA.Provider, tierForKind(kind), defaultPA.Version)
	}
	s := pa.Provider
	if pa.Model != "" {
		s += " " + pa.Model
	}
	if pa.Version != "" {
		s += " " + pa.Version
	}
	return s
}

// buildProviderAssignments constructs a per-section ProviderAssignments map from the
// manifest's ConfiguredProviders registry and per-section SectionModels overrides.
//
// SectionModels values are formatted as "Provider · Tier" (e.g. "Claude · Sonnet").
// Sections with no override or "default" are omitted; the orchestrator falls back to
// its default agent for those.
func buildProviderAssignments(m *manifest.Manifest) manifest.ProviderAssignments {
	if len(m.ConfiguredProviders) == 0 {
		return nil
	}

	sections := []string{"backend", "data", "contracts", "frontend", "infra", "crosscut"}
	result := make(manifest.ProviderAssignments)

	for _, section := range sections {
		sectionModel, ok := m.Realize.SectionModels[section]
		if !ok || sectionModel == "" || sectionModel == "default" {
			continue
		}

		// Parse "Provider · Tier" format.
		parts := strings.SplitN(sectionModel, " · ", 2)
		if len(parts) != 2 {
			continue
		}
		provLabel, tier := parts[0], parts[1]

		pa, exists := m.ConfiguredProviders[provLabel]
		if !exists || pa.Credential == "" {
			continue
		}

		// Use the configured provider's credentials with the specified tier.
		pa.Model = tier
		pa.Version = "" // use the fallback version for that tier
		result[section] = pa
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
